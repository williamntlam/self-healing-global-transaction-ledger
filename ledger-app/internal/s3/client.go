package s3

import (
	"bytes"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.uber.org/zap"
)

// Client wraps the S3 client for LocalStack
type Client struct {
	s3Client s3API
	bucket   string
	logger   *zap.Logger
}

// Config holds S3 configuration
type Config struct {
	Endpoint string
	Region   string
	Bucket   string
}

// New creates a new S3 client
func New(config Config, logger *zap.Logger) (*Client, error) {
	// Create AWS session with LocalStack endpoint
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(config.Region),
		Endpoint:         aws.String(config.Endpoint),
		S3ForcePathStyle: aws.Bool(true), // Required for LocalStack
		Credentials:      credentials.NewStaticCredentials("test", "test", ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	s3Client := s3.New(sess)

	// Ensure bucket exists
	if err := ensureBucket(s3Client, config.Bucket); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	logger.Info("S3 client initialized",
		zap.String("endpoint", config.Endpoint),
		zap.String("region", config.Region),
		zap.String("bucket", config.Bucket),
	)

	return &Client{
		s3Client: s3Client,
		bucket:   config.Bucket,
		logger:   logger,
	}, nil
}

// s3API defines the S3 operations we need
type s3API interface {
	HeadBucket(input *s3.HeadBucketInput) (*s3.HeadBucketOutput, error)
	CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error)
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

// ensureBucket creates the bucket if it doesn't exist
func ensureBucket(s3Client s3API, bucketName string) error {
	// Check if bucket exists
	_, err := s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		// Bucket exists
		return nil
	}

	// Try to create the bucket
	_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// Bucket might have been created by another instance
		// Check again
		_, checkErr := s3Client.HeadBucket(&s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if checkErr != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return nil
}

// WriteAuditLog writes an audit log entry to S3
func (c *Client) WriteAuditLog(key string, content []byte) error {
	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		c.logger.Error("Failed to write audit log to S3",
			zap.Error(err),
			zap.String("key", key),
		)
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	c.logger.Info("Audit log written to S3",
		zap.String("key", key),
		zap.String("bucket", c.bucket),
	)

	return nil
}

// WriteAuditLogWithTimestamp writes an audit log with a timestamp-based key
func (c *Client) WriteAuditLogWithTimestamp(prefix string, content []byte) error {
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05")
	key := fmt.Sprintf("%s/%s-%d.json", prefix, timestamp, time.Now().UnixNano())
	return c.WriteAuditLog(key, content)
}

// Health checks if S3 is accessible
func (c *Client) Health() error {
	_, err := c.s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("S3 health check failed: %w", err)
	}
	return nil
}

