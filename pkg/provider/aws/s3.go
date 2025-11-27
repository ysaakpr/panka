package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"go.uber.org/zap"
)

// S3Provider implements S3 bucket management
type S3Provider struct {
	provider *Provider
	client   *s3.Client
}

// NewS3Provider creates a new S3 provider
func NewS3Provider(p *Provider) *S3Provider {
	return &S3Provider{
		provider: p,
		client:   s3.NewFromConfig(p.GetConfig()),
	}
}

// Create creates a new S3 bucket
func (sp *S3Provider) Create(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	s3Resource, ok := resource.(*schema.S3)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "create",
			Message:   "invalid resource type for S3 provider",
		}
	}

	sp.provider.GetLogger().Info("Creating S3 bucket",
		zap.String("name", s3Resource.Metadata.Name),
	)

	// Generate bucket name if not specified
	bucketName := s3Resource.Spec.Bucket.Name
	if bucketName == "" {
		bucketName = sp.generateBucketName(s3Resource, opts)
	}

	// Build tags
	tags := sp.provider.GetTagHelper().BuildTags(opts, resource)

	// Create bucket
	createInput := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// Set ACL if specified
	if s3Resource.Spec.Bucket.ACL != "" {
		createInput.ACL = types.BucketCannedACL(s3Resource.Spec.Bucket.ACL)
	}

	// Add location constraint for regions other than us-east-1
	if sp.provider.GetRegion() != "us-east-1" {
		createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(sp.provider.GetRegion()),
		}
	}

	if !opts.DryRun {
		_, err := sp.client.CreateBucket(ctx, createInput)
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "create",
				ResourceID: bucketName,
				Message:    "failed to create S3 bucket",
				Cause:      err,
			}
		}

		// Wait for bucket to exist
		waiter := s3.NewBucketExistsWaiter(sp.client)
		if err := waiter.Wait(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		}, 2*time.Minute); err != nil {
			sp.provider.GetLogger().Warn("Bucket created but wait failed", zap.Error(err))
		}

		// Apply tags
		if len(tags) > 0 {
			if err := sp.applyTags(ctx, bucketName, tags); err != nil {
				sp.provider.GetLogger().Warn("Failed to apply tags", zap.Error(err))
			}
		}

		// Configure versioning if enabled
		if s3Resource.Spec.Versioning != nil && s3Resource.Spec.Versioning.Enabled {
			if err := sp.configureVersioning(ctx, bucketName, true); err != nil {
				sp.provider.GetLogger().Warn("Failed to configure versioning", zap.Error(err))
			}
		}

		// Configure encryption if enabled
		if s3Resource.Spec.Encryption != nil && s3Resource.Spec.Encryption.Enabled {
			if err := sp.configureEncryption(ctx, bucketName, s3Resource.Spec.Encryption); err != nil {
				sp.provider.GetLogger().Warn("Failed to configure encryption", zap.Error(err))
			}
		}

		// Configure lifecycle rules
		if len(s3Resource.Spec.Lifecycle) > 0 {
			if err := sp.configureLifecycle(ctx, bucketName, s3Resource.Spec.Lifecycle); err != nil {
				sp.provider.GetLogger().Warn("Failed to configure lifecycle", zap.Error(err))
			}
		}

		// Configure CORS if specified
		if s3Resource.Spec.CORS != nil && len(s3Resource.Spec.CORS.AllowedOrigins) > 0 {
			if err := sp.configureCORS(ctx, bucketName, s3Resource.Spec.CORS); err != nil {
				sp.provider.GetLogger().Warn("Failed to configure CORS", zap.Error(err))
			}
		}
	}

	// Build ARN
	arn := fmt.Sprintf("arn:aws:s3:::%s", bucketName)

	// Determine status based on dry-run
	status := provider.StatusAvailable
	if opts.DryRun {
		status = provider.StatusPending
	}

	result := &provider.ResourceResult{
		ResourceID: bucketName,
		Kind:       schema.KindS3,
		Status:     status,
		Outputs: map[string]string{
			"bucket_name": bucketName,
			"arn":         arn,
			"region":      sp.provider.GetRegion(),
			"endpoint":    fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucketName, sp.provider.GetRegion()),
		},
		Metadata: map[string]string{
			"provider": "aws",
			"region":   sp.provider.GetRegion(),
		},
		Timestamp: time.Now(),
	}

	sp.provider.GetLogger().Info("S3 bucket created successfully",
		zap.String("bucket", bucketName),
		zap.String("arn", arn),
	)

	return result, nil
}

// Read reads the current state of an S3 bucket
func (sp *S3Provider) Read(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	// Check if bucket exists
	_, err := sp.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(resourceID),
	})
	if err != nil {
		return nil, &provider.ProviderError{
			Provider:   "aws",
			Operation:  "read",
			ResourceID: resourceID,
			Message:    "bucket not found or access denied",
			Cause:      err,
		}
	}

	arn := fmt.Sprintf("arn:aws:s3:::%s", resourceID)

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindS3,
		Status:     provider.StatusAvailable,
		Outputs: map[string]string{
			"bucket_name": resourceID,
			"arn":         arn,
			"region":      sp.provider.GetRegion(),
		},
		Timestamp: time.Now(),
	}, nil
}

// Update updates an existing S3 bucket
func (sp *S3Provider) Update(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	s3Resource, ok := resource.(*schema.S3)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "update",
			Message:   "invalid resource type for S3 provider",
		}
	}

	bucketName := s3Resource.Spec.Bucket.Name
	if bucketName == "" {
		bucketName = sp.generateBucketName(s3Resource, opts)
	}

	sp.provider.GetLogger().Info("Updating S3 bucket", zap.String("bucket", bucketName))

	// Update versioning
	if s3Resource.Spec.Versioning != nil {
		if err := sp.configureVersioning(ctx, bucketName, s3Resource.Spec.Versioning.Enabled); err != nil {
			return nil, err
		}
	}

	// Update encryption
	if s3Resource.Spec.Encryption != nil && s3Resource.Spec.Encryption.Enabled {
		if err := sp.configureEncryption(ctx, bucketName, s3Resource.Spec.Encryption); err != nil {
			return nil, err
		}
	}

	// Update lifecycle
	if len(s3Resource.Spec.Lifecycle) > 0 {
		if err := sp.configureLifecycle(ctx, bucketName, s3Resource.Spec.Lifecycle); err != nil {
			return nil, err
		}
	}

	return sp.Read(ctx, bucketName, opts)
}

// Delete deletes an S3 bucket
func (sp *S3Provider) Delete(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	sp.provider.GetLogger().Info("Deleting S3 bucket", zap.String("bucket", resourceID))

	if !opts.DryRun {
		// Note: Bucket must be empty before deletion
		// In production, you might want to add logic to empty the bucket first
		_, err := sp.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(resourceID),
		})
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "delete",
				ResourceID: resourceID,
				Message:    "failed to delete S3 bucket",
				Cause:      err,
			}
		}

		// Wait for bucket to be deleted
		waiter := s3.NewBucketNotExistsWaiter(sp.client)
		if err := waiter.Wait(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(resourceID),
		}, 2*time.Minute); err != nil {
			sp.provider.GetLogger().Warn("Bucket deleted but wait failed", zap.Error(err))
		}
	}

	sp.provider.GetLogger().Info("S3 bucket deleted successfully", zap.String("bucket", resourceID))

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindS3,
		Status:     provider.StatusDeleted,
		Timestamp:  time.Now(),
	}, nil
}

// Exists checks if an S3 bucket exists
func (sp *S3Provider) Exists(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (bool, error) {
	_, err := sp.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(resourceID),
	})
	if err != nil {
		// Bucket doesn't exist or we don't have access
		return false, nil
	}
	return true, nil
}

// GetOutputs returns the outputs of an S3 bucket
func (sp *S3Provider) GetOutputs(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (map[string]string, error) {
	result, err := sp.Read(ctx, resourceID, opts)
	if err != nil {
		return nil, err
	}
	return result.Outputs, nil
}

// Helper functions

func (sp *S3Provider) generateBucketName(resource *schema.S3, opts *provider.ResourceOptions) string {
	// Generate a unique bucket name
	// Format: {stack}-{service}-{resource}-{region}-{account}
	name := fmt.Sprintf("%s-%s-%s",
		opts.StackName,
		opts.ServiceName,
		resource.Metadata.Name,
	)

	// S3 bucket names must be lowercase
	return toLowerAlphanumeric(name)
}

func (sp *S3Provider) applyTags(ctx context.Context, bucket string, tags map[string]string) error {
	tagSet := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		tagSet = append(tagSet, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	_, err := sp.client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(bucket),
		Tagging: &types.Tagging{
			TagSet: tagSet,
		},
	})
	return err
}

func (sp *S3Provider) configureVersioning(ctx context.Context, bucket string, enabled bool) error {
	status := types.BucketVersioningStatusSuspended
	if enabled {
		status = types.BucketVersioningStatusEnabled
	}

	_, err := sp.client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: status,
		},
	})
	return err
}

func (sp *S3Provider) configureEncryption(ctx context.Context, bucket string, encryption *schema.S3Encryption) error {
	rule := types.ServerSideEncryptionRule{
		ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
			SSEAlgorithm: types.ServerSideEncryption(encryption.Algorithm),
		},
	}

	if encryption.KMSKeyID != "" {
		rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID = aws.String(encryption.KMSKeyID)
	}

	_, err := sp.client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucket),
		ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
			Rules: []types.ServerSideEncryptionRule{rule},
		},
	})
	return err
}

func (sp *S3Provider) configureLifecycle(ctx context.Context, bucket string, rules []schema.LifecycleRule) error {
	lifecycleRules := make([]types.LifecycleRule, 0, len(rules))

	for _, rule := range rules {
		lifecycleRule := types.LifecycleRule{
			ID:     aws.String(rule.ID),
			Status: types.ExpirationStatusEnabled,
		}

		if !rule.Enabled {
			lifecycleRule.Status = types.ExpirationStatusDisabled
		}

		if rule.Prefix != "" {
			lifecycleRule.Prefix = aws.String(rule.Prefix)
		}

		if rule.Expiration != nil {
			lifecycleRule.Expiration = &types.LifecycleExpiration{
				Days: aws.Int32(int32(rule.Expiration.Days)),
			}
		}

		lifecycleRules = append(lifecycleRules, lifecycleRule)
	}

	_, err := sp.client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: lifecycleRules,
		},
	})
	return err
}

func (sp *S3Provider) configureCORS(ctx context.Context, bucket string, corsConfig *schema.CORSConfig) error {
	corsRule := types.CORSRule{
		AllowedOrigins: corsConfig.AllowedOrigins,
		AllowedMethods: corsConfig.AllowedMethods,
	}

	if len(corsConfig.AllowedHeaders) > 0 {
		corsRule.AllowedHeaders = corsConfig.AllowedHeaders
	}

	if len(corsConfig.ExposeHeaders) > 0 {
		corsRule.ExposeHeaders = corsConfig.ExposeHeaders
	}

	if corsConfig.MaxAgeSeconds > 0 {
		corsRule.MaxAgeSeconds = aws.Int32(int32(corsConfig.MaxAgeSeconds))
	}

	_, err := sp.client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{corsRule},
		},
	})
	return err
}

// Helper function to convert string to lowercase alphanumeric with hyphens
func toLowerAlphanumeric(s string) string {
	result := ""
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			result += string(c)
		} else if c >= 'A' && c <= 'Z' {
			result += string(c + 32) // Convert to lowercase
		} else if c == '_' || c == ' ' {
			result += "-"
		}
	}
	return result
}

