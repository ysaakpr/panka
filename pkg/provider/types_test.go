package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/panka/pkg/parser/schema"
)

func TestNewTagHelper(t *testing.T) {
	defaultTags := map[string]string{
		"environment": "production",
		"team":        "platform",
	}
	
	helper := NewTagHelper(defaultTags)
	
	assert.NotNil(t, helper)
	assert.Equal(t, "production", helper.DefaultTags["environment"])
	assert.Equal(t, "platform", helper.DefaultTags["team"])
}

func TestNewTagHelper_NilDefaults(t *testing.T) {
	helper := NewTagHelper(nil)
	
	assert.NotNil(t, helper)
	assert.NotNil(t, helper.DefaultTags)
	assert.Len(t, helper.DefaultTags, 0)
}

func TestTagHelper_BuildTags_DefaultTags(t *testing.T) {
	helper := NewTagHelper(map[string]string{
		"environment": "production",
		"team":        "platform",
	})
	
	resource := schema.NewS3("uploads", "backend", "my-stack")
	opts := &ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
	}
	
	tags := helper.BuildTags(opts, resource)
	
	// Check default tags
	assert.Equal(t, "production", tags["environment"])
	assert.Equal(t, "platform", tags["team"])
}

func TestTagHelper_BuildTags_StandardTags(t *testing.T) {
	helper := NewTagHelper(nil)
	
	resource := schema.NewS3("uploads", "backend", "my-stack")
	opts := &ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		TenantID:    "tenant-123",
	}
	
	tags := helper.BuildTags(opts, resource)
	
	// Check standard panka tags
	assert.Equal(t, "tenant-123", tags["panka:tenant"])
	assert.Equal(t, "my-stack", tags["panka:stack"])
	assert.Equal(t, "backend", tags["panka:service"])
	assert.Equal(t, "uploads", tags["panka:resource"])
	assert.Equal(t, string(schema.KindS3), tags["panka:kind"])
	assert.Equal(t, "true", tags["panka:managed"])
	assert.Equal(t, "v1", tags["panka:version"])
}

func TestTagHelper_BuildTags_ResourceLabels(t *testing.T) {
	helper := NewTagHelper(nil)
	
	resource := schema.NewS3("uploads", "backend", "my-stack")
	resource.Metadata.Labels = map[string]string{
		"custom-label-1": "value-1",
		"custom-label-2": "value-2",
	}
	
	opts := &ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
	}
	
	tags := helper.BuildTags(opts, resource)
	
	// Check custom labels are included
	assert.Equal(t, "value-1", tags["custom-label-1"])
	assert.Equal(t, "value-2", tags["custom-label-2"])
}

func TestTagHelper_BuildTags_CustomTags(t *testing.T) {
	helper := NewTagHelper(nil)
	
	resource := schema.NewS3("uploads", "backend", "my-stack")
	opts := &ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		Tags: map[string]string{
			"cost-center": "engineering",
			"project":     "platform",
		},
	}
	
	tags := helper.BuildTags(opts, resource)
	
	// Check custom tags from options
	assert.Equal(t, "engineering", tags["cost-center"])
	assert.Equal(t, "platform", tags["project"])
}

func TestTagHelper_BuildTags_TagPriority(t *testing.T) {
	// Test that tags are merged in the correct priority order:
	// Default tags < Standard tags < Resource labels < Custom tags
	
	helper := NewTagHelper(map[string]string{
		"priority": "default",
		"environment": "default-env",
	})
	
	resource := schema.NewS3("uploads", "backend", "my-stack")
	resource.Metadata.Labels = map[string]string{
		"priority": "label",
	}
	
	opts := &ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		Tags: map[string]string{
			"priority": "custom",
		},
	}
	
	tags := helper.BuildTags(opts, resource)
	
	// Custom tags should have highest priority
	assert.Equal(t, "custom", tags["priority"])
	assert.Equal(t, "default-env", tags["environment"])
}

func TestTagHelper_BuildTags_WithoutTenant(t *testing.T) {
	helper := NewTagHelper(nil)
	
	resource := schema.NewS3("uploads", "backend", "my-stack")
	opts := &ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		// TenantID is empty
	}
	
	tags := helper.BuildTags(opts, resource)
	
	// Tenant tag should not be present
	_, hasTenant := tags["panka:tenant"]
	assert.False(t, hasTenant)
	
	// But other tags should still be present
	assert.Equal(t, "my-stack", tags["panka:stack"])
}

func TestFormatARN(t *testing.T) {
	tests := []struct {
		name      string
		partition string
		service   string
		region    string
		accountID string
		resource  string
		expected  string
	}{
		{
			name:      "S3 bucket",
			partition: "aws",
			service:   "s3",
			region:    "",
			accountID: "",
			resource:  "my-bucket",
			expected:  "arn:aws:s3:::my-bucket",
		},
		{
			name:      "DynamoDB table",
			partition: "aws",
			service:   "dynamodb",
			region:    "us-east-1",
			accountID: "123456789012",
			resource:  "table/my-table",
			expected:  "arn:aws:dynamodb:us-east-1:123456789012:table/my-table",
		},
		{
			name:      "SQS queue",
			partition: "aws",
			service:   "sqs",
			region:    "us-west-2",
			accountID: "123456789012",
			resource:  "my-queue",
			expected:  "arn:aws:sqs:us-west-2:123456789012:my-queue",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arn := FormatARN(tt.partition, tt.service, tt.region, tt.accountID, tt.resource)
			assert.Equal(t, tt.expected, arn)
		})
	}
}

func TestResourceStatus_Constants(t *testing.T) {
	// Verify all status constants are defined
	statuses := []ResourceStatus{
		StatusPending,
		StatusCreating,
		StatusAvailable,
		StatusUpdating,
		StatusDeleting,
		StatusDeleted,
		StatusFailed,
		StatusUnknown,
	}
	
	assert.Len(t, statuses, 8)
	assert.Equal(t, "pending", string(StatusPending))
	assert.Equal(t, "available", string(StatusAvailable))
	assert.Equal(t, "deleted", string(StatusDeleted))
}

func TestResourceOptions_Defaults(t *testing.T) {
	opts := &ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
	}
	
	assert.Equal(t, "my-stack", opts.StackName)
	assert.Equal(t, "backend", opts.ServiceName)
	assert.False(t, opts.DryRun)
	assert.False(t, opts.Force)
	assert.Empty(t, opts.TenantID)
}

func TestResourceResult_Structure(t *testing.T) {
	result := &ResourceResult{
		ResourceID: "test-resource",
		Kind:       schema.KindS3,
		Status:     StatusAvailable,
		Outputs: map[string]string{
			"key": "value",
		},
		Metadata: map[string]string{
			"provider": "aws",
		},
	}
	
	assert.Equal(t, "test-resource", result.ResourceID)
	assert.Equal(t, schema.KindS3, result.Kind)
	assert.Equal(t, StatusAvailable, result.Status)
	assert.Equal(t, "value", result.Outputs["key"])
	assert.Equal(t, "aws", result.Metadata["provider"])
}

