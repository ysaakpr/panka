package tenant

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/yourusername/panka/internal/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// S3RegistryBackend implements RegistryBackend using S3
type S3RegistryBackend struct {
	client *s3.Client
	bucket string
	region string
	logger *logger.Logger
}

// NewS3RegistryBackend creates a new S3 registry backend
func NewS3RegistryBackend(bucket, region string) (*S3RegistryBackend, error) {
	log := logger.Global()
	
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	return &S3RegistryBackend{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
		region: region,
		logger: log,
	}, nil
}

// LoadRegistry loads the tenant registry from S3
func (sb *S3RegistryBackend) LoadRegistry(ctx context.Context) (*Registry, error) {
	sb.logger.Debug("Loading registry from S3",
		zap.String("bucket", sb.bucket),
		zap.String("key", "tenants.yaml"),
	)
	
	// Get object from S3
	result, err := sb.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(sb.bucket),
		Key:    aws.String("tenants.yaml"),
	})
	if err != nil {
		// Check if registry doesn't exist yet
		return sb.createInitialRegistry(ctx)
	}
	defer result.Body.Close()
	
	// Parse YAML
	var registry Registry
	decoder := yaml.NewDecoder(result.Body)
	if err := decoder.Decode(&registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}
	
	sb.logger.Info("Registry loaded",
		zap.Int("tenants", len(registry.Tenants)),
	)
	
	return &registry, nil
}

// SaveRegistry saves the tenant registry to S3
func (sb *S3RegistryBackend) SaveRegistry(ctx context.Context, registry *Registry) error {
	sb.logger.Debug("Saving registry to S3",
		zap.String("bucket", sb.bucket),
		zap.Int("tenants", len(registry.Tenants)),
	)
	
	// Marshal to YAML
	data, err := yaml.Marshal(registry)
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}
	
	// Put object to S3
	_, err = sb.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(sb.bucket),
		Key:         aws.String("tenants.yaml"),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/x-yaml"),
	})
	if err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}
	
	sb.logger.Info("Registry saved")
	
	return nil
}

// createInitialRegistry creates an initial empty registry
func (sb *S3RegistryBackend) createInitialRegistry(ctx context.Context) (*Registry, error) {
	sb.logger.Info("Creating initial registry")
	
	registry := &Registry{
		Version: "v1",
		Metadata: RegistryMetadata{
			Created: time.Now(),
			Updated: time.Now(),
			Bucket:  sb.bucket,
			Region:  sb.region,
		},
		Config: RegistryConfig{
			LockTable:      "", // Will be set during init
			DefaultVersion: "v1",
		},
		Tenants: []Tenant{},
	}
	
	// Save to S3
	if err := sb.SaveRegistry(ctx, registry); err != nil {
		return nil, fmt.Errorf("failed to create initial registry: %w", err)
	}
	
	return registry, nil
}

// CreateTenantDirectory creates the S3 directory structure for a tenant
func (sb *S3RegistryBackend) CreateTenantDirectory(ctx context.Context, tenant *Tenant) error {
	sb.logger.Info("Creating tenant directory structure",
		zap.String("tenant", tenant.ID),
		zap.String("prefix", tenant.Storage.Path),
	)
	
	// Create tenant.yaml
	tenantYAML := map[string]interface{}{
		"tenant": map[string]interface{}{
			"id":          tenant.ID,
			"displayName": tenant.DisplayName,
			"version":     tenant.Storage.Version,
			"created":     tenant.Created.Format(time.RFC3339),
		},
		"storage": map[string]interface{}{
			"bucket": sb.bucket,
			"prefix": tenant.Storage.Path,
		},
		"locks": map[string]interface{}{
			"prefix": tenant.Locks.Prefix,
		},
		"config": map[string]interface{}{
			"environments": []string{"production", "staging", "development"},
			"regions":      []string{sb.region},
			"policies": map[string]bool{
				"requireApprovalForProduction": true,
				"enableDriftDetection":         true,
				"enableCostTracking":           tenant.Limits.CostTracking,
			},
		},
	}
	
	data, err := yaml.Marshal(tenantYAML)
	if err != nil {
		return fmt.Errorf("failed to marshal tenant.yaml: %w", err)
	}
	
	// Upload tenant.yaml
	key := fmt.Sprintf("%s/tenant.yaml", tenant.Storage.Prefix)
	_, err = sb.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(sb.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/x-yaml"),
	})
	if err != nil {
		return fmt.Errorf("failed to create tenant.yaml: %w", err)
	}
	
	sb.logger.Info("Tenant directory created", zap.String("tenant", tenant.ID))

	return nil
}

// LoadTenantConfig loads the full tenant configuration from the registry
func (sb *S3RegistryBackend) LoadTenantConfig(ctx context.Context, tenantID string) (*Tenant, error) {
	sb.logger.Debug("Loading tenant config",
		zap.String("tenant_id", tenantID),
	)

	// Load registry
	registry, err := sb.LoadRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Find tenant
	for _, t := range registry.Tenants {
		if t.ID == tenantID {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("tenant not found: %s", tenantID)
}

