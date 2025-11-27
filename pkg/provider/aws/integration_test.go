// +build integration

package aws

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/panka/internal/logger"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
)

// Integration tests require LocalStack to be running
// Run with: go test -tags=integration ./pkg/provider/aws/...
//
// Start LocalStack with:
// docker-compose -f test/docker-compose.localstack.yml up -d

func getLocalStackConfig(t *testing.T) aws.Config {
	endpoint := os.Getenv("LOCALSTACK_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           endpoint,
					SigningRegion: region,
				}, nil
			},
		)),
	)
	require.NoError(t, err)
	
	return cfg
}

func TestIntegration_S3Provider_CreateAndRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "000000000000", // LocalStack default
		region:    "us-east-1",
	}
	awsProvider.cfg = getLocalStackConfig(t)
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	awsProvider.initialized = true
	
	s3Provider := NewS3Provider(awsProvider)

	resource := schema.NewS3("integration-test-bucket", "backend", "test-stack")
	resource.Spec.Bucket.ACL = "private"

	opts := &provider.ResourceOptions{
		StackName:   "test-stack",
		ServiceName: "backend",
	}

	ctx := context.Background()

	// Create
	result, err := s3Provider.Create(ctx, resource, opts)
	require.NoError(t, err)
	assert.Equal(t, schema.KindS3, result.Kind)
	assert.Equal(t, provider.StatusAvailable, result.Status)

	bucketName := result.Outputs["bucket_name"]
	assert.NotEmpty(t, bucketName)

	// Wait a bit for eventual consistency
	time.Sleep(1 * time.Second)

	// Read
	readResult, err := s3Provider.Read(ctx, bucketName, opts)
	require.NoError(t, err)
	assert.Equal(t, bucketName, readResult.ResourceID)
	assert.Equal(t, provider.StatusAvailable, readResult.Status)

	// Exists
	exists, err := s3Provider.Exists(ctx, bucketName, opts)
	require.NoError(t, err)
	assert.True(t, exists)

	// Clean up
	_, err = s3Provider.Delete(ctx, bucketName, opts)
	require.NoError(t, err)

	// Verify deletion
	time.Sleep(1 * time.Second)
	exists, err = s3Provider.Exists(ctx, bucketName, opts)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestIntegration_DynamoDBProvider_CreateAndRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "000000000000",
		region:    "us-east-1",
	}
	awsProvider.cfg = getLocalStackConfig(t)
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	awsProvider.initialized = true
	
	dynamoProvider := NewDynamoDBProvider(awsProvider)

	resource := schema.NewDynamoDB("integration-test-table", "backend", "test-stack")
	resource.Spec.BillingMode = "PAY_PER_REQUEST"
	resource.Spec.HashKey = schema.AttributeDefinition{
		Name: "id",
		Type: "S",
	}

	opts := &provider.ResourceOptions{
		StackName:   "test-stack",
		ServiceName: "backend",
	}

	ctx := context.Background()

	// Create
	result, err := dynamoProvider.Create(ctx, resource, opts)
	require.NoError(t, err)
	assert.Equal(t, schema.KindDynamoDB, result.Kind)

	tableName := result.Outputs["table_name"]
	assert.NotEmpty(t, tableName)

	// Wait for table to be active
	time.Sleep(2 * time.Second)

	// Read
	readResult, err := dynamoProvider.Read(ctx, tableName, opts)
	require.NoError(t, err)
	assert.Equal(t, tableName, readResult.ResourceID)

	// Exists
	exists, err := dynamoProvider.Exists(ctx, tableName, opts)
	require.NoError(t, err)
	assert.True(t, exists)

	// Clean up
	_, err = dynamoProvider.Delete(ctx, tableName, opts)
	require.NoError(t, err)
}

func TestIntegration_SQSProvider_CreateAndRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "000000000000",
		region:    "us-east-1",
	}
	awsProvider.cfg = getLocalStackConfig(t)
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	awsProvider.initialized = true
	
	sqsProvider := NewSQSProvider(awsProvider)

	resource := schema.NewSQS("integration-test-queue", "backend", "test-stack")
	resource.Spec.Type = "standard"
	resource.Spec.VisibilityTimeout = 30

	opts := &provider.ResourceOptions{
		StackName:   "test-stack",
		ServiceName: "backend",
	}

	ctx := context.Background()

	// Create
	result, err := sqsProvider.Create(ctx, resource, opts)
	require.NoError(t, err)
	assert.Equal(t, schema.KindSQS, result.Kind)

	queueURL := result.Outputs["queue_url"]
	assert.NotEmpty(t, queueURL)

	// Wait a bit
	time.Sleep(1 * time.Second)

	// Read
	readResult, err := sqsProvider.Read(ctx, queueURL, opts)
	require.NoError(t, err)
	assert.Equal(t, queueURL, readResult.ResourceID)

	// Exists
	exists, err := sqsProvider.Exists(ctx, queueURL, opts)
	require.NoError(t, err)
	assert.True(t, exists)

	// Clean up
	_, err = sqsProvider.Delete(ctx, queueURL, opts)
	require.NoError(t, err)
}

func TestIntegration_SNSProvider_CreateAndRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, _ := logger.NewDevelopment()
	awsProvider := &Provider{
		logger:    log,
		accountID: "000000000000",
		region:    "us-east-1",
	}
	awsProvider.cfg = getLocalStackConfig(t)
	awsProvider.tagHelper = provider.NewTagHelper(nil)
	awsProvider.initialized = true
	
	snsProvider := NewSNSProvider(awsProvider)

	resource := schema.NewSNS("integration-test-topic", "backend", "test-stack")
	resource.Spec.DisplayName = "Integration Test Topic"

	opts := &provider.ResourceOptions{
		StackName:   "test-stack",
		ServiceName: "backend",
	}

	ctx := context.Background()

	// Create
	result, err := snsProvider.Create(ctx, resource, opts)
	require.NoError(t, err)
	assert.Equal(t, schema.KindSNS, result.Kind)

	topicARN := result.Outputs["arn"]
	assert.NotEmpty(t, topicARN)

	// Wait a bit
	time.Sleep(1 * time.Second)

	// Read
	readResult, err := snsProvider.Read(ctx, topicARN, opts)
	require.NoError(t, err)
	assert.Equal(t, topicARN, readResult.ResourceID)

	// Exists
	exists, err := snsProvider.Exists(ctx, topicARN, opts)
	require.NoError(t, err)
	assert.True(t, exists)

	// Clean up
	_, err = snsProvider.Delete(ctx, topicARN, opts)
	require.NoError(t, err)
}

