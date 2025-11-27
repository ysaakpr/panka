package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	
	assert.NotNil(t, p)
	assert.Equal(t, "aws", p.Name())
	assert.False(t, p.initialized)
	assert.NotNil(t, p.logger)
	assert.NotNil(t, p.resourceProviders)
}

func TestProvider_Name(t *testing.T) {
	p := NewProvider()
	assert.Equal(t, "aws", p.Name())
}

func TestProvider_GetResourceProvider_NotInitialized(t *testing.T) {
	p := NewProvider()
	
	_, err := p.GetResourceProvider(schema.KindS3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestProvider_GetResourceProvider_UnsupportedKind(t *testing.T) {
	log, _ := logger.NewDevelopment()
	p := &Provider{
		logger:            log,
		accountID:         "123456789012",
		region:            "us-east-1",
		resourceProviders: make(map[schema.Kind]provider.ResourceProvider),
		initialized:       true,
	}
	
	_, err := p.GetResourceProvider(schema.KindLambda)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource kind")
}

func TestProvider_RegisterResourceProviders(t *testing.T) {
	log, _ := logger.NewDevelopment()
	p := &Provider{
		logger:            log,
		resourceProviders: make(map[schema.Kind]provider.ResourceProvider),
	}
	
	// Mock AWS config
	p.accountID = "123456789012"
	p.region = "us-east-1"
	
	p.registerResourceProviders()
	
	// Verify all providers are registered
	assert.Len(t, p.resourceProviders, 6)
	assert.Contains(t, p.resourceProviders, schema.KindS3)
	assert.Contains(t, p.resourceProviders, schema.KindDynamoDB)
	assert.Contains(t, p.resourceProviders, schema.KindSQS)
	assert.Contains(t, p.resourceProviders, schema.KindSNS)
	assert.Contains(t, p.resourceProviders, schema.KindRDS)
	assert.Contains(t, p.resourceProviders, schema.KindMicroService)
}

func TestProvider_GetAccountID(t *testing.T) {
	p := &Provider{
		accountID: "123456789012",
	}
	
	assert.Equal(t, "123456789012", p.GetAccountID())
}

func TestProvider_GetRegion(t *testing.T) {
	p := &Provider{
		region: "us-west-2",
	}
	
	assert.Equal(t, "us-west-2", p.GetRegion())
}

func TestProvider_Close(t *testing.T) {
	log, _ := logger.NewDevelopment()
	p := &Provider{
		logger:      log,
		initialized: true,
	}
	
	err := p.Close()
	assert.NoError(t, err)
	assert.False(t, p.initialized)
}

func TestProviderError(t *testing.T) {
	err := &provider.ProviderError{
		Provider:   "aws",
		Operation:  "create",
		ResourceID: "my-bucket",
		Message:    "failed to create bucket",
		Cause:      assert.AnError,
	}
	
	assert.Contains(t, err.Error(), "aws")
	assert.Contains(t, err.Error(), "create")
	assert.Contains(t, err.Error(), "my-bucket")
	assert.Contains(t, err.Error(), "failed to create bucket")
}

func TestProviderError_WithoutCause(t *testing.T) {
	err := &provider.ProviderError{
		Provider:   "aws",
		Operation:  "delete",
		ResourceID: "my-table",
		Message:    "resource not found",
	}
	
	assert.Contains(t, err.Error(), "aws")
	assert.Contains(t, err.Error(), "resource not found")
	assert.NotContains(t, err.Error(), "caused by")
}

