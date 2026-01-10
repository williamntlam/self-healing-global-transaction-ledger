package s3

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// mockS3API is a mock implementation of S3 API operations
type mockS3API struct {
	mock.Mock
}

func (m *mockS3API) HeadBucket(input *s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadBucketOutput), args.Error(1)
}

func (m *mockS3API) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.CreateBucketOutput), args.Error(1)
}

func (m *mockS3API) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

// newTestableClient creates a client with injectable S3 API (for testing)
func newTestableClient(s3Client s3API, bucket string, logger *zap.Logger) *Client {
	return &Client{
		s3Client: s3Client,
		bucket:   bucket,
		logger:   logger,
	}
}

func TestClient_WriteAuditLog_Success(t *testing.T) {
	mockAPI := new(mockS3API)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "test-bucket", logger)

	key := "transactions/test-key.json"
	content := []byte(`{"test": "data"}`)

	mockAPI.On("PutObject", mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Bucket == "test-bucket" && *input.Key == key
	})).Return(&s3.PutObjectOutput{}, nil)

	err := client.WriteAuditLog(key, content)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_WriteAuditLog_Error(t *testing.T) {
	mockAPI := new(mockS3API)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "test-bucket", logger)

	key := "transactions/test-key.json"
	content := []byte(`{"test": "data"}`)

	mockAPI.On("PutObject", mock.Anything).Return(nil, errors.New("S3 error"))

	err := client.WriteAuditLog(key, content)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "failed to write audit log: S3 error" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_WriteAuditLogWithTimestamp(t *testing.T) {
	mockAPI := new(mockS3API)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "test-bucket", logger)

	prefix := "transactions"
	content := []byte(`{"test": "data"}`)

	mockAPI.On("PutObject", mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Bucket == "test-bucket" && 
			len(*input.Key) > len(prefix) &&
			(*input.Key)[:len(prefix)] == prefix
	})).Return(&s3.PutObjectOutput{}, nil)

	err := client.WriteAuditLogWithTimestamp(prefix, content)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_Health_Success(t *testing.T) {
	mockAPI := new(mockS3API)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "test-bucket", logger)

	mockAPI.On("HeadBucket", mock.MatchedBy(func(input *s3.HeadBucketInput) bool {
		return *input.Bucket == "test-bucket"
	})).Return(&s3.HeadBucketOutput{}, nil)

	err := client.Health()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_Health_Error(t *testing.T) {
	mockAPI := new(mockS3API)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "test-bucket", logger)

	mockAPI.On("HeadBucket", mock.Anything).Return(nil, errors.New("bucket not found"))

	err := client.Health()
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "S3 health check failed: bucket not found" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestEnsureBucket_BucketExists(t *testing.T) {
	mockAPI := new(mockS3API)

	mockAPI.On("HeadBucket", mock.MatchedBy(func(input *s3.HeadBucketInput) bool {
		return *input.Bucket == "existing-bucket"
	})).Return(&s3.HeadBucketOutput{}, nil)

	err := ensureBucket(mockAPI, "existing-bucket")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
	mockAPI.AssertNotCalled(t, "CreateBucket")
}

func TestEnsureBucket_BucketDoesNotExist_CreateSuccess(t *testing.T) {
	mockAPI := new(mockS3API)

	// First HeadBucket fails (bucket doesn't exist)
	mockAPI.On("HeadBucket", mock.MatchedBy(func(input *s3.HeadBucketInput) bool {
		return *input.Bucket == "new-bucket"
	})).Return(nil, awserr.New("NotFound", "bucket not found", nil))

	// CreateBucket succeeds
	mockAPI.On("CreateBucket", mock.MatchedBy(func(input *s3.CreateBucketInput) bool {
		return *input.Bucket == "new-bucket"
	})).Return(&s3.CreateBucketOutput{}, nil)

	err := ensureBucket(mockAPI, "new-bucket")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestEnsureBucket_BucketDoesNotExist_CreateFailsButBucketExists(t *testing.T) {
	mockAPI := new(mockS3API)

	// First HeadBucket fails (bucket doesn't exist)
	mockAPI.On("HeadBucket", mock.MatchedBy(func(input *s3.HeadBucketInput) bool {
		return *input.Bucket == "new-bucket"
	})).Return(nil, awserr.New("NotFound", "bucket not found", nil)).Once()

	// CreateBucket fails (maybe race condition)
	mockAPI.On("CreateBucket", mock.MatchedBy(func(input *s3.CreateBucketInput) bool {
		return *input.Bucket == "new-bucket"
	})).Return(nil, errors.New("bucket already exists"))

	// Second HeadBucket succeeds (bucket was created by another instance)
	mockAPI.On("HeadBucket", mock.MatchedBy(func(input *s3.HeadBucketInput) bool {
		return *input.Bucket == "new-bucket"
	})).Return(&s3.HeadBucketOutput{}, nil).Once()

	err := ensureBucket(mockAPI, "new-bucket")
	if err != nil {
		t.Errorf("Expected no error (bucket exists after failed create), got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestEnsureBucket_BucketDoesNotExist_CreateFailsAndBucketStillMissing(t *testing.T) {
	mockAPI := new(mockS3API)

	// First HeadBucket fails (bucket doesn't exist)
	mockAPI.On("HeadBucket", mock.MatchedBy(func(input *s3.HeadBucketInput) bool {
		return *input.Bucket == "new-bucket"
	})).Return(nil, awserr.New("NotFound", "bucket not found", nil)).Once()

	// CreateBucket fails
	mockAPI.On("CreateBucket", mock.MatchedBy(func(input *s3.CreateBucketInput) bool {
		return *input.Bucket == "new-bucket"
	})).Return(nil, errors.New("create failed"))

	// Second HeadBucket also fails (bucket still doesn't exist)
	mockAPI.On("HeadBucket", mock.MatchedBy(func(input *s3.HeadBucketInput) bool {
		return *input.Bucket == "new-bucket"
	})).Return(nil, awserr.New("NotFound", "bucket not found", nil)).Once()

	err := ensureBucket(mockAPI, "new-bucket")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	mockAPI.AssertExpectations(t)
}
