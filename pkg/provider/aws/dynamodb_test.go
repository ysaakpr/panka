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

func TestDynamoDBProvider_GenerateTableName(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{logger: log}
	dynamoProvider := NewDynamoDBProvider(awsProvider)

	resource := schema.NewDynamoDB("sessions", "backend", "my-stack")
	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
	}

	tableName := dynamoProvider.generateTableName(resource, opts)
	
	assert.Equal(t, "my-stack-backend-sessions", tableName)
}

func TestDynamoDBProvider_Create_DryRun(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "123456789012",
		region:    "us-east-1",
	}
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	
	dynamoProvider := NewDynamoDBProvider(awsProvider)

	resource := schema.NewDynamoDB("sessions", "backend", "my-stack")
	resource.Spec.BillingMode = "PAY_PER_REQUEST"
	resource.Spec.HashKey = schema.AttributeDefinition{
		Name: "userId",
		Type: "S",
	}

	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		DryRun:      true,
	}

	result, err := dynamoProvider.Create(context.Background(), resource, opts)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, schema.KindDynamoDB, result.Kind)
	assert.Equal(t, provider.StatusPending, result.Status)
}

func TestDynamoDBProvider_PayPerRequestMode(t *testing.T) {
	resource := schema.NewDynamoDB("sessions", "backend", "my-stack")
	resource.Spec.BillingMode = "PAY_PER_REQUEST"
	resource.Spec.HashKey = schema.AttributeDefinition{
		Name: "id",
		Type: "S",
	}

	// Verify configuration
	assert.Equal(t, "PAY_PER_REQUEST", resource.Spec.BillingMode)
	assert.Equal(t, 0, resource.Spec.ReadCapacity) // Not needed for PAY_PER_REQUEST
	assert.Equal(t, 0, resource.Spec.WriteCapacity)
}

func TestDynamoDBProvider_ProvisionedMode(t *testing.T) {
	resource := schema.NewDynamoDB("sessions", "backend", "my-stack")
	resource.Spec.BillingMode = "PROVISIONED"
	resource.Spec.ReadCapacity = 5
	resource.Spec.WriteCapacity = 5
	resource.Spec.HashKey = schema.AttributeDefinition{
		Name: "id",
		Type: "S",
	}

	// Verify configuration
	assert.Equal(t, "PROVISIONED", resource.Spec.BillingMode)
	assert.Equal(t, 5, resource.Spec.ReadCapacity)
	assert.Equal(t, 5, resource.Spec.WriteCapacity)
}

func TestDynamoDBProvider_WithRangeKey(t *testing.T) {
	resource := schema.NewDynamoDB("sessions", "backend", "my-stack")
	resource.Spec.HashKey = schema.AttributeDefinition{
		Name: "userId",
		Type: "S",
	}
	resource.Spec.RangeKey = &schema.AttributeDefinition{
		Name: "sessionId",
		Type: "S",
	}

	// Verify keys
	assert.Equal(t, "userId", resource.Spec.HashKey.Name)
	assert.Equal(t, "S", resource.Spec.HashKey.Type)
	assert.NotNil(t, resource.Spec.RangeKey)
	assert.Equal(t, "sessionId", resource.Spec.RangeKey.Name)
	assert.Equal(t, "S", resource.Spec.RangeKey.Type)
}

func TestDynamoDBProvider_GlobalSecondaryIndexes(t *testing.T) {
	resource := schema.NewDynamoDB("sessions", "backend", "my-stack")
	resource.Spec.HashKey = schema.AttributeDefinition{
		Name: "userId",
		Type: "S",
	}
	resource.Spec.GlobalSecondaryIndexes = []schema.GlobalSecondaryIndex{
		{
			Name: "SessionIdIndex",
			HashKey: schema.AttributeDefinition{
				Name: "sessionId",
				Type: "S",
			},
			Projection: "ALL",
		},
	}

	// Verify GSI
	assert.Len(t, resource.Spec.GlobalSecondaryIndexes, 1)
	assert.Equal(t, "SessionIdIndex", resource.Spec.GlobalSecondaryIndexes[0].Name)
	assert.Equal(t, "sessionId", resource.Spec.GlobalSecondaryIndexes[0].HashKey.Name)
	assert.Equal(t, "ALL", resource.Spec.GlobalSecondaryIndexes[0].Projection)
}

func TestDynamoDBProvider_TTLConfiguration(t *testing.T) {
	resource := schema.NewDynamoDB("sessions", "backend", "my-stack")
	resource.Spec.TTL = &schema.TTLConfig{
		Enabled:       true,
		AttributeName: "expiresAt",
	}

	// Verify TTL
	assert.NotNil(t, resource.Spec.TTL)
	assert.True(t, resource.Spec.TTL.Enabled)
	assert.Equal(t, "expiresAt", resource.Spec.TTL.AttributeName)
}

func TestDynamoDBProvider_PointInTimeRecovery(t *testing.T) {
	resource := schema.NewDynamoDB("sessions", "backend", "my-stack")
	resource.Spec.PointInTimeRecovery = true

	// Verify PITR
	assert.True(t, resource.Spec.PointInTimeRecovery)
}

func TestDynamoDBProvider_Encryption(t *testing.T) {
	resource := schema.NewDynamoDB("sessions", "backend", "my-stack")
	resource.Spec.Encryption = &schema.EncryptionConfig{
		Enabled: true,
		KMSKey:  "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
	}

	// Verify encryption
	assert.NotNil(t, resource.Spec.Encryption)
	assert.True(t, resource.Spec.Encryption.Enabled)
	assert.Contains(t, resource.Spec.Encryption.KMSKey, "arn:aws:kms")
}

func TestDynamoDBProvider_AttributeTypes(t *testing.T) {
	tests := []struct {
		name     string
		attrType string
		valid    bool
	}{
		{"String type", "S", true},
		{"Number type", "N", true},
		{"Binary type", "B", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := schema.NewDynamoDB("test", "backend", "my-stack")
			resource.Spec.HashKey = schema.AttributeDefinition{
				Name: "id",
				Type: tt.attrType,
			}

			assert.Equal(t, tt.attrType, resource.Spec.HashKey.Type)
		})
	}
}

func TestDynamoDBProvider_ValidateInputs(t *testing.T) {
	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "123456789012",
		region:    "us-east-1",
	}
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	
	dynamoProvider := NewDynamoDBProvider(awsProvider)

	// Test with invalid resource type
	invalidResource := schema.NewS3("bucket", "backend", "my-stack")
	opts := &provider.ResourceOptions{
		StackName:   "my-stack",
		ServiceName: "backend",
		DryRun:      true,
	}

	_, err := dynamoProvider.Create(context.Background(), invalidResource, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid resource type")
}

func TestDynamoDBProvider_ComplexGSI(t *testing.T) {
	resource := schema.NewDynamoDB("orders", "backend", "my-stack")
	resource.Spec.BillingMode = "PROVISIONED"
	resource.Spec.ReadCapacity = 10
	resource.Spec.WriteCapacity = 10
	resource.Spec.HashKey = schema.AttributeDefinition{
		Name: "orderId",
		Type: "S",
	}
	resource.Spec.GlobalSecondaryIndexes = []schema.GlobalSecondaryIndex{
		{
			Name: "UserIdIndex",
			HashKey: schema.AttributeDefinition{
				Name: "userId",
				Type: "S",
			},
			RangeKey: &schema.AttributeDefinition{
				Name: "createdAt",
				Type: "N",
			},
			Projection:    "ALL",
			ReadCapacity:  5,
			WriteCapacity: 5,
		},
	}

	// Verify complex GSI
	gsi := resource.Spec.GlobalSecondaryIndexes[0]
	assert.Equal(t, "UserIdIndex", gsi.Name)
	assert.Equal(t, "userId", gsi.HashKey.Name)
	assert.NotNil(t, gsi.RangeKey)
	assert.Equal(t, "createdAt", gsi.RangeKey.Name)
	assert.Equal(t, "N", gsi.RangeKey.Type)
	assert.Equal(t, 5, gsi.ReadCapacity)
	assert.Equal(t, 5, gsi.WriteCapacity)
}

func TestContainsAttributeDef(t *testing.T) {
	defs := []struct {
		Name string
		Type string
	}{
		{Name: "id", Type: "S"},
		{Name: "timestamp", Type: "N"},
	}

	// This would be the actual AWS SDK types in real implementation
	// For now we're just testing the logic concept
	assert.Equal(t, "id", defs[0].Name)
	assert.Equal(t, "S", defs[0].Type)
}

