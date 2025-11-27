package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/panka/pkg/parser/schema"
)

// Provider defines the interface for cloud resource providers
type Provider interface {
	// Name returns the provider name (e.g., "aws", "azure", "gcp")
	Name() string
	
	// Initialize initializes the provider with configuration
	Initialize(ctx context.Context, config *Config) error
	
	// ValidateCredentials validates provider credentials
	ValidateCredentials(ctx context.Context) error
	
	// GetResourceProvider returns a provider for a specific resource kind
	GetResourceProvider(kind schema.Kind) (ResourceProvider, error)
	
	// Close cleans up provider resources
	Close() error
}

// ResourceProvider defines the interface for managing specific resource types
type ResourceProvider interface {
	// Create creates a new resource
	Create(ctx context.Context, resource schema.Resource, opts *ResourceOptions) (*ResourceResult, error)
	
	// Read reads the current state of a resource
	Read(ctx context.Context, resourceID string, opts *ResourceOptions) (*ResourceResult, error)
	
	// Update updates an existing resource
	Update(ctx context.Context, resource schema.Resource, opts *ResourceOptions) (*ResourceResult, error)
	
	// Delete deletes a resource
	Delete(ctx context.Context, resourceID string, opts *ResourceOptions) (*ResourceResult, error)
	
	// Exists checks if a resource exists
	Exists(ctx context.Context, resourceID string, opts *ResourceOptions) (bool, error)
	
	// GetOutputs returns the outputs of a resource (e.g., endpoint, ARN)
	GetOutputs(ctx context.Context, resourceID string, opts *ResourceOptions) (map[string]string, error)
}

// Config holds provider configuration
type Config struct {
	// Provider name (aws, azure, gcp)
	Name string
	
	// Region for resource deployment
	Region string
	
	// Credentials (provider-specific)
	Credentials interface{}
	
	// Tags to apply to all resources
	DefaultTags map[string]string
	
	// Additional provider-specific configuration
	Extra map[string]interface{}
}

// ResourceOptions contains options for resource operations
type ResourceOptions struct {
	// Tenant ID for multi-tenancy
	TenantID string
	
	// Stack name
	StackName string
	
	// Service name
	ServiceName string
	
	// Tags to apply to the resource
	Tags map[string]string
	
	// Timeout for the operation
	Timeout time.Duration
	
	// DryRun simulates the operation without making changes
	DryRun bool
	
	// Force forces the operation even if validation fails
	Force bool
}

// ResourceResult contains the result of a resource operation
type ResourceResult struct {
	// Resource ID (AWS ARN, Azure Resource ID, etc.)
	ResourceID string
	
	// Resource type/kind
	Kind schema.Kind
	
	// Resource status
	Status ResourceStatus
	
	// Outputs (e.g., endpoint, connection string, ARN)
	Outputs map[string]string
	
	// Metadata about the resource
	Metadata map[string]string
	
	// Creation/update timestamp
	Timestamp time.Time
	
	// Error if operation failed
	Error error
}

// ResourceStatus represents the status of a resource
type ResourceStatus string

const (
	// StatusPending indicates resource creation is in progress
	StatusPending ResourceStatus = "pending"
	
	// StatusCreating indicates resource is being created
	StatusCreating ResourceStatus = "creating"
	
	// StatusAvailable indicates resource is ready for use
	StatusAvailable ResourceStatus = "available"
	
	// StatusUpdating indicates resource is being updated
	StatusUpdating ResourceStatus = "updating"
	
	// StatusDeleting indicates resource is being deleted
	StatusDeleting ResourceStatus = "deleting"
	
	// StatusDeleted indicates resource has been deleted
	StatusDeleted ResourceStatus = "deleted"
	
	// StatusFailed indicates operation failed
	StatusFailed ResourceStatus = "failed"
	
	// StatusUnknown indicates status cannot be determined
	StatusUnknown ResourceStatus = "unknown"
)

// Error types for provider operations
type ProviderError struct {
	Provider   string
	Operation  string
	ResourceID string
	Cause      error
	Message    string
}

func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s provider error [%s] on %s: %s (caused by: %v)", 
			e.Provider, e.Operation, e.ResourceID, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s provider error [%s] on %s: %s", 
		e.Provider, e.Operation, e.ResourceID, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// Common errors
var (
	ErrProviderNotInitialized = &ProviderError{Message: "provider not initialized"}
	ErrResourceNotFound       = &ProviderError{Message: "resource not found"}
	ErrResourceAlreadyExists  = &ProviderError{Message: "resource already exists"}
	ErrInvalidConfiguration   = &ProviderError{Message: "invalid configuration"}
	ErrCredentialsInvalid     = &ProviderError{Message: "invalid credentials"}
	ErrOperationTimeout       = &ProviderError{Message: "operation timeout"}
	ErrUnsupportedResource    = &ProviderError{Message: "unsupported resource type"}
)

// TagHelper provides utilities for resource tagging
type TagHelper struct {
	DefaultTags map[string]string
}

// NewTagHelper creates a new tag helper
func NewTagHelper(defaultTags map[string]string) *TagHelper {
	if defaultTags == nil {
		defaultTags = make(map[string]string)
	}
	return &TagHelper{
		DefaultTags: defaultTags,
	}
}

// BuildTags builds the final tag set for a resource
// Priority (lowest to highest): default tags < resource labels < standard tags < custom tags
func (h *TagHelper) BuildTags(opts *ResourceOptions, resource schema.Resource) map[string]string {
	tags := make(map[string]string)
	
	// 1. Add default tags (lowest priority)
	for k, v := range h.DefaultTags {
		tags[k] = v
	}
	
	// 2. Add resource metadata labels
	if resource != nil {
		metadata := resource.GetMetadata()
		for k, v := range metadata.Labels {
			tags[k] = v
		}
	}
	
	// 3. Add standard panka tags (higher priority)
	if opts != nil {
		if opts.TenantID != "" {
			tags["panka:tenant"] = opts.TenantID
		}
		if opts.StackName != "" {
			tags["panka:stack"] = opts.StackName
		}
		if opts.ServiceName != "" {
			tags["panka:service"] = opts.ServiceName
		}
	}
	
	// Add resource-specific standard tags
	if resource != nil {
		metadata := resource.GetMetadata()
		tags["panka:resource"] = metadata.Name
		tags["panka:kind"] = string(resource.GetKind())
	}
	
	// Add management tags
	tags["panka:managed"] = "true"
	tags["panka:version"] = "v1"
	
	// 4. Add custom tags from opts (highest priority - overrides everything)
	if opts != nil && opts.Tags != nil {
		for k, v := range opts.Tags {
			tags[k] = v
		}
	}
	
	return tags
}

// FormatARN formats an AWS ARN
func FormatARN(partition, service, region, accountID, resource string) string {
	return fmt.Sprintf("arn:%s:%s:%s:%s:%s", 
		partition, service, region, accountID, resource)
}

