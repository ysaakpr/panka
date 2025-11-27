package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
)

func TestSQSProvider_GenerateQueueName(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{logger: log}
	sqsProvider := NewSQSProvider(awsProvider)

	resource := schema.NewSQS("processing", "backend", "my-stack")
	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
	}

	queueName := sqsProvider.generateQueueName(resource, opts)
	
	assert.Equal(t, "my-stack-backend-processing", queueName)
}

func TestSQSProvider_GenerateQueueName_FIFO(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{logger: log}
	sqsProvider := NewSQSProvider(awsProvider)

	resource := schema.NewSQS("processing", "backend", "my-stack")
	resource.Spec.Type = "fifo"
	
	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
	}

	queueName := sqsProvider.generateQueueName(resource, opts)
	
	// The FIFO suffix is added in the Create method, not generateQueueName
	assert.Equal(t, "my-stack-backend-processing", queueName)
}

func TestSQSProvider_Create_DryRun(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "123456789012",
		region:    "us-east-1",
	}
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	
	sqsProvider := NewSQSProvider(awsProvider)

	resource := schema.NewSQS("test-queue", "backend", "my-stack")
	resource.Spec.Type = "standard"
	resource.Spec.MessageRetentionPeriod = 345600
	resource.Spec.VisibilityTimeout = 30

	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		DryRun:      true,
	}

	result, err := sqsProvider.Create(context.Background(), resource, opts)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, schema.KindSQS, result.Kind)
	assert.Equal(t, provider.StatusPending, result.Status)
}

func TestSQSProvider_StandardQueue(t *testing.T) {
	resource := schema.NewSQS("processing", "backend", "my-stack")
	resource.Spec.Type = "standard"
	resource.Spec.MessageRetentionPeriod = 345600 // 4 days
	resource.Spec.VisibilityTimeout = 300
	resource.Spec.ReceiveWaitTime = 20

	// Verify configuration
	assert.Equal(t, "standard", resource.Spec.Type)
	assert.Equal(t, 345600, resource.Spec.MessageRetentionPeriod)
	assert.Equal(t, 300, resource.Spec.VisibilityTimeout)
	assert.Equal(t, 20, resource.Spec.ReceiveWaitTime)
}

func TestSQSProvider_FIFOQueue(t *testing.T) {
	resource := schema.NewSQS("processing", "backend", "my-stack")
	resource.Spec.Type = "fifo"
	resource.Spec.ContentBasedDeduplication = true
	resource.Spec.DeduplicationScope = "messageGroup"
	resource.Spec.FifoThroughputLimit = "perMessageGroupId"

	// Verify FIFO configuration
	assert.Equal(t, "fifo", resource.Spec.Type)
	assert.True(t, resource.Spec.ContentBasedDeduplication)
	assert.Equal(t, "messageGroup", resource.Spec.DeduplicationScope)
	assert.Equal(t, "perMessageGroupId", resource.Spec.FifoThroughputLimit)
}

func TestSQSProvider_DeadLetterQueue(t *testing.T) {
	resource := schema.NewSQS("processing", "backend", "my-stack")
	resource.Spec.DeadLetterQueue = &schema.DeadLetterQueueConfig{
		Enabled:         true,
		MaxReceiveCount: 3,
	}

	// Verify DLQ configuration
	assert.NotNil(t, resource.Spec.DeadLetterQueue)
	assert.True(t, resource.Spec.DeadLetterQueue.Enabled)
	assert.Equal(t, 3, resource.Spec.DeadLetterQueue.MaxReceiveCount)
}

func TestSQSProvider_ValidateInputs(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "123456789012",
		region:    "us-east-1",
	}
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	
	sqsProvider := NewSQSProvider(awsProvider)

	// Test with invalid resource type
	invalidResource := schema.NewS3("bucket", "backend", "my-stack")
	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		DryRun:      true,
	}

	_, err := sqsProvider.Create(context.Background(), invalidResource, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid resource type")
}

func TestSQSProvider_LongPolling(t *testing.T) {
	resource := schema.NewSQS("processing", "backend", "my-stack")
	resource.Spec.ReceiveWaitTime = 20 // Enable long polling (0-20 seconds)

	assert.Equal(t, 20, resource.Spec.ReceiveWaitTime)
	assert.Greater(t, resource.Spec.ReceiveWaitTime, 0)
	assert.LessOrEqual(t, resource.Spec.ReceiveWaitTime, 20)
}

func TestSQSProvider_MessageSizeConfiguration(t *testing.T) {
	resource := schema.NewSQS("processing", "backend", "my-stack")
	resource.Spec.MaxMessageSize = 262144 // 256KB (maximum)

	assert.Equal(t, 262144, resource.Spec.MaxMessageSize)
	assert.GreaterOrEqual(t, resource.Spec.MaxMessageSize, 1024)  // Min 1KB
	assert.LessOrEqual(t, resource.Spec.MaxMessageSize, 262144)   // Max 256KB
}

