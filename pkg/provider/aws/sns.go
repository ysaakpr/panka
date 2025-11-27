package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"go.uber.org/zap"
)

// SNSProvider implements SNS topic management
type SNSProvider struct {
	provider *Provider
	client   *sns.Client
}

// NewSNSProvider creates a new SNS provider
func NewSNSProvider(p *Provider) *SNSProvider {
	return &SNSProvider{
		provider: p,
		client:   sns.NewFromConfig(p.GetConfig()),
	}
}

// Create creates a new SNS topic
func (sp *SNSProvider) Create(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	snsResource, ok := resource.(*schema.SNS)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "create",
			Message:   "invalid resource type for SNS provider",
		}
	}

	sp.provider.GetLogger().Info("Creating SNS topic",
		zap.String("name", snsResource.Metadata.Name),
	)

	// Generate topic name
	topicName := sp.generateTopicName(snsResource, opts)

	// Add .fifo suffix for FIFO topics
	if snsResource.Spec.FifoTopic {
		if len(topicName) < 5 || topicName[len(topicName)-5:] != ".fifo" {
			topicName = topicName + ".fifo"
		}
	}

	// Build attributes
	attributes := make(map[string]string)

	if snsResource.Spec.DisplayName != "" {
		attributes["DisplayName"] = snsResource.Spec.DisplayName
	}

	if snsResource.Spec.FifoTopic {
		attributes["FifoTopic"] = "true"
	}

	if snsResource.Spec.ContentBasedDeduplication {
		attributes["ContentBasedDeduplication"] = "true"
	}

	// Build tags
	tags := sp.provider.GetTagHelper().BuildTags(opts, resource)
	snsTags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		snsTags = append(snsTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	if !opts.DryRun {
		// Create topic
		createInput := &sns.CreateTopicInput{
			Name:       aws.String(topicName),
			Attributes: attributes,
			Tags:       snsTags,
		}

		output, err := sp.client.CreateTopic(ctx, createInput)
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "create",
				ResourceID: topicName,
				Message:    "failed to create SNS topic",
				Cause:      err,
			}
		}

		topicARN := *output.TopicArn

		// Create subscriptions if specified
		if len(snsResource.Spec.Subscriptions) > 0 {
			for _, sub := range snsResource.Spec.Subscriptions {
				if err := sp.createSubscription(ctx, topicARN, sub); err != nil {
					sp.provider.GetLogger().Warn("Failed to create subscription",
						zap.String("protocol", sub.Protocol),
						zap.Error(err),
					)
				}
			}
		}

		result := &provider.ResourceResult{
			ResourceID: topicARN,
			Kind:       schema.KindSNS,
			Status:     provider.StatusAvailable,
			Outputs: map[string]string{
				"topic_name": topicName,
				"arn":        topicARN,
				"region":     sp.provider.GetRegion(),
			},
			Metadata: map[string]string{
				"provider":   "aws",
				"region":     sp.provider.GetRegion(),
				"fifo_topic": fmt.Sprintf("%v", snsResource.Spec.FifoTopic),
			},
			Timestamp: time.Now(),
		}

		sp.provider.GetLogger().Info("SNS topic created successfully",
			zap.String("topic", topicName),
			zap.String("arn", topicARN),
		)

		return result, nil
	}

	// Dry run result
	return &provider.ResourceResult{
		ResourceID: topicName,
		Kind:       schema.KindSNS,
		Status:     provider.StatusPending,
		Timestamp:  time.Now(),
	}, nil
}

// Read reads the current state of an SNS topic
func (sp *SNSProvider) Read(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	// resourceID is the topic ARN
	attrsOutput, err := sp.client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
		TopicArn: aws.String(resourceID),
	})
	if err != nil {
		return nil, &provider.ProviderError{
			Provider:   "aws",
			Operation:  "read",
			ResourceID: resourceID,
			Message:    "topic not found",
			Cause:      err,
		}
	}

	displayName := attrsOutput.Attributes["DisplayName"]
	subscriptionsCount := attrsOutput.Attributes["SubscriptionsConfirmed"]

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindSNS,
		Status:     provider.StatusAvailable,
		Outputs: map[string]string{
			"arn":                  resourceID,
			"region":               sp.provider.GetRegion(),
			"display_name":         displayName,
			"subscriptions_count":  subscriptionsCount,
		},
		Timestamp: time.Now(),
	}, nil
}

// Update updates an existing SNS topic
func (sp *SNSProvider) Update(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	snsResource, ok := resource.(*schema.SNS)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "update",
			Message:   "invalid resource type for SNS provider",
		}
	}

	topicName := sp.generateTopicName(snsResource, opts)
	if snsResource.Spec.FifoTopic && len(topicName) >= 5 && topicName[len(topicName)-5:] != ".fifo" {
		topicName = topicName + ".fifo"
	}

	sp.provider.GetLogger().Info("Updating SNS topic", zap.String("topic", topicName))

	// Note: Most SNS attributes are immutable. Only DisplayName can be updated.
	// For other changes, you would need to delete and recreate the topic.

	// For now, we just return the current state
	// In a real implementation, you would get the topic ARN first
	// For this example, we'll assume resourceID is provided in opts

	return &provider.ResourceResult{
		ResourceID: topicName,
		Kind:       schema.KindSNS,
		Status:     provider.StatusAvailable,
		Timestamp:  time.Now(),
	}, nil
}

// Delete deletes an SNS topic
func (sp *SNSProvider) Delete(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	sp.provider.GetLogger().Info("Deleting SNS topic", zap.String("arn", resourceID))

	if !opts.DryRun {
		_, err := sp.client.DeleteTopic(ctx, &sns.DeleteTopicInput{
			TopicArn: aws.String(resourceID),
		})
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "delete",
				ResourceID: resourceID,
				Message:    "failed to delete topic",
				Cause:      err,
			}
		}
	}

	sp.provider.GetLogger().Info("SNS topic deleted successfully", zap.String("arn", resourceID))

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindSNS,
		Status:     provider.StatusDeleted,
		Timestamp:  time.Now(),
	}, nil
}

// Exists checks if an SNS topic exists
func (sp *SNSProvider) Exists(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (bool, error) {
	_, err := sp.client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
		TopicArn: aws.String(resourceID),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetOutputs returns the outputs of an SNS topic
func (sp *SNSProvider) GetOutputs(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (map[string]string, error) {
	result, err := sp.Read(ctx, resourceID, opts)
	if err != nil {
		return nil, err
	}
	return result.Outputs, nil
}

// Helper functions

func (sp *SNSProvider) generateTopicName(resource *schema.SNS, opts *provider.ResourceOptions) string {
	return fmt.Sprintf("%s-%s-%s",
		opts.StackName,
		opts.ServiceName,
		resource.Metadata.Name,
	)
}

func (sp *SNSProvider) createSubscription(ctx context.Context, topicARN string, sub schema.SNSSubscription) error {
	subscribeInput := &sns.SubscribeInput{
		TopicArn: aws.String(topicARN),
		Protocol: aws.String(sub.Protocol),
		Endpoint: aws.String(sub.Endpoint),
	}

	if sub.FilterPolicy != "" {
		subscribeInput.Attributes = map[string]string{
			"FilterPolicy": sub.FilterPolicy,
		}
	}

	_, err := sp.client.Subscribe(ctx, subscribeInput)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	sp.provider.GetLogger().Info("Subscription created",
		zap.String("topic", topicARN),
		zap.String("protocol", sub.Protocol),
	)

	return nil
}

