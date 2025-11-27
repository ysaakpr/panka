package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"go.uber.org/zap"
)

// Provider implements the AWS cloud provider
type Provider struct {
	logger *logger.Logger
	
	// AWS configuration
	config    aws.Config
	awsConfig *provider.Config
	
	// Account information
	accountID string
	region    string
	
	// Resource providers
	resourceProviders map[schema.Kind]provider.ResourceProvider
	
	// Tag helper
	tagHelper *provider.TagHelper
	
	// Initialized flag
	initialized bool
}

// NewProvider creates a new AWS provider
func NewProvider() *Provider {
	log, _ := logger.NewDevelopment()
	return &Provider{
		logger:            log,
		resourceProviders: make(map[schema.Kind]provider.ResourceProvider),
		initialized:       false,
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "aws"
}

// Initialize initializes the AWS provider
func (p *Provider) Initialize(ctx context.Context, cfg *provider.Config) error {
	if cfg == nil {
		return provider.ErrInvalidConfiguration
	}
	
	if cfg.Name != "aws" {
		return &provider.ProviderError{
			Provider:  "aws",
			Operation: "initialize",
			Message:   fmt.Sprintf("invalid provider name: %s", cfg.Name),
		}
	}
	
	p.logger.Info("Initializing AWS provider", zap.String("region", cfg.Region))
	
	// Store configuration
	p.awsConfig = cfg
	p.region = cfg.Region
	
	// Load AWS SDK configuration
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return &provider.ProviderError{
			Provider:  "aws",
			Operation: "initialize",
			Message:   "failed to load AWS configuration",
			Cause:     err,
		}
	}
	
	p.config = awsConfig
	
	// Validate credentials by calling STS GetCallerIdentity
	stsClient := sts.NewFromConfig(p.config)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return &provider.ProviderError{
			Provider:  "aws",
			Operation: "initialize",
			Message:   "failed to validate AWS credentials",
			Cause:     err,
		}
	}
	
	p.accountID = *identity.Account
	
	p.logger.Info("AWS credentials validated",
		zap.String("account_id", p.accountID),
		zap.String("user_arn", *identity.Arn),
	)
	
	// Initialize tag helper
	p.tagHelper = provider.NewTagHelper(cfg.DefaultTags)
	
	// Register resource providers
	p.registerResourceProviders()
	
	p.initialized = true
	
	p.logger.Info("AWS provider initialized successfully")
	
	return nil
}

// ValidateCredentials validates AWS credentials
func (p *Provider) ValidateCredentials(ctx context.Context) error {
	if !p.initialized {
		return provider.ErrProviderNotInitialized
	}
	
	stsClient := sts.NewFromConfig(p.config)
	_, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return &provider.ProviderError{
			Provider:  "aws",
			Operation: "validate_credentials",
			Message:   "AWS credentials are invalid",
			Cause:     err,
		}
	}
	
	return nil
}

// GetResourceProvider returns a provider for a specific resource kind
func (p *Provider) GetResourceProvider(kind schema.Kind) (provider.ResourceProvider, error) {
	if !p.initialized {
		return nil, provider.ErrProviderNotInitialized
	}
	
	resourceProvider, exists := p.resourceProviders[kind]
	if !exists {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "get_resource_provider",
			Message:   fmt.Sprintf("unsupported resource kind: %s", kind),
		}
	}
	
	return resourceProvider, nil
}

// Close cleans up provider resources
func (p *Provider) Close() error {
	p.logger.Info("Closing AWS provider")
	p.initialized = false
	return nil
}

// registerResourceProviders registers all supported resource providers
func (p *Provider) registerResourceProviders() {
	// Register S3 provider
	p.resourceProviders[schema.KindS3] = NewS3Provider(p)
	
	// Register DynamoDB provider
	p.resourceProviders[schema.KindDynamoDB] = NewDynamoDBProvider(p)
	
	// Register SQS provider
	p.resourceProviders[schema.KindSQS] = NewSQSProvider(p)
	
	// Register SNS provider
	p.resourceProviders[schema.KindSNS] = NewSNSProvider(p)
	
	// Register RDS provider
	p.resourceProviders[schema.KindRDS] = NewRDSProvider(p)
	
	// Register MicroService provider (ECS/Fargate)
	p.resourceProviders[schema.KindMicroService] = NewECSProvider(p)
	
	p.logger.Info("Registered AWS resource providers",
		zap.Int("count", len(p.resourceProviders)),
	)
}

// GetAccountID returns the AWS account ID
func (p *Provider) GetAccountID() string {
	return p.accountID
}

// GetRegion returns the AWS region
func (p *Provider) GetRegion() string {
	return p.region
}

// GetConfig returns the AWS SDK configuration
func (p *Provider) GetConfig() aws.Config {
	return p.config
}

// GetTagHelper returns the tag helper
func (p *Provider) GetTagHelper() *provider.TagHelper {
	return p.tagHelper
}

// GetLogger returns the logger
func (p *Provider) GetLogger() *logger.Logger {
	return p.logger
}

