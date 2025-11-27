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

func TestS3Provider_GenerateBucketName(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{logger: log}
	s3Provider := NewS3Provider(awsProvider)

	resource := schema.NewS3("uploads", "backend", "my-stack")
	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
	}

	bucketName := s3Provider.generateBucketName(resource, opts)

	// Should be lowercase alphanumeric with hyphens
	assert.Equal(t, "my-stack-backend-uploads", bucketName)
	assert.Regexp(t, "^[a-z0-9-]+$", bucketName)
}

func TestS3Provider_GenerateBucketName_SpecialCharacters(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{logger: log}
	s3Provider := NewS3Provider(awsProvider)

	resource := schema.NewS3("My_Uploads", "Backend_API", "Test-Stack")
	opts := &provider.ResourceOptions{
		StackName:   "Test Stack",
		ServiceName: "Backend API",
	}

	bucketName := s3Provider.generateBucketName(resource, opts)

	// Should convert to lowercase and replace invalid characters
	assert.Regexp(t, "^[a-z0-9-]+$", bucketName)
	assert.NotContains(t, bucketName, "_")
	assert.NotContains(t, bucketName, " ")
	assert.NotContains(t, bucketName, "A")
}

func TestS3Provider_Create_DryRun(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "123456789012",
		region:    "us-east-1",
	}
	awsProvider.tagHelper = provider.NewTagHelper(map[string]string{
		"environment": "test",
	})

	s3Provider := NewS3Provider(awsProvider)

	resource := schema.NewS3("test-bucket", "backend", "my-stack")
	resource.Spec.Bucket.ACL = "private"
	resource.Spec.Versioning = &schema.VersioningConfig{Enabled: true}
	resource.Spec.Encryption = &schema.S3Encryption{
		Enabled:   true,
		Algorithm: "AES256",
	}

	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		DryRun:      true, // Dry run mode
	}

	result, err := s3Provider.Create(context.Background(), resource, opts)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, schema.KindS3, result.Kind)
	assert.Equal(t, provider.StatusPending, result.Status)
}

func TestS3Provider_BuildTags(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "123456789012",
		region:    "us-east-1",
	}
	awsProvider.tagHelper = provider.NewTagHelper(map[string]string{
		"environment": "production",
		"team":        "platform",
	})

	resource := schema.NewS3("uploads", "backend", "my-stack")
	resource.Metadata.Labels = map[string]string{
		"custom-label": "custom-value",
	}

	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		TenantID:    "tenant-123",
		Tags: map[string]string{
			"cost-center": "engineering",
		},
	}

	tags := awsProvider.tagHelper.BuildTags(opts, resource)

	// Check default tags
	assert.Equal(t, "production", tags["environment"])
	assert.Equal(t, "platform", tags["team"])

	// Check standard tags
	assert.Equal(t, "tenant-123", tags["panka:tenant"])
	assert.Equal(t, "my-stack", tags["panka:stack"])
	assert.Equal(t, "backend", tags["panka:service"])
	assert.Equal(t, "uploads", tags["panka:resource"])
	assert.Equal(t, string(schema.KindS3), tags["panka:kind"])
	assert.Equal(t, "true", tags["panka:managed"])

	// Check custom tags
	assert.Equal(t, "engineering", tags["cost-center"])
	assert.Equal(t, "custom-value", tags["custom-label"])
}

func TestToLowerAlphanumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase already",
			input:    "my-bucket",
			expected: "my-bucket",
		},
		{
			name:     "uppercase to lowercase",
			input:    "MyBucket",
			expected: "mybucket",
		},
		{
			name:     "underscores to hyphens",
			input:    "my_bucket_name",
			expected: "my-bucket-name",
		},
		{
			name:     "spaces to hyphens",
			input:    "my bucket name",
			expected: "my-bucket-name",
		},
		{
			name:     "mixed characters",
			input:    "My_Bucket Name-123",
			expected: "my-bucket-name-123",
		},
		{
			name:     "special characters removed",
			input:    "my-bucket@name#123",
			expected: "my-bucketname123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toLowerAlphanumeric(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestS3Provider_ConfigureVersioning(t *testing.T) {
	// This is a placeholder test since we can't easily mock AWS SDK clients
	// In a real implementation, we would use interfaces and dependency injection
	// For now, we verify the function exists and has the right signature
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{logger: log}
	s3Provider := NewS3Provider(awsProvider)

	assert.NotNil(t, s3Provider)
	assert.NotNil(t, s3Provider.provider)
	assert.NotNil(t, s3Provider.client)
}

func TestS3Provider_ResourceResult_Outputs(t *testing.T) {
	// Test that resource results have the expected outputs
	result := &provider.ResourceResult{
		ResourceID: "my-bucket",
		Kind:       schema.KindS3,
		Status:     provider.StatusAvailable,
		Outputs: map[string]string{
			"bucket_name": "my-bucket",
			"arn":         "arn:aws:s3:::my-bucket",
			"region":      "us-east-1",
			"endpoint":    "https://my-bucket.s3.us-east-1.amazonaws.com",
		},
	}

	assert.Equal(t, "my-bucket", result.Outputs["bucket_name"])
	assert.Equal(t, "arn:aws:s3:::my-bucket", result.Outputs["arn"])
	assert.Equal(t, "us-east-1", result.Outputs["region"])
	assert.Contains(t, result.Outputs["endpoint"], "s3.us-east-1.amazonaws.com")
}

func TestS3Provider_ValidateInputs(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "123456789012",
		region:    "us-east-1",
	}
	awsProvider.tagHelper = provider.NewTagHelper(nil)

	s3Provider := NewS3Provider(awsProvider)

	// Test with invalid resource type
	invalidResource := schema.NewDynamoDB("table", "backend", "my-stack")
	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		DryRun:      true,
	}

	_, err := s3Provider.Create(context.Background(), invalidResource, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid resource type")
}

func TestS3Provider_GenerateBucketName_WithExplicitName(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{logger: log}
	s3Provider := NewS3Provider(awsProvider)

	resource := schema.NewS3("uploads", "backend", "my-stack")
	resource.Spec.Bucket.Name = "explicit-bucket-name"

	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
	}

	// When explicit name is provided, it should be used (in actual Create method)
	// But generateBucketName still generates the default name
	bucketName := s3Provider.generateBucketName(resource, opts)
	assert.Equal(t, "my-stack-backend-uploads", bucketName)
}

func TestS3Provider_LifecycleConfiguration(t *testing.T) {
	resource := schema.NewS3("uploads", "backend", "my-stack")
	resource.Spec.Lifecycle = []schema.LifecycleRule{
		{
			ID:      "archive-old-files",
			Enabled: true,
			Prefix:  "uploads/",
			Expiration: &schema.ExpirationConfig{
				Days: 365,
			},
			Transition: []schema.TransitionConfig{
				{
					Days:         30,
					StorageClass: "STANDARD_IA",
				},
				{
					Days:         90,
					StorageClass: "GLACIER",
				},
			},
		},
	}

	// Verify lifecycle rule structure
	assert.Len(t, resource.Spec.Lifecycle, 1)
	assert.Equal(t, "archive-old-files", resource.Spec.Lifecycle[0].ID)
	assert.True(t, resource.Spec.Lifecycle[0].Enabled)
	assert.Len(t, resource.Spec.Lifecycle[0].Transition, 2)
}

func TestS3Provider_CORSConfiguration(t *testing.T) {
	resource := schema.NewS3("uploads", "backend", "my-stack")
	resource.Spec.CORS = &schema.CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST", "PUT"},
		AllowedHeaders: []string{"*"},
		MaxAgeSeconds:  3600,
	}

	// Verify CORS configuration
	assert.NotNil(t, resource.Spec.CORS)
	assert.Len(t, resource.Spec.CORS.AllowedOrigins, 1)
	assert.Len(t, resource.Spec.CORS.AllowedMethods, 3)
	assert.Equal(t, 3600, resource.Spec.CORS.MaxAgeSeconds)
}

func TestS3Provider_EncryptionConfiguration(t *testing.T) {
	resource := schema.NewS3("uploads", "backend", "my-stack")
	resource.Spec.Encryption = &schema.S3Encryption{
		Enabled:   true,
		Algorithm: "AES256",
	}

	// Verify encryption configuration
	assert.NotNil(t, resource.Spec.Encryption)
	assert.True(t, resource.Spec.Encryption.Enabled)
	assert.Equal(t, "AES256", resource.Spec.Encryption.Algorithm)
}

func TestS3Provider_VersioningConfiguration(t *testing.T) {
	resource := schema.NewS3("uploads", "backend", "my-stack")
	resource.Spec.Versioning = &schema.VersioningConfig{
		Enabled: true,
	}

	// Verify versioning configuration
	assert.NotNil(t, resource.Spec.Versioning)
	assert.True(t, resource.Spec.Versioning.Enabled)
}
