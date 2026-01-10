package sqs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"
)

// sqsAPI defines the SQS operations we need
type sqsAPI interface {
	GetQueueUrl(input *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error)
	CreateQueue(input *sqs.CreateQueueInput) (*sqs.CreateQueueOutput, error)
	SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
	ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	GetQueueAttributes(input *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error)
}

// Client wraps the SQS client for LocalStack
type Client struct {
	sqsClient sqsAPI
	queueURL  string
	logger    *zap.Logger
}

// Config holds SQS configuration
type Config struct {
	Endpoint string
	Region   string
	Queue    string
}

// Message represents an SQS message
type Message struct {
	TransactionID string    `json:"transaction_id"`
	Region         string    `json:"region"`
	Action         string    `json:"action"`
	Timestamp      time.Time `json:"timestamp"`
	Data           string    `json:"data"`
}

// New creates a new SQS client
func New(config Config, logger *zap.Logger) (*Client, error) {
	// Create AWS session with LocalStack endpoint
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(config.Region),
		Endpoint:    aws.String(config.Endpoint),
		Credentials: credentials.NewStaticCredentials("test", "test", ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	sqsClient := sqs.New(sess)

	// Get or create queue
	queueURL, err := ensureQueue(sqsClient, config.Queue, config.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure queue exists: %w", err)
	}

	logger.Info("SQS client initialized",
		zap.String("endpoint", config.Endpoint),
		zap.String("region", config.Region),
		zap.String("queue", config.Queue),
		zap.String("queue_url", queueURL),
	)

	return &Client{
		sqsClient: sqsClient,
		queueURL:  queueURL,
		logger:    logger,
	}, nil
}

// ensureQueue gets the queue URL or creates the queue if it doesn't exist
func ensureQueue(sqsClient sqsAPI, queueName, region string) (string, error) {
	// Try to get queue URL
	result, err := sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err == nil {
		return *result.QueueUrl, nil
	}

	// Queue doesn't exist, create it
	createResult, err := sqsClient.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
		Attributes: map[string]*string{
			"VisibilityTimeoutSeconds":   aws.String("30"),
			"MessageRetentionPeriod":    aws.String("1209600"), // 14 days
			"ReceiveMessageWaitTimeSeconds": aws.String("0"), // Short polling
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create queue: %w", err)
	}

	return *createResult.QueueUrl, nil
}

// SendMessage sends a message to the queue
func (c *Client) SendMessage(msg *Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = c.sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(c.queueURL),
		MessageBody: aws.String(string(body)),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"Region": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.Region),
			},
			"Action": {
				DataType:    aws.String("String"),
				StringValue: aws.String(msg.Action),
			},
		},
	})

	if err != nil {
		c.logger.Error("Failed to send message to SQS",
			zap.Error(err),
			zap.String("transaction_id", msg.TransactionID),
		)
		return fmt.Errorf("failed to send message: %w", err)
	}

	c.logger.Info("Message sent",
		zap.String("transaction_id", msg.TransactionID),
		zap.String("action", msg.Action),
	)

	return nil
}

// ReceiveMessages receives messages from the queue
func (c *Client) ReceiveMessages(maxMessages int64, waitTimeSeconds int64) ([]*Message, error) {
	result, err := c.sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: aws.Int64(maxMessages),
		WaitTimeSeconds:     aws.Int64(waitTimeSeconds),
		MessageAttributeNames: []*string{
			aws.String("All"),
		},
	})

	if err != nil {
		c.logger.Error("Failed to receive messages from SQS", zap.Error(err))
		return nil, fmt.Errorf("failed to receive messages: %w", err)
	}

	var messages []*Message
	for _, sqsMsg := range result.Messages {
		var msg Message
		if err := json.Unmarshal([]byte(*sqsMsg.Body), &msg); err != nil {
			c.logger.Warn("Failed to unmarshal message",
				zap.Error(err),
				zap.String("message_id", *sqsMsg.MessageId),
			)
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

// DeleteMessage deletes a message from the queue
func (c *Client) DeleteMessage(receiptHandle string) error {
	_, err := c.sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})

	if err != nil {
		c.logger.Error("Failed to delete message from SQS",
			zap.Error(err),
			zap.String("receipt_handle", receiptHandle),
		)
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// Health checks if SQS is accessible
func (c *Client) Health() error {
	_, err := c.sqsClient.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(c.queueURL),
		AttributeNames: []*string{
			aws.String("All"),
		},
	})
	if err != nil {
		return fmt.Errorf("SQS health check failed: %w", err)
	}
	return nil
}

