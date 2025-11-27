package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/yourusername/panka/pkg/parser/schema"
	"github.com/yourusername/panka/pkg/provider"
	"go.uber.org/zap"
)

// DynamoDBProvider implements DynamoDB table management
type DynamoDBProvider struct {
	provider *Provider
	client   *dynamodb.Client
}

// NewDynamoDBProvider creates a new DynamoDB provider
func NewDynamoDBProvider(p *Provider) *DynamoDBProvider {
	return &DynamoDBProvider{
		provider: p,
		client:   dynamodb.NewFromConfig(p.GetConfig()),
	}
}

// Create creates a new DynamoDB table
func (dp *DynamoDBProvider) Create(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	dynamoResource, ok := resource.(*schema.DynamoDB)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "create",
			Message:   "invalid resource type for DynamoDB provider",
		}
	}

	dp.provider.GetLogger().Info("Creating DynamoDB table",
		zap.String("name", dynamoResource.Metadata.Name),
	)

	// Generate table name if not specified
	tableName := dynamoResource.Spec.TableName
	if tableName == "" {
		tableName = dp.generateTableName(dynamoResource, opts)
	}

	// Build tags
	tags := dp.provider.GetTagHelper().BuildTags(opts, resource)
	awsTags := make([]types.Tag, 0, len(tags))
	for k, v := range tags {
		awsTags = append(awsTags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	// Build attribute definitions
	attributeDefs := []types.AttributeDefinition{
		{
			AttributeName: aws.String(dynamoResource.Spec.HashKey.Name),
			AttributeType: types.ScalarAttributeType(dynamoResource.Spec.HashKey.Type),
		},
	}

	// Build key schema
	keySchema := []types.KeySchemaElement{
		{
			AttributeName: aws.String(dynamoResource.Spec.HashKey.Name),
			KeyType:       types.KeyTypeHash,
		},
	}

	// Add range key if specified
	if dynamoResource.Spec.RangeKey != nil {
		attributeDefs = append(attributeDefs, types.AttributeDefinition{
			AttributeName: aws.String(dynamoResource.Spec.RangeKey.Name),
			AttributeType: types.ScalarAttributeType(dynamoResource.Spec.RangeKey.Type),
		})
		keySchema = append(keySchema, types.KeySchemaElement{
			AttributeName: aws.String(dynamoResource.Spec.RangeKey.Name),
			KeyType:       types.KeyTypeRange,
		})
	}

	// Create table input
	createInput := &dynamodb.CreateTableInput{
		TableName:            aws.String(tableName),
		AttributeDefinitions: attributeDefs,
		KeySchema:            keySchema,
		BillingMode:          types.BillingMode(dynamoResource.Spec.BillingMode),
		Tags:                 awsTags,
	}

	// Add provisioned throughput if PROVISIONED billing mode
	if dynamoResource.Spec.BillingMode == "PROVISIONED" {
		createInput.ProvisionedThroughput = &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(int64(dynamoResource.Spec.ReadCapacity)),
			WriteCapacityUnits: aws.Int64(int64(dynamoResource.Spec.WriteCapacity)),
		}
	}

	// Add GSIs if specified
	if len(dynamoResource.Spec.GlobalSecondaryIndexes) > 0 {
		gsis := make([]types.GlobalSecondaryIndex, 0, len(dynamoResource.Spec.GlobalSecondaryIndexes))
		
		for _, gsi := range dynamoResource.Spec.GlobalSecondaryIndexes {
			// Add GSI attribute definitions if not already present
			gsiAttrDef := types.AttributeDefinition{
				AttributeName: aws.String(gsi.HashKey.Name),
				AttributeType: types.ScalarAttributeType(gsi.HashKey.Type),
			}
			if !containsAttributeDef(attributeDefs, gsiAttrDef) {
				createInput.AttributeDefinitions = append(createInput.AttributeDefinitions, gsiAttrDef)
			}

			gsiKeySchema := []types.KeySchemaElement{
				{
					AttributeName: aws.String(gsi.HashKey.Name),
					KeyType:       types.KeyTypeHash,
				},
			}

			if gsi.RangeKey != nil {
				rangeAttrDef := types.AttributeDefinition{
					AttributeName: aws.String(gsi.RangeKey.Name),
					AttributeType: types.ScalarAttributeType(gsi.RangeKey.Type),
				}
				if !containsAttributeDef(createInput.AttributeDefinitions, rangeAttrDef) {
					createInput.AttributeDefinitions = append(createInput.AttributeDefinitions, rangeAttrDef)
				}
				gsiKeySchema = append(gsiKeySchema, types.KeySchemaElement{
					AttributeName: aws.String(gsi.RangeKey.Name),
					KeyType:       types.KeyTypeRange,
				})
			}

			gsiDef := types.GlobalSecondaryIndex{
				IndexName: aws.String(gsi.Name),
				KeySchema: gsiKeySchema,
				Projection: &types.Projection{
					ProjectionType: types.ProjectionType(gsi.Projection),
				},
			}

			if dynamoResource.Spec.BillingMode == "PROVISIONED" {
				gsiDef.ProvisionedThroughput = &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(int64(gsi.ReadCapacity)),
					WriteCapacityUnits: aws.Int64(int64(gsi.WriteCapacity)),
				}
			}

			gsis = append(gsis, gsiDef)
		}

		createInput.GlobalSecondaryIndexes = gsis
	}

	if !opts.DryRun {
		// Create table
		output, err := dp.client.CreateTable(ctx, createInput)
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "create",
				ResourceID: tableName,
				Message:    "failed to create DynamoDB table",
				Cause:      err,
			}
		}

		// Wait for table to be active
		waiter := dynamodb.NewTableExistsWaiter(dp.client)
		if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		}, 5*time.Minute); err != nil {
			dp.provider.GetLogger().Warn("Table created but wait failed", zap.Error(err))
		}

		// Configure TTL if specified
		if dynamoResource.Spec.TTL != nil && dynamoResource.Spec.TTL.Enabled {
			if err := dp.configureTTL(ctx, tableName, dynamoResource.Spec.TTL); err != nil {
				dp.provider.GetLogger().Warn("Failed to configure TTL", zap.Error(err))
			}
		}

		// Enable point-in-time recovery if specified
		if dynamoResource.Spec.PointInTimeRecovery {
			if err := dp.configurePointInTimeRecovery(ctx, tableName, true); err != nil {
				dp.provider.GetLogger().Warn("Failed to enable PITR", zap.Error(err))
			}
		}

		tableARN := *output.TableDescription.TableArn

		result := &provider.ResourceResult{
			ResourceID: tableName,
			Kind:       schema.KindDynamoDB,
			Status:     provider.StatusAvailable,
			Outputs: map[string]string{
				"table_name": tableName,
				"arn":        tableARN,
				"region":     dp.provider.GetRegion(),
			},
			Metadata: map[string]string{
				"provider": "aws",
				"region":   dp.provider.GetRegion(),
			},
			Timestamp: time.Now(),
		}

		dp.provider.GetLogger().Info("DynamoDB table created successfully",
			zap.String("table", tableName),
			zap.String("arn", tableARN),
		)

		return result, nil
	}

	// Dry run result
	return &provider.ResourceResult{
		ResourceID: tableName,
		Kind:       schema.KindDynamoDB,
		Status:     provider.StatusPending,
		Timestamp:  time.Now(),
	}, nil
}

// Read reads the current state of a DynamoDB table
func (dp *DynamoDBProvider) Read(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	output, err := dp.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(resourceID),
	})
	if err != nil {
		return nil, &provider.ProviderError{
			Provider:   "aws",
			Operation:  "read",
			ResourceID: resourceID,
			Message:    "table not found",
			Cause:      err,
		}
	}

	status := provider.StatusAvailable
	if output.Table.TableStatus != types.TableStatusActive {
		status = provider.StatusPending
	}

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindDynamoDB,
		Status:     status,
		Outputs: map[string]string{
			"table_name": resourceID,
			"arn":        *output.Table.TableArn,
			"region":     dp.provider.GetRegion(),
		},
		Timestamp: time.Now(),
	}, nil
}

// Update updates an existing DynamoDB table
func (dp *DynamoDBProvider) Update(ctx context.Context, resource schema.Resource, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	dynamoResource, ok := resource.(*schema.DynamoDB)
	if !ok {
		return nil, &provider.ProviderError{
			Provider:  "aws",
			Operation: "update",
			Message:   "invalid resource type for DynamoDB provider",
		}
	}

	tableName := dynamoResource.Spec.TableName
	if tableName == "" {
		tableName = dp.generateTableName(dynamoResource, opts)
	}

	dp.provider.GetLogger().Info("Updating DynamoDB table", zap.String("table", tableName))

	// Update provisioned throughput if PROVISIONED
	if dynamoResource.Spec.BillingMode == "PROVISIONED" {
		_, err := dp.client.UpdateTable(ctx, &dynamodb.UpdateTableInput{
			TableName: aws.String(tableName),
			ProvisionedThroughput: &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(int64(dynamoResource.Spec.ReadCapacity)),
				WriteCapacityUnits: aws.Int64(int64(dynamoResource.Spec.WriteCapacity)),
			},
		})
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "update",
				ResourceID: tableName,
				Message:    "failed to update table",
				Cause:      err,
			}
		}
	}

	// Update TTL
	if dynamoResource.Spec.TTL != nil {
		if err := dp.configureTTL(ctx, tableName, dynamoResource.Spec.TTL); err != nil {
			dp.provider.GetLogger().Warn("Failed to update TTL", zap.Error(err))
		}
	}

	// Update PITR
	if err := dp.configurePointInTimeRecovery(ctx, tableName, dynamoResource.Spec.PointInTimeRecovery); err != nil {
		dp.provider.GetLogger().Warn("Failed to update PITR", zap.Error(err))
	}

	return dp.Read(ctx, tableName, opts)
}

// Delete deletes a DynamoDB table
func (dp *DynamoDBProvider) Delete(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (*provider.ResourceResult, error) {
	dp.provider.GetLogger().Info("Deleting DynamoDB table", zap.String("table", resourceID))

	if !opts.DryRun {
		_, err := dp.client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
			TableName: aws.String(resourceID),
		})
		if err != nil {
			return nil, &provider.ProviderError{
				Provider:   "aws",
				Operation:  "delete",
				ResourceID: resourceID,
				Message:    "failed to delete table",
				Cause:      err,
			}
		}

		// Wait for table to be deleted
		waiter := dynamodb.NewTableNotExistsWaiter(dp.client)
		if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(resourceID),
		}, 5*time.Minute); err != nil {
			dp.provider.GetLogger().Warn("Table deleted but wait failed", zap.Error(err))
		}
	}

	dp.provider.GetLogger().Info("DynamoDB table deleted successfully", zap.String("table", resourceID))

	return &provider.ResourceResult{
		ResourceID: resourceID,
		Kind:       schema.KindDynamoDB,
		Status:     provider.StatusDeleted,
		Timestamp:  time.Now(),
	}, nil
}

// Exists checks if a DynamoDB table exists
func (dp *DynamoDBProvider) Exists(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (bool, error) {
	_, err := dp.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(resourceID),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// GetOutputs returns the outputs of a DynamoDB table
func (dp *DynamoDBProvider) GetOutputs(ctx context.Context, resourceID string, opts *provider.ResourceOptions) (map[string]string, error) {
	result, err := dp.Read(ctx, resourceID, opts)
	if err != nil {
		return nil, err
	}
	return result.Outputs, nil
}

// Helper functions

func (dp *DynamoDBProvider) generateTableName(resource *schema.DynamoDB, opts *provider.ResourceOptions) string {
	return fmt.Sprintf("%s-%s-%s",
		opts.StackName,
		opts.ServiceName,
		resource.Metadata.Name,
	)
}

func (dp *DynamoDBProvider) configureTTL(ctx context.Context, tableName string, ttlConfig *schema.TTLConfig) error {
	_, err := dp.client.UpdateTimeToLive(ctx, &dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String(tableName),
		TimeToLiveSpecification: &types.TimeToLiveSpecification{
			Enabled:       aws.Bool(ttlConfig.Enabled),
			AttributeName: aws.String(ttlConfig.AttributeName),
		},
	})
	return err
}

func (dp *DynamoDBProvider) configurePointInTimeRecovery(ctx context.Context, tableName string, enabled bool) error {
	_, err := dp.client.UpdateContinuousBackups(ctx, &dynamodb.UpdateContinuousBackupsInput{
		TableName: aws.String(tableName),
		PointInTimeRecoverySpecification: &types.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(enabled),
		},
	})
	return err
}

func containsAttributeDef(defs []types.AttributeDefinition, def types.AttributeDefinition) bool {
	for _, d := range defs {
		if *d.AttributeName == *def.AttributeName {
			return true
		}
	}
	return false
}

