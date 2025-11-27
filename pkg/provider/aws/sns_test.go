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

func TestSNSProvider_GenerateTopicName(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{logger: log}
	snsProvider := NewSNSProvider(awsProvider)

	resource := schema.NewSNS("notifications", "backend", "my-stack")
	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
	}

	topicName := snsProvider.generateTopicName(resource, opts)
	
	assert.Equal(t, "my-stack-backend-notifications", topicName)
}

func TestSNSProvider_Create_DryRun(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "123456789012",
		region:    "us-east-1",
	}
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	
	snsProvider := NewSNSProvider(awsProvider)

	resource := schema.NewSNS("test-topic", "backend", "my-stack")
	resource.Spec.DisplayName = "Test Topic"

	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		DryRun:      true,
	}

	result, err := snsProvider.Create(context.Background(), resource, opts)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, schema.KindSNS, result.Kind)
	assert.Equal(t, provider.StatusPending, result.Status)
}

func TestSNSProvider_StandardTopic(t *testing.T) {
	resource := schema.NewSNS("notifications", "backend", "my-stack")
	resource.Spec.DisplayName = "Notifications Topic"
	resource.Spec.FifoTopic = false

	// Verify configuration
	assert.Equal(t, "Notifications Topic", resource.Spec.DisplayName)
	assert.False(t, resource.Spec.FifoTopic)
}

func TestSNSProvider_FIFOTopic(t *testing.T) {
	resource := schema.NewSNS("notifications", "backend", "my-stack")
	resource.Spec.FifoTopic = true
	resource.Spec.ContentBasedDeduplication = true

	// Verify FIFO configuration
	assert.True(t, resource.Spec.FifoTopic)
	assert.True(t, resource.Spec.ContentBasedDeduplication)
}

func TestSNSProvider_WithSubscriptions(t *testing.T) {
	resource := schema.NewSNS("notifications", "backend", "my-stack")
	resource.Spec.Subscriptions = []schema.SNSSubscription{
		{
			Protocol: "email",
			Endpoint: "admin@example.com",
		},
		{
			Protocol:     "sqs",
			Endpoint:     "arn:aws:sqs:us-east-1:123456789012:my-queue",
			FilterPolicy: `{"event": ["order.created"]}`,
		},
		{
			Protocol: "https",
			Endpoint: "https://api.example.com/webhook",
		},
	}

	// Verify subscriptions
	assert.Len(t, resource.Spec.Subscriptions, 3)
	assert.Equal(t, "email", resource.Spec.Subscriptions[0].Protocol)
	assert.Equal(t, "sqs", resource.Spec.Subscriptions[1].Protocol)
	assert.NotEmpty(t, resource.Spec.Subscriptions[1].FilterPolicy)
	assert.Equal(t, "https", resource.Spec.Subscriptions[2].Protocol)
}

func TestSNSProvider_ValidateInputs(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "123456789012",
		region:    "us-east-1",
	}
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	
	snsProvider := NewSNSProvider(awsProvider)

	// Test with invalid resource type
	invalidResource := schema.NewS3("bucket", "backend", "my-stack")
	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		DryRun:      true,
	}

	_, err := snsProvider.Create(context.Background(), invalidResource, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid resource type")
}

func TestSNSProvider_MultiProtocolSubscriptions(t *testing.T) {
	protocols := []string{
		"http",
		"https",
		"email",
		"email-json",
		"sms",
		"sqs",
		"lambda",
		"application",
	}

	for _, protocol := range protocols {
		resource := schema.NewSNS("test-topic", "backend", "my-stack")
		resource.Spec.Subscriptions = []schema.SNSSubscription{
			{
				Protocol: protocol,
				Endpoint: "test-endpoint",
			},
		}

		assert.Len(t, resource.Spec.Subscriptions, 1)
		assert.Equal(t, protocol, resource.Spec.Subscriptions[0].Protocol)
	}
}

func TestSNSProvider_FilterPolicies(t *testing.T) {
	resource := schema.NewSNS("notifications", "backend", "my-stack")
	resource.Spec.Subscriptions = []schema.SNSSubscription{
		{
			Protocol:     "sqs",
			Endpoint:     "arn:aws:sqs:us-east-1:123456789012:my-queue",
			FilterPolicy: `{"event": ["order.created", "order.updated"], "amount": [{"numeric": [">", 100]}]}`,
		},
	}

	// Verify filter policy
	sub := resource.Spec.Subscriptions[0]
	assert.NotEmpty(t, sub.FilterPolicy)
	assert.Contains(t, sub.FilterPolicy, "order.created")
	assert.Contains(t, sub.FilterPolicy, "numeric")
}

