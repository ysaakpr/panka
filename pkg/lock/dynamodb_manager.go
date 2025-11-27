package lock

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DynamoDBManager implements the Manager interface using AWS DynamoDB
type DynamoDBManager struct {
	client    *dynamodb.Client
	tableName string
	logger    *zap.Logger
	config    *Config
}

// DynamoDBConfig holds DynamoDB-specific configuration
type DynamoDBConfig struct {
	Client    *dynamodb.Client
	TableName string
	Logger    *zap.Logger
	Config    *Config
}

// NewDynamoDBManager creates a new DynamoDB lock manager
func NewDynamoDBManager(cfg *DynamoDBConfig) (*DynamoDBManager, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("DynamoDB client is required")
	}
	if cfg.TableName == "" {
		return nil, fmt.Errorf("table name is required")
	}

	// Use defaults if not provided
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	config := cfg.Config
	if config == nil {
		config = DefaultConfig()
	}

	return &DynamoDBManager{
		client:    cfg.Client,
		tableName: cfg.TableName,
		logger:    logger,
		config:    config,
	}, nil
}

// Acquire attempts to acquire a lock
func (m *DynamoDBManager) Acquire(ctx context.Context, key string, ttl time.Duration, owner string) (*Lock, error) {
	lockID := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(ttl)
	expiresAtUnix := expiresAt.Unix()

	m.logger.Info("Attempting to acquire lock",
		zap.String("key", key),
		zap.String("lock_id", lockID),
		zap.String("owner", owner),
		zap.Duration("ttl", ttl),
	)

	// Try to put item with condition that it doesn't exist or is expired
	_, err := m.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(m.tableName),
		Item: map[string]types.AttributeValue{
			"lockKey":    &types.AttributeValueMemberS{Value: key},
			"lockID":     &types.AttributeValueMemberS{Value: lockID},
			"owner":      &types.AttributeValueMemberS{Value: owner},
			"acquiredAt": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Unix(), 10)},
			"expiresAt":  &types.AttributeValueMemberN{Value: strconv.FormatInt(expiresAtUnix, 10)},
			"ttl":        &types.AttributeValueMemberN{Value: strconv.FormatInt(int64(ttl.Seconds()), 10)},
		},
		ConditionExpression: aws.String("attribute_not_exists(lockKey) OR expiresAt < :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":now": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Unix(), 10)},
		},
	})

	if err != nil {
		// Check if condition failed (lock already held)
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			// Lock is already held, get info about who holds it
			info, _ := m.Get(ctx, key)
			if info != nil {
				m.logger.Warn("Lock already held",
					zap.String("key", key),
					zap.String("held_by", info.Owner),
					zap.Duration("age", info.Age),
				)
			}
			return nil, ErrLockAlreadyHeld
		}
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	lock := &Lock{
		Key:        key,
		ID:         lockID,
		Owner:      owner,
		AcquiredAt: now,
		ExpiresAt:  expiresAt,
		TTL:        int64(ttl.Seconds()),
		Metadata:   make(map[string]string),
	}

	m.logger.Info("Lock acquired successfully",
		zap.String("key", key),
		zap.String("lock_id", lockID),
		zap.String("owner", owner),
	)

	return lock, nil
}

// Refresh refreshes an existing lock (heartbeat)
func (m *DynamoDBManager) Refresh(ctx context.Context, lock *Lock) error {
	if lock == nil {
		return fmt.Errorf("lock cannot be nil")
	}

	now := time.Now()
	newExpiresAt := now.Add(time.Duration(lock.TTL) * time.Second)

	m.logger.Debug("Refreshing lock",
		zap.String("key", lock.Key),
		zap.String("lock_id", lock.ID),
	)

	// Update the lock with condition that lockID matches
	_, err := m.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(m.tableName),
		Key: map[string]types.AttributeValue{
			"lockKey": &types.AttributeValueMemberS{Value: lock.Key},
		},
		UpdateExpression: aws.String("SET expiresAt = :newExpiry"),
		ConditionExpression: aws.String("lockID = :lockID AND expiresAt > :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":lockID":    &types.AttributeValueMemberS{Value: lock.ID},
			":newExpiry": &types.AttributeValueMemberN{Value: strconv.FormatInt(newExpiresAt.Unix(), 10)},
			":now":       &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Unix(), 10)},
		},
	})

	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			// Check if lock expired or ID mismatch
			info, getErr := m.Get(ctx, lock.Key)
			if getErr != nil || info == nil {
				return ErrLockNotFound
			}
			if info.IsExpired {
				return ErrLockExpired
			}
			return ErrInvalidLockID
		}
		return fmt.Errorf("failed to refresh lock: %w", err)
	}

	// Update local lock object
	lock.ExpiresAt = newExpiresAt

	m.logger.Debug("Lock refreshed successfully",
		zap.String("key", lock.Key),
		zap.Time("new_expiry", newExpiresAt),
	)

	return nil
}

// Release releases a lock
func (m *DynamoDBManager) Release(ctx context.Context, lock *Lock) error {
	if lock == nil {
		return fmt.Errorf("lock cannot be nil")
	}

	m.logger.Info("Releasing lock",
		zap.String("key", lock.Key),
		zap.String("lock_id", lock.ID),
	)

	// Delete with condition that lockID matches
	_, err := m.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(m.tableName),
		Key: map[string]types.AttributeValue{
			"lockKey": &types.AttributeValueMemberS{Value: lock.Key},
		},
		ConditionExpression: aws.String("lockID = :lockID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":lockID": &types.AttributeValueMemberS{Value: lock.ID},
		},
	})

	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			// Lock ID mismatch or lock doesn't exist
			return ErrInvalidLockID
		}
		return fmt.Errorf("failed to release lock: %w", err)
	}

	m.logger.Info("Lock released successfully", zap.String("key", lock.Key))
	return nil
}

// ForceRelease forcibly releases a lock (admin operation)
func (m *DynamoDBManager) ForceRelease(ctx context.Context, key string) error {
	m.logger.Warn("Force releasing lock (admin operation)", zap.String("key", key))

	_, err := m.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(m.tableName),
		Key: map[string]types.AttributeValue{
			"lockKey": &types.AttributeValueMemberS{Value: key},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to force release lock: %w", err)
	}

	m.logger.Info("Lock force released successfully", zap.String("key", key))
	return nil
}

// Get retrieves information about a lock
func (m *DynamoDBManager) Get(ctx context.Context, key string) (*LockInfo, error) {
	result, err := m.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(m.tableName),
		Key: map[string]types.AttributeValue{
			"lockKey": &types.AttributeValueMemberS{Value: key},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get lock: %w", err)
	}

	if result.Item == nil {
		return nil, ErrLockNotFound
	}

	// Parse the item
	info := &LockInfo{
		Key:      key,
		Metadata: make(map[string]string),
	}

	if owner, ok := result.Item["owner"].(*types.AttributeValueMemberS); ok {
		info.Owner = owner.Value
	}

	if acquiredAt, ok := result.Item["acquiredAt"].(*types.AttributeValueMemberN); ok {
		if ts, err := strconv.ParseInt(acquiredAt.Value, 10, 64); err == nil {
			info.AcquiredAt = time.Unix(ts, 0)
			info.Age = time.Since(info.AcquiredAt)
		}
	}

	if expiresAt, ok := result.Item["expiresAt"].(*types.AttributeValueMemberN); ok {
		if ts, err := strconv.ParseInt(expiresAt.Value, 10, 64); err == nil {
			info.ExpiresAt = time.Unix(ts, 0)
			info.IsExpired = time.Now().After(info.ExpiresAt)
		}
	}

	return info, nil
}

// List lists all locks with the given prefix
func (m *DynamoDBManager) List(ctx context.Context, prefix string) ([]*LockInfo, error) {
	m.logger.Debug("Listing locks", zap.String("prefix", prefix))

	var locks []*LockInfo
	var lastEvaluatedKey map[string]types.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName: aws.String(m.tableName),
		}

		if lastEvaluatedKey != nil {
			input.ExclusiveStartKey = lastEvaluatedKey
		}

		// Add filter if prefix provided
		if prefix != "" {
			input.FilterExpression = aws.String("begins_with(lockKey, :prefix)")
			input.ExpressionAttributeValues = map[string]types.AttributeValue{
				":prefix": &types.AttributeValueMemberS{Value: prefix},
			}
		}

		result, err := m.client.Scan(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list locks: %w", err)
		}

		// Parse items
		for _, item := range result.Items {
			info := &LockInfo{
				Metadata: make(map[string]string),
			}

			if lockKey, ok := item["lockKey"].(*types.AttributeValueMemberS); ok {
				info.Key = lockKey.Value
			}
			if owner, ok := item["owner"].(*types.AttributeValueMemberS); ok {
				info.Owner = owner.Value
			}
			if acquiredAt, ok := item["acquiredAt"].(*types.AttributeValueMemberN); ok {
				if ts, err := strconv.ParseInt(acquiredAt.Value, 10, 64); err == nil {
					info.AcquiredAt = time.Unix(ts, 0)
					info.Age = time.Since(info.AcquiredAt)
				}
			}
			if expiresAt, ok := item["expiresAt"].(*types.AttributeValueMemberN); ok {
				if ts, err := strconv.ParseInt(expiresAt.Value, 10, 64); err == nil {
					info.ExpiresAt = time.Unix(ts, 0)
					info.IsExpired = time.Now().After(info.ExpiresAt)
				}
			}

			locks = append(locks, info)
		}

		// Check if there are more results
		lastEvaluatedKey = result.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			break
		}
	}

	m.logger.Debug("Listed locks", zap.Int("count", len(locks)))
	return locks, nil
}

// Close closes the DynamoDB manager (no-op)
func (m *DynamoDBManager) Close() error {
	m.logger.Info("Closing DynamoDB lock manager")
	return nil
}

// Ensure DynamoDBManager implements Manager interface
var _ Manager = (*DynamoDBManager)(nil)

