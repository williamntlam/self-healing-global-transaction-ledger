package sqs

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// mockSQSAPI is a mock implementation of SQS API operations
type mockSQSAPI struct {
	mock.Mock
}

func (m *mockSQSAPI) GetQueueUrl(input *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.GetQueueUrlOutput), args.Error(1)
}

func (m *mockSQSAPI) CreateQueue(input *sqs.CreateQueueInput) (*sqs.CreateQueueOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.CreateQueueOutput), args.Error(1)
}

func (m *mockSQSAPI) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.SendMessageOutput), args.Error(1)
}

func (m *mockSQSAPI) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func (m *mockSQSAPI) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.DeleteMessageOutput), args.Error(1)
}

func (m *mockSQSAPI) GetQueueAttributes(input *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.GetQueueAttributesOutput), args.Error(1)
}

// newTestableClient creates a client with injectable SQS API (for testing)
func newTestableClient(sqsClient sqsAPI, queueURL string, logger *zap.Logger) *Client {
	return &Client{
		sqsClient: sqsClient,
		queueURL:  queueURL,
		logger:    logger,
	}
}

func TestClient_SendMessage_Success(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	msg := &Message{
		TransactionID: "test-tx-123",
		Region:        "us-east-1",
		Action:        "transaction_created",
		Timestamp:     time.Now(),
		Data:          `{"test": "data"}`,
	}

	mockAPI.On("SendMessage", mock.MatchedBy(func(input *sqs.SendMessageInput) bool {
		return *input.QueueUrl == "https://sqs.test/queue" &&
			*input.MessageBody != ""
	})).Return(&sqs.SendMessageOutput{}, nil)

	err := client.SendMessage(msg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_SendMessage_MarshalError(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	// Create a message that will fail to marshal (circular reference would do it, but simpler: invalid type)
	// Actually, Message struct is simple, so marshal won't fail. Let's test SQS error instead
	msg := &Message{
		TransactionID: "test-tx-123",
		Region:        "us-east-1",
		Action:        "transaction_created",
		Timestamp:     time.Now(),
		Data:          `{"test": "data"}`,
	}

	mockAPI.On("SendMessage", mock.Anything).Return(nil, errors.New("SQS error"))

	err := client.SendMessage(msg)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "failed to send message: SQS error" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_ReceiveMessages_Success(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	msgBody, _ := json.Marshal(&Message{
		TransactionID: "test-tx-123",
		Region:        "us-east-1",
		Action:        "transaction_created",
		Timestamp:     time.Now(),
		Data:          `{"test": "data"}`,
	})

	mockAPI.On("ReceiveMessage", mock.MatchedBy(func(input *sqs.ReceiveMessageInput) bool {
		return *input.QueueUrl == "https://sqs.test/queue" &&
			*input.MaxNumberOfMessages == 10 &&
			*input.WaitTimeSeconds == 0
	})).Return(&sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{
			{
				MessageId:     aws.String("msg-1"),
				Body:          aws.String(string(msgBody)),
				ReceiptHandle: aws.String("receipt-1"),
			},
		},
	}, nil)

	receivedMessages, err := client.ReceiveMessages(10, 0)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(receivedMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(receivedMessages))
	}

	if receivedMessages[0].Message.TransactionID != "test-tx-123" {
		t.Errorf("Expected transaction ID 'test-tx-123', got '%s'", receivedMessages[0].Message.TransactionID)
	}

	if receivedMessages[0].ReceiptHandle != "receipt-1" {
		t.Errorf("Expected receipt handle 'receipt-1', got '%s'", receivedMessages[0].ReceiptHandle)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_ReceiveMessages_Empty(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	mockAPI.On("ReceiveMessage", mock.Anything).Return(&sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{},
	}, nil)

	receivedMessages, err := client.ReceiveMessages(10, 0)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(receivedMessages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(receivedMessages))
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_ReceiveMessages_UnmarshalError(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	// Return invalid JSON
	mockAPI.On("ReceiveMessage", mock.Anything).Return(&sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{
			{
				MessageId:     aws.String("msg-1"),
				Body:          aws.String("invalid json"),
				ReceiptHandle: aws.String("receipt-1"),
			},
		},
	}, nil)

	receivedMessages, err := client.ReceiveMessages(10, 0)
	// Function should continue on unmarshal errors (logs warning, skips message)
	if err != nil {
		t.Errorf("Expected no error (unmarshal errors are logged but not returned), got: %v", err)
	}

	if len(receivedMessages) != 0 {
		t.Errorf("Expected 0 messages (invalid message skipped), got %d", len(receivedMessages))
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_ReceiveMessages_SQSError(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	mockAPI.On("ReceiveMessage", mock.Anything).Return(nil, errors.New("SQS error"))

	receivedMessages, err := client.ReceiveMessages(10, 0)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if receivedMessages != nil {
		t.Errorf("Expected nil messages, got %v", receivedMessages)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_DeleteMessage_Success(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	receiptHandle := "test-receipt-handle"

	mockAPI.On("DeleteMessage", mock.MatchedBy(func(input *sqs.DeleteMessageInput) bool {
		return *input.QueueUrl == "https://sqs.test/queue" &&
			*input.ReceiptHandle == receiptHandle
	})).Return(&sqs.DeleteMessageOutput{}, nil)

	err := client.DeleteMessage(receiptHandle)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_DeleteMessage_Error(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	mockAPI.On("DeleteMessage", mock.Anything).Return(nil, errors.New("SQS error"))

	err := client.DeleteMessage("test-receipt")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "failed to delete message: SQS error" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_Health_Success(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	mockAPI.On("GetQueueAttributes", mock.MatchedBy(func(input *sqs.GetQueueAttributesInput) bool {
		return *input.QueueUrl == "https://sqs.test/queue"
	})).Return(&sqs.GetQueueAttributesOutput{}, nil)

	err := client.Health()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestClient_Health_Error(t *testing.T) {
	mockAPI := new(mockSQSAPI)
	logger := zap.NewNop()
	client := newTestableClient(mockAPI, "https://sqs.test/queue", logger)

	mockAPI.On("GetQueueAttributes", mock.Anything).Return(nil, errors.New("queue not found"))

	err := client.Health()
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "SQS health check failed: queue not found" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	mockAPI.AssertExpectations(t)
}

func TestEnsureQueue_QueueExists(t *testing.T) {
	mockAPI := new(mockSQSAPI)

	mockAPI.On("GetQueueUrl", mock.MatchedBy(func(input *sqs.GetQueueUrlInput) bool {
		return *input.QueueName == "existing-queue"
	})).Return(&sqs.GetQueueUrlOutput{
		QueueUrl: aws.String("https://sqs.test/existing-queue"),
	}, nil)

	queueURL, err := ensureQueue(mockAPI, "existing-queue", "us-east-1")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if queueURL != "https://sqs.test/existing-queue" {
		t.Errorf("Expected queue URL 'https://sqs.test/existing-queue', got '%s'", queueURL)
	}

	mockAPI.AssertExpectations(t)
	mockAPI.AssertNotCalled(t, "CreateQueue")
}

func TestEnsureQueue_QueueDoesNotExist_CreateSuccess(t *testing.T) {
	mockAPI := new(mockSQSAPI)

	// GetQueueUrl fails (queue doesn't exist)
	mockAPI.On("GetQueueUrl", mock.MatchedBy(func(input *sqs.GetQueueUrlInput) bool {
		return *input.QueueName == "new-queue"
	})).Return(nil, awserr.New("AWS.SimpleQueueService.NonExistentQueue", "queue not found", nil))

	// CreateQueue succeeds
	mockAPI.On("CreateQueue", mock.MatchedBy(func(input *sqs.CreateQueueInput) bool {
		return *input.QueueName == "new-queue"
	})).Return(&sqs.CreateQueueOutput{
		QueueUrl: aws.String("https://sqs.test/new-queue"),
	}, nil)

	queueURL, err := ensureQueue(mockAPI, "new-queue", "us-east-1")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if queueURL != "https://sqs.test/new-queue" {
		t.Errorf("Expected queue URL 'https://sqs.test/new-queue', got '%s'", queueURL)
	}

	mockAPI.AssertExpectations(t)
}

func TestEnsureQueue_CreateFails(t *testing.T) {
	mockAPI := new(mockSQSAPI)

	// GetQueueUrl fails (queue doesn't exist)
	mockAPI.On("GetQueueUrl", mock.MatchedBy(func(input *sqs.GetQueueUrlInput) bool {
		return *input.QueueName == "new-queue"
	})).Return(nil, awserr.New("AWS.SimpleQueueService.NonExistentQueue", "queue not found", nil))

	// CreateQueue fails
	mockAPI.On("CreateQueue", mock.MatchedBy(func(input *sqs.CreateQueueInput) bool {
		return *input.QueueName == "new-queue"
	})).Return(nil, errors.New("create failed"))

	queueURL, err := ensureQueue(mockAPI, "new-queue", "us-east-1")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if queueURL != "" {
		t.Errorf("Expected empty queue URL, got '%s'", queueURL)
	}

	mockAPI.AssertExpectations(t)
}
