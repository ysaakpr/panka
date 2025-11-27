package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"go.uber.org/zap"
)

// SQSProvider implements SQS queue management
type SQSProvider struct {
	provider *Provider
	client   *sqs.Client
}

// NewSQSProvider creates a new SQS provider
func NewSQSProvider(p *Provider) *SQSProvider {
	return &SQSProvider{
		provider: p,
		client:   sqs.NewFromConfig(p.GetConfig()),
	}
}

// Create creates a new SQS queue
func (sp *SQSProvider) Create(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	sqsResource, ok := resource.(*schema.SQS)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "create",
			Message:   "invalid resource type for SQS provider",
		}
	}

	sp.provider.GetLogger().Info("Creating SQS queue",
		zap.String("name", sqsResource.Metadata.Name),
		zap.String("type", sqsResource.Spec.Type),
	)

	// Generate queue name
	queueName := sp.generateQueueName(sqsResource, opts)

	// Build queue attributes
	attributes := make(map[string]string)

	// Message retention period (default: 4 days)
	if sqsResource.Spec.MessageRetentionPeriod > 0 {
		attributes["MessageRetentionPeriod"] = fmt.Sprintf("%d", sqsResource.Spec.MessageRetentionPeriod)
	}

	// Visibility timeout (default: 30 seconds)
	if sqsResource.Spec.VisibilityTimeout > 0 {
		attributes["VisibilityTimeout"] = fmt.Sprintf("%d", sqsResource.Spec.VisibilityTimeout)
	}

	// Maximum message size
	if sqsResource.Spec.MaxMessageSize > 0 {
		attributes["MaximumMessageSize"] = fmt.Sprintf("%d", sqsResource.Spec.MaxMessageSize)
	}

	// Receive wait time (long polling)
	if sqsResource.Spec.ReceiveWaitTime > 0 {
		attributes["ReceiveMessageWaitTimeSeconds"] = fmt.Sprintf("%d", sqsResource.Spec.ReceiveWaitTime)
	}

	// Delay seconds
	if sqsResource.Spec.DelaySeconds > 0 {
		attributes["DelaySeconds"] = fmt.Sprintf("%d", sqsResource.Spec.DelaySeconds)
	}

	// FIFO queue specific attributes
	if sqsResource.Spec.Type == "fifo" {
		attributes["FifoQueue"] = "true"

		if sqsResource.Spec.ContentBasedDeduplication {
			attributes["ContentBasedDeduplication"] = "true"
		}

		if sqsResource.Spec.DeduplicationScope != "" {
			attributes["DeduplicationScope"] = sqsResource.Spec.DeduplicationScope
		}

		if sqsResource.Spec.FifoThroughputLimit != "" {
			attributes["FifoThroughputLimit"] = sqsResource.Spec.FifoThroughputLimit
		}

		// Add .fifo suffix if not present
		if len(queueName) < 5 || queueName[len(queueName)-5:] != ".fifo" {
			queueName = queueName + ".fifo"
		}
	}

	// Build tags
	tags := sp.provider.GetTagHelper().BuildTags(opts, resource)

	if !opts.DryRun {
		// Create queue
		createInput := &sqs.CreateQueueInput{
			QueueName:  aws.String(queueName),
			Attributes: attributes,
			Tags:       tags,
		}

		output, err := sp.client.CreateQueue(ctx, createInput)
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "create",
				ResourceID: queueName,
				Message:    "failed to create SQS queue",
				Cause:      err,
			}
		}

		queueURL := *output.QueueUrl

		// Get queue ARN
		attrsOutput, err := sp.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(queueURL),
			AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameQueueArn},
		})
		if err != nil {
			sp.provider.GetLogger().Warn("Failed to get queue ARN", zap.Error(err))
		}

		queueARN := ""
		if attrsOutput != nil && attrsOutput.Attributes != nil {
			queueARN = attrsOutput.Attributes["QueueArn"]
		}

		// Configure dead letter queue if specified
		if sqsResource.Spec.DeadLetterQueue != nil && sqsResource.Spec.DeadLetterQueue.Enabled {
			if err := sp.configureDLQ(ctx, queueURL, sqsResource.Spec.DeadLetterQueue); err != nil {
				sp.provider.GetLogger().Warn("Failed to configure DLQ", zap.Error(err))
			}
		}

		result := &provider.ResourceResult{
			ResourceID: queueURL,
			Kind:       schema.KindSQS,
			Status:     provider.StatusAvailable,
			Outputs: map[string]string{
				"queue_name": queueName,
				"queue_url":  queueURL,
				"arn":        queueARN,
				"region":     sp.provider.GetRegion(),
			},
			Metadata: map[string]string{
				"provider": "aws",
				"region":   sp.provider.GetRegion(),
				"type":     sqsResource.Spec.Type,
			},
			Timestamp: time.Now(),
		}

		sp.provider.GetLogger().Info("SQS queue created successfully",
			zap.String("queue", queueName),
			zap.String("url", queueURL),
			zap.String("arn", queueARN),
		)

		return result, nil
	}

	// Dry run result
	return &provider.ResourceResult{
		ResourceID: queueName,
		Kind:       schema.KindSQS,
		Status:     provider.StatusPending,
		Timestamp:  time.Now(),
	}, nil
}

// Read reads the current state of an SQS queue
func (sp *SQSProvider) Read(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	// resourceID is the queue URL
	attrsOutput, err := sp.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(resourceID),
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameQueueArn,
			types.QueueAttributeNameApproximateNumberOfMessages,
		},
	})
	if err != nil {
		return nil, &provider.ProviderError{
			Provider:   "aws",
			Operation:  "read",
			ResourceID: resourceID,
			Message:    "queue not found",
			Cause:      err,
		}
	}

	queueARN := attrsOutput.Attributes["QueueArn"]
	messageCount := attrsOutput.Attributes["ApproximateNumberOfMessages"]

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindSQS,
		Status:     provider.StatusAvailable,
		Outputs: map[string]string{
			"queue_url":     resourceID,
			"arn":           queueARN,
			"region":        sp.provider.GetRegion(),
			"message_count": messageCount,
		},
		Timestamp: time.Now(),
	}, nil
}

// Update updates an existing SQS queue
func (sp *SQSProvider) Update(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	sqsResource, ok := resource.(*schema.SQS)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "update",
			Message:   "invalid resource type for SQS provider",
		}
	}

	queueName := sp.generateQueueName(sqsResource, opts)
	if sqsResource.Spec.Type == "fifo" && len(queueName) >= 5 && queueName[len(queueName)-5:] != ".fifo" {
		queueName = queueName + ".fifo"
	}

	sp.provider.GetLogger().Info("Updating SQS queue", zap.String("queue", queueName))

	// Get queue URL
	urlOutput, err := sp.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, &provider.ProviderError{
			Provider:   "aws",
			Operation:  "update",
			ResourceID: queueName,
			Message:    "queue not found",
			Cause:      err,
		}
	}

	queueURL := *urlOutput.QueueUrl

	// Build updated attributes
	attributes := make(map[string]string)

	if sqsResource.Spec.MessageRetentionPeriod > 0 {
		attributes["MessageRetentionPeriod"] = fmt.Sprintf("%d", sqsResource.Spec.MessageRetentionPeriod)
	}

	if sqsResource.Spec.VisibilityTimeout > 0 {
		attributes["VisibilityTimeout"] = fmt.Sprintf("%d", sqsResource.Spec.VisibilityTimeout)
	}

	// Update queue attributes
	if len(attributes) > 0 {
		_, err = sp.client.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{
			QueueUrl:   aws.String(queueURL),
			Attributes: attributes,
		})
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "update",
				ResourceID: queueURL,
				Message:    "failed to update queue",
				Cause:      err,
			}
		}
	}

	return sp.Read(ctx, queueURL, opts)
}

// Delete deletes an SQS queue
func (sp *SQSProvider) Delete(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	sp.provider.GetLogger().Info("Deleting SQS queue", zap.String("url", resourceID))

	if !opts.DryRun {
		_, err := sp.client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
			QueueUrl: aws.String(resourceID),
		})
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "delete",
				ResourceID: resourceID,
				Message:    "failed to delete queue",
				Cause:      err,
			}
		}
	}

	sp.provider.GetLogger().Info("SQS queue deleted successfully", zap.String("url", resourceID))

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindSQS,
		Status:     provider.StatusDeleted,
		Timestamp:  time.Now(),
	}, nil
}

// Exists checks if an SQS queue exists
func (sp *SQSProvider) Exists(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (bool, error) {
	_, err := sp.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(resourceID),
		AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetOutputs returns the outputs of an SQS queue
func (sp *SQSProvider) GetOutputs(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (map[string]string, error) {
	result, err := sp.Read(ctx, resourceID, opts)
	if err != nil {
		return nil, err
	}
	return result.Outputs, nil
}

// Helper functions

func (sp *SQSProvider) generateQueueName(resource *schema.SQS, opts *provider.ResourceOptions) string {
	return fmt.Sprintf("%s-%s-%s",
		opts.StackName,
		opts.ServiceName,
		resource.Metadata.Name,
	)
}

func (sp *SQSProvider) configureDLQ(ctx context.Context, queueURL string, dlqConfig *schema.DeadLetterQueueConfig) error {
	// Note: In a real implementation, you would need to create the DLQ first
	// and get its ARN, then configure the redrive policy
	// For now, this is a placeholder
	sp.provider.GetLogger().Info("DLQ configuration would be applied here")
	return nil
}

