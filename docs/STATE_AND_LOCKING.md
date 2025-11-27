# State Management and Distributed Locking

## Overview

This document describes the state management and distributed locking implementation for the deployer system.

---

## State Backend (S3)

### Bucket Structure

```
s3://company-deployer-state/
├── stacks/
│   ├── user-platform/
│   │   ├── production/
│   │   │   ├── state.json                    # Current state
│   │   │   ├── state.json.lock               # Lock file (optional, for visibility)
│   │   │   ├── history/                      # State history (versioned snapshots)
│   │   │   │   ├── 2024-01-15-10-30-00.json
│   │   │   │   ├── 2024-01-15-11-45-00.json
│   │   │   │   └── 2024-01-15-14-20-00.json
│   │   │   └── pulumi/                       # Pulumi state directory
│   │   │       └── .pulumi/
│   │   │           ├── stacks/
│   │   │           │   └── production.json
│   │   │           ├── backups/
│   │   │           └── history/
│   │   │
│   │   ├── staging/
│   │   │   ├── state.json
│   │   │   ├── history/
│   │   │   └── pulumi/
│   │   │
│   │   └── development/
│   │       ├── state.json
│   │       ├── history/
│   │       └── pulumi/
│   │
│   ├── payment-platform/
│   │   └── production/
│   │       └── ...
│   │
│   └── analytics-platform/
│       └── production/
│           └── ...
│
└── locks/                                     # Optional: centralized lock tracking
    ├── user-platform-production.lock
    └── payment-platform-production.lock
```

### State File Format

```json
{
  "version": "1.0.0",
  "format_version": "deployer-state-v1",
  
  "metadata": {
    "stack": "user-platform",
    "environment": "production",
    "last_updated": "2024-01-15T14:20:00Z",
    "last_updated_by": "github-actions-run-12345",
    "deployment_id": "dep-abc123def456",
    "git_commit": "a1b2c3d4e5f6",
    "git_branch": "main"
  },
  
  "lock": {
    "locked": false,
    "lock_id": null,
    "locked_by": null,
    "locked_at": null,
    "expires_at": null
  },
  
  "resources": {
    "user-service/database": {
      "kind": "RDS",
      "api_version": "components.deployer.io/v1",
      "status": "ready",
      
      "lifecycle": {
        "created_at": "2024-01-10T10:00:00Z",
        "last_updated": "2024-01-15T14:20:00Z",
        "version": 5
      },
      
      "desired_config": {
        "engine": {
          "type": "postgres",
          "version": "15.4"
        },
        "instance": {
          "class": "db.r6g.xlarge",
          "storage": {
            "allocatedGB": 100,
            "type": "gp3"
          }
        }
      },
      
      "actual_state": {
        "instance_id": "user-db-prod",
        "endpoint": "user-db-prod.abc123.us-east-1.rds.amazonaws.com",
        "port": 5432,
        "status": "available",
        "engine": "postgres",
        "engine_version": "15.4",
        "instance_class": "db.r6g.xlarge",
        "allocated_storage": 100,
        "multi_az": true
      },
      
      "outputs": {
        "endpoint": "user-db-prod.abc123.us-east-1.rds.amazonaws.com",
        "port": "5432",
        "reader_endpoint": "user-db-prod-ro.abc123.us-east-1.rds.amazonaws.com"
      },
      
      "pulumi": {
        "urn": "urn:pulumi:production::user-platform::aws:rds/instance:Instance::user-service-database",
        "id": "user-db-prod"
      },
      
      "dependencies": [],
      
      "health": {
        "status": "healthy",
        "last_check": "2024-01-15T14:30:00Z"
      }
    },
    
    "user-service/cache": {
      "kind": "ElastiCacheRedis",
      "api_version": "components.deployer.io/v1",
      "status": "ready",
      
      "lifecycle": {
        "created_at": "2024-01-10T10:05:00Z",
        "last_updated": "2024-01-10T10:05:00Z",
        "version": 1
      },
      
      "desired_config": {
        "engine": {
          "version": "7.0"
        },
        "cluster": {
          "nodeType": "cache.r6g.large",
          "numNodes": 3
        }
      },
      
      "actual_state": {
        "cluster_id": "user-cache-prod",
        "configuration_endpoint": "user-cache-prod.abc123.cache.amazonaws.com",
        "port": 6379,
        "status": "available",
        "engine": "redis",
        "engine_version": "7.0.7",
        "num_cache_nodes": 3
      },
      
      "outputs": {
        "endpoint": "user-cache-prod.abc123.cache.amazonaws.com",
        "port": "6379"
      },
      
      "pulumi": {
        "urn": "urn:pulumi:production::user-platform::aws:elasticache/replicationGroup:ReplicationGroup::user-service-cache",
        "id": "user-cache-prod"
      },
      
      "dependencies": [],
      
      "health": {
        "status": "healthy",
        "last_check": "2024-01-15T14:30:00Z"
      }
    },
    
    "user-service/api": {
      "kind": "MicroService",
      "api_version": "components.deployer.io/v1",
      "status": "ready",
      
      "lifecycle": {
        "created_at": "2024-01-10T10:10:00Z",
        "last_updated": "2024-01-15T14:20:00Z",
        "version": 8
      },
      
      "desired_config": {
        "image": {
          "repository": "123456789012.dkr.ecr.us-east-1.amazonaws.com/user-api",
          "tag": "v1.2.3"
        },
        "resources": {
          "cpu": 1024,
          "memory": 2048
        },
        "scaling": {
          "minReplicas": 5,
          "maxReplicas": 50
        }
      },
      
      "actual_state": {
        "service_arn": "arn:aws:ecs:us-east-1:123456789012:service/user-platform-prod/user-api",
        "task_definition": "user-api-prod:15",
        "desired_count": 5,
        "running_count": 5,
        "pending_count": 0,
        "image": "123456789012.dkr.ecr.us-east-1.amazonaws.com/user-api:v1.2.3",
        "load_balancer_arn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/user-api-prod/abc123",
        "target_group_arn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/user-api-prod/xyz789"
      },
      
      "outputs": {
        "url": "http://user-api.user-platform.local:8080",
        "loadBalancerUrl": "https://api.company.com",
        "targetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/user-api-prod/xyz789"
      },
      
      "pulumi": {
        "urns": [
          "urn:pulumi:production::user-platform::aws:ecs/service:Service::user-api",
          "urn:pulumi:production::user-platform::aws:ecs/taskDefinition:TaskDefinition::user-api",
          "urn:pulumi:production::user-platform::aws:lb/targetGroup:TargetGroup::user-api"
        ]
      },
      
      "dependencies": [
        "user-service/database",
        "user-service/cache"
      ],
      
      "health": {
        "status": "healthy",
        "checks": {
          "readiness": {
            "status": "passing",
            "last_check": "2024-01-15T14:30:15Z"
          },
          "liveness": {
            "status": "passing",
            "last_check": "2024-01-15T14:30:20Z"
          }
        },
        "last_check": "2024-01-15T14:30:20Z"
      },
      
      "deployment": {
        "last_deployment_id": "dep-abc123def456",
        "last_deployment_at": "2024-01-15T14:20:00Z",
        "last_deployment_duration_seconds": 342,
        "rollout_status": "completed"
      }
    }
  },
  
  "outputs": {
    "userServiceApiUrl": "https://api.company.com/users",
    "authServiceApiUrl": "https://api.company.com/auth"
  },
  
  "deployment_history": [
    {
      "deployment_id": "dep-abc123def456",
      "timestamp": "2024-01-15T14:20:00Z",
      "triggered_by": "github-actions-run-12345",
      "git_commit": "a1b2c3d4e5f6",
      "version": "v1.2.3",
      "status": "success",
      "duration_seconds": 342,
      "resources_changed": {
        "created": 0,
        "updated": 3,
        "deleted": 0
      }
    },
    {
      "deployment_id": "dep-xyz789abc123",
      "timestamp": "2024-01-14T10:15:00Z",
      "triggered_by": "github-actions-run-12344",
      "git_commit": "b2c3d4e5f6a7",
      "version": "v1.2.2",
      "status": "success",
      "duration_seconds": 318,
      "resources_changed": {
        "created": 0,
        "updated": 1,
        "deleted": 0
      }
    }
  ],
  
  "checksums": {
    "resources": "sha256:abc123...",
    "outputs": "sha256:def456..."
  }
}
```

### S3 Bucket Configuration

```hcl
resource "aws_s3_bucket" "deployer_state" {
  bucket = "company-deployer-state"
  
  tags = {
    Name      = "deployer-state"
    ManagedBy = "terraform"
  }
}

resource "aws_s3_bucket_versioning" "deployer_state" {
  bucket = aws_s3_bucket.deployer_state.id
  
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "deployer_state" {
  bucket = aws_s3_bucket.deployer_state.id
  
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "deployer_state" {
  bucket = aws_s3_bucket.deployer_state.id
  
  rule {
    id     = "delete-old-versions"
    status = "Enabled"
    
    noncurrent_version_expiration {
      days = 90
    }
  }
  
  rule {
    id     = "delete-old-history"
    status = "Enabled"
    
    filter {
      prefix = "stacks/*/*/history/"
    }
    
    expiration {
      days = 90
    }
  }
}

resource "aws_s3_bucket_public_access_block" "deployer_state" {
  bucket = aws_s3_bucket.deployer_state.id
  
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
```

---

## Lock Backend (DynamoDB)

### Table Schema

```hcl
resource "aws_dynamodb_table" "deployer_locks" {
  name         = "deployer-state-locks"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "lockKey"
  
  attribute {
    name = "lockKey"
    type = "S"
  }
  
  ttl {
    attribute_name = "expiresAt"
    enabled        = true
  }
  
  point_in_time_recovery {
    enabled = true
  }
  
  tags = {
    Name      = "deployer-state-locks"
    ManagedBy = "terraform"
  }
}
```

### Lock Item Structure

```json
{
  "lockKey": "stack:user-platform:env:production",
  "lockId": "550e8400-e29b-41d4-a716-446655440000",
  "lockedBy": "github-actions-run-12345",
  "lockedAt": 1705329600,
  "expiresAt": 1705333200,
  "lastHeartbeat": 1705330800,
  "metadata": {
    "user": "alice@company.com",
    "ci_system": "github-actions",
    "ci_run_id": "12345",
    "deployment_id": "dep-abc123def456",
    "git_commit": "a1b2c3d4e5f6",
    "git_branch": "main",
    "hostname": "runner-abc123",
    "pid": 54321
  }
}
```

### Lock Key Formats

```
Level 1 - Stack Level (default):
  lockKey: "stack:{stack-name}:env:{environment}"
  Example: "stack:user-platform:env:production"

Level 2 - Service Level:
  lockKey: "stack:{stack-name}:env:{environment}:service:{service-name}"
  Example: "stack:user-platform:env:production:service:user-service"

Level 3 - Component Level:
  lockKey: "stack:{stack-name}:env:{environment}:component:{component-path}"
  Example: "stack:user-platform:env:production:component:user-service/api"
```

---

## Implementation

### Go Interfaces

```go
package state

import (
    "context"
    "time"
)

// StateManager manages deployment state
type StateManager interface {
    // Load loads the current state
    Load(ctx context.Context, stack, environment string) (*State, error)
    
    // Save saves the state
    Save(ctx context.Context, state *State) error
    
    // History returns deployment history
    History(ctx context.Context, stack, environment string, limit int) ([]*Deployment, error)
    
    // Backup creates a backup of current state
    Backup(ctx context.Context, stack, environment string) error
    
    // Restore restores state from backup
    Restore(ctx context.Context, stack, environment string, timestamp time.Time) error
}

// LockManager manages distributed locks
type LockManager interface {
    // Acquire acquires a lock
    Acquire(ctx context.Context, key string, metadata map[string]string) (*Lock, error)
    
    // Release releases a lock
    Release(ctx context.Context, lock *Lock) error
    
    // Heartbeat sends a heartbeat to keep lock alive
    Heartbeat(ctx context.Context, lock *Lock) error
    
    // IsLocked checks if a lock is currently held
    IsLocked(ctx context.Context, key string) (bool, error)
    
    // GetLock gets information about a lock
    GetLock(ctx context.Context, key string) (*LockInfo, error)
    
    // ForceUnlock forcefully releases a lock (admin operation)
    ForceUnlock(ctx context.Context, key string) error
    
    // ListLocks lists all active locks
    ListLocks(ctx context.Context) ([]*LockInfo, error)
}

// State represents the deployment state
type State struct {
    Version         string                 `json:"version"`
    FormatVersion   string                 `json:"format_version"`
    Metadata        *StateMetadata         `json:"metadata"`
    Resources       map[string]*Resource   `json:"resources"`
    Outputs         map[string]interface{} `json:"outputs"`
    DeploymentHistory []*Deployment        `json:"deployment_history"`
    Checksums       map[string]string      `json:"checksums"`
}

// StateMetadata contains state metadata
type StateMetadata struct {
    Stack        string    `json:"stack"`
    Environment  string    `json:"environment"`
    LastUpdated  time.Time `json:"last_updated"`
    LastUpdatedBy string   `json:"last_updated_by"`
    DeploymentID string    `json:"deployment_id"`
    GitCommit    string    `json:"git_commit"`
    GitBranch    string    `json:"git_branch"`
}

// Resource represents a deployed resource
type Resource struct {
    Kind           string                 `json:"kind"`
    APIVersion     string                 `json:"api_version"`
    Status         string                 `json:"status"`
    Lifecycle      *ResourceLifecycle     `json:"lifecycle"`
    DesiredConfig  map[string]interface{} `json:"desired_config"`
    ActualState    map[string]interface{} `json:"actual_state"`
    Outputs        map[string]interface{} `json:"outputs"`
    Pulumi         *PulumiInfo            `json:"pulumi"`
    Dependencies   []string               `json:"dependencies"`
    Health         *HealthStatus          `json:"health"`
    Deployment     *DeploymentInfo        `json:"deployment,omitempty"`
}

// Lock represents an acquired lock
type Lock struct {
    Key          string
    ID           string
    Metadata     map[string]string
    ExpiresAt    time.Time
    heartbeatStop chan struct{}
}

// LockInfo contains information about a lock
type LockInfo struct {
    Key           string            `json:"key"`
    LockID        string            `json:"lock_id"`
    LockedBy      string            `json:"locked_by"`
    LockedAt      time.Time         `json:"locked_at"`
    ExpiresAt     time.Time         `json:"expires_at"`
    LastHeartbeat time.Time         `json:"last_heartbeat"`
    Metadata      map[string]string `json:"metadata"`
    IsStale       bool              `json:"is_stale"`
}
```

### DynamoDB Lock Implementation

```go
package dynamodb

import (
    "context"
    "fmt"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/google/uuid"
)

type DynamoDBLockManager struct {
    client    *dynamodb.Client
    tableName string
    ttl       time.Duration
}

func NewDynamoDBLockManager(client *dynamodb.Client, tableName string) *DynamoDBLockManager {
    return &DynamoDBLockManager{
        client:    client,
        tableName: tableName,
        ttl:       1 * time.Hour, // Default TTL
    }
}

// Acquire acquires a distributed lock
func (m *DynamoDBLockManager) Acquire(ctx context.Context, key string, metadata map[string]string) (*Lock, error) {
    lockID := uuid.New().String()
    now := time.Now()
    expiresAt := now.Add(m.ttl)
    
    // Try to acquire lock with conditional write
    _, err := m.client.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String(m.tableName),
        Item: map[string]types.AttributeValue{
            "lockKey":       &types.AttributeValueMemberS{Value: key},
            "lockId":        &types.AttributeValueMemberS{Value: lockID},
            "lockedBy":      &types.AttributeValueMemberS{Value: metadata["lockedBy"]},
            "lockedAt":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
            "expiresAt":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expiresAt.Unix())},
            "lastHeartbeat": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
            "metadata":      marshalMetadata(metadata),
        },
        // Conditional: only succeed if lock doesn't exist OR is expired
        ConditionExpression: aws.String("attribute_not_exists(lockKey) OR expiresAt < :now"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":now": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
        },
    })
    
    if err != nil {
        // Check if condition failed (lock is held)
        var ccf *types.ConditionalCheckFailedException
        if errors.As(err, &ccf) {
            return nil, fmt.Errorf("lock is currently held: %w", ErrLockHeld)
        }
        return nil, fmt.Errorf("failed to acquire lock: %w", err)
    }
    
    lock := &Lock{
        Key:       key,
        ID:        lockID,
        Metadata:  metadata,
        ExpiresAt: expiresAt,
        heartbeatStop: make(chan struct{}),
    }
    
    // Start heartbeat goroutine
    go m.heartbeatLoop(context.Background(), lock)
    
    return lock, nil
}

// Release releases a lock
func (m *DynamoDBLockManager) Release(ctx context.Context, lock *Lock) error {
    // Stop heartbeat
    close(lock.heartbeatStop)
    
    // Delete lock item (only if we own it)
    _, err := m.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
        TableName: aws.String(m.tableName),
        Key: map[string]types.AttributeValue{
            "lockKey": &types.AttributeValueMemberS{Value: lock.Key},
        },
        ConditionExpression: aws.String("lockId = :lockId"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":lockId": &types.AttributeValueMemberS{Value: lock.ID},
        },
    })
    
    if err != nil {
        var ccf *types.ConditionalCheckFailedException
        if errors.As(err, &ccf) {
            // Lock was already released or taken over by someone else
            return nil
        }
        return fmt.Errorf("failed to release lock: %w", err)
    }
    
    return nil
}

// heartbeatLoop sends periodic heartbeats
func (m *DynamoDBLockManager) heartbeatLoop(ctx context.Context, lock *Lock) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if err := m.Heartbeat(ctx, lock); err != nil {
                // Log error but continue
                log.Errorf("Failed to send heartbeat for lock %s: %v", lock.Key, err)
            }
        case <-lock.heartbeatStop:
            return
        }
    }
}

// Heartbeat sends a heartbeat to extend lock expiry
func (m *DynamoDBLockManager) Heartbeat(ctx context.Context, lock *Lock) error {
    now := time.Now()
    newExpiresAt := now.Add(m.ttl)
    
    _, err := m.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
        TableName: aws.String(m.tableName),
        Key: map[string]types.AttributeValue{
            "lockKey": &types.AttributeValueMemberS{Value: lock.Key},
        },
        UpdateExpression: aws.String("SET lastHeartbeat = :now, expiresAt = :expiresAt"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":now":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
            ":expiresAt": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", newExpiresAt.Unix())},
            ":lockId":    &types.AttributeValueMemberS{Value: lock.ID},
        },
        ConditionExpression: aws.String("lockId = :lockId"),
    })
    
    if err != nil {
        return fmt.Errorf("failed to send heartbeat: %w", err)
    }
    
    lock.ExpiresAt = newExpiresAt
    return nil
}

// GetLock gets information about a lock
func (m *DynamoDBLockManager) GetLock(ctx context.Context, key string) (*LockInfo, error) {
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
    
    info := parseLockInfo(result.Item)
    
    // Check if lock is stale (no heartbeat for >2x heartbeat interval)
    if time.Since(info.LastHeartbeat) > 2*time.Minute {
        info.IsStale = true
    }
    
    return info, nil
}

// ForceUnlock forcefully releases a lock
func (m *DynamoDBLockManager) ForceUnlock(ctx context.Context, key string) error {
    _, err := m.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
        TableName: aws.String(m.tableName),
        Key: map[string]types.AttributeValue{
            "lockKey": &types.AttributeValueMemberS{Value: key},
        },
    })
    
    if err != nil {
        return fmt.Errorf("failed to force unlock: %w", err)
    }
    
    return nil
}
```

### Usage Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/company/deployer/pkg/state"
)

func deployStack(ctx context.Context, stack, environment string) error {
    lockMgr := state.NewDynamoDBLockManager(dynamoClient, "deployer-state-locks")
    stateMgr := state.NewS3StateManager(s3Client, "company-deployer-state")
    
    // 1. Acquire lock
    lockKey := fmt.Sprintf("stack:%s:env:%s", stack, environment)
    lock, err := lockMgr.Acquire(ctx, lockKey, map[string]string{
        "lockedBy":     "github-actions-run-12345",
        "deploymentId": "dep-abc123",
        "gitCommit":    "a1b2c3d4e5f6",
    })
    if err != nil {
        return fmt.Errorf("failed to acquire lock: %w", err)
    }
    defer lockMgr.Release(ctx, lock)
    
    // 2. Load current state
    currentState, err := stateMgr.Load(ctx, stack, environment)
    if err != nil {
        return fmt.Errorf("failed to load state: %w", err)
    }
    
    // 3. Compute desired state
    desiredState, err := buildDesiredState(stack, environment)
    if err != nil {
        return fmt.Errorf("failed to build desired state: %w", err)
    }
    
    // 4. Generate execution plan
    plan, err := generatePlan(currentState, desiredState)
    if err != nil {
        return fmt.Errorf("failed to generate plan: %w", err)
    }
    
    // 5. Execute deployment
    newState, err := executePlan(ctx, plan)
    if err != nil {
        return fmt.Errorf("deployment failed: %w", err)
    }
    
    // 6. Save new state
    if err := stateMgr.Save(ctx, newState); err != nil {
        return fmt.Errorf("failed to save state: %w", err)
    }
    
    return nil
}
```

---

## Lock Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│ Lock Lifecycle                                                   │
└─────────────────────────────────────────────────────────────────┘

1. ACQUIRE LOCK
   ├─ Generate unique lock ID (UUID)
   ├─ Try conditional write to DynamoDB
   │  Condition: attribute_not_exists(lockKey) OR expiresAt < now
   ├─ If success:
   │  ├─ Lock acquired ✓
   │  └─ Start heartbeat goroutine
   └─ If failure:
      └─ Lock already held, wait or fail

2. HOLD LOCK (during deployment)
   ├─ Heartbeat every 30 seconds
   │  ├─ Update lastHeartbeat timestamp
   │  └─ Extend expiresAt by TTL (1 hour)
   └─ Perform deployment operations

3. RELEASE LOCK
   ├─ Stop heartbeat goroutine
   ├─ Delete item from DynamoDB
   │  Condition: lockId = ourLockId (ensure we own it)
   └─ Lock released ✓

4. STALE LOCK DETECTION
   ├─ No heartbeat for >2 minutes
   ├─ Mark as stale
   └─ Can be force-unlocked by admin

5. AUTOMATIC CLEANUP
   ├─ DynamoDB TTL deletes expired items
   ├─ TTL attribute: expiresAt
   └─ Cleanup happens within 48 hours of expiry
```

---

## Error Handling

### Lock Acquisition Failure

```go
lock, err := lockMgr.Acquire(ctx, lockKey, metadata)
if err != nil {
    if errors.Is(err, state.ErrLockHeld) {
        // Lock is currently held by someone else
        
        // Option 1: Wait and retry
        fmt.Println("Waiting for lock to be released...")
        time.Sleep(30 * time.Second)
        return retry()
        
        // Option 2: Check who holds the lock
        info, _ := lockMgr.GetLock(ctx, lockKey)
        fmt.Printf("Lock held by: %s (since %s)\n", info.LockedBy, info.LockedAt)
        
        // Option 3: Fail fast
        return fmt.Errorf("cannot deploy: another deployment is in progress")
    }
    
    return fmt.Errorf("failed to acquire lock: %w", err)
}
```

### Stale Lock Handling

```bash
# Check lock status
$ deployer state locks --stack user-platform --environment production

Lock Status:
  Stack: user-platform
  Environment: production
  Locked: Yes
  Locked by: github-actions-run-12345
  Locked at: 2024-01-15 10:30:00 (4 hours ago)
  Last heartbeat: 2024-01-15 10:45:00 (3 hours 45 minutes ago)
  Status: ⚠ STALE (no heartbeat for >2 hours)

# Force unlock if stale
$ deployer unlock --stack user-platform --environment production --force

⚠ Warning: This will forcefully release the lock.
  Current holder: github-actions-run-12345
  Locked since: 2024-01-15 10:30:00

Are you sure? (yes/no): yes

✓ Lock released successfully
```

---

## Monitoring and Observability

### CloudWatch Metrics

```
Metrics to track:
- deployer.lock.acquisition.duration (ms)
- deployer.lock.acquisition.success (count)
- deployer.lock.acquisition.failure (count)
- deployer.lock.held.duration (seconds)
- deployer.lock.heartbeat.success (count)
- deployer.lock.heartbeat.failure (count)
- deployer.lock.stale.detected (count)

Dimensions:
- Stack
- Environment
- LockLevel (stack|service|component)
```

### CloudWatch Alarms

```hcl
resource "aws_cloudwatch_metric_alarm" "stale_locks" {
  alarm_name          = "deployer-stale-locks"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "deployer.lock.stale.detected"
  namespace           = "Deployer"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "Alert when stale locks are detected"
  alarm_actions       = [aws_sns_topic.platform_alerts.arn]
}

resource "aws_cloudwatch_metric_alarm" "lock_contention" {
  alarm_name          = "deployer-high-lock-contention"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "deployer.lock.acquisition.failure"
  namespace           = "Deployer"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "Alert when lock acquisition failures are high"
  alarm_actions       = [aws_sns_topic.platform_alerts.arn]
}
```

### Audit Logging

All lock operations are logged to CloudWatch Logs:

```json
{
  "timestamp": "2024-01-15T14:20:00Z",
  "level": "INFO",
  "event": "lock.acquired",
  "lock_key": "stack:user-platform:env:production",
  "lock_id": "550e8400-e29b-41d4-a716-446655440000",
  "locked_by": "github-actions-run-12345",
  "metadata": {
    "user": "alice@company.com",
    "deployment_id": "dep-abc123",
    "git_commit": "a1b2c3d4e5f6"
  }
}

{
  "timestamp": "2024-01-15T14:25:42Z",
  "level": "INFO",
  "event": "lock.released",
  "lock_key": "stack:user-platform:env:production",
  "lock_id": "550e8400-e29b-41d4-a716-446655440000",
  "duration_seconds": 342
}
```

---

## Best Practices

### 1. Always Release Locks

```go
lock, err := lockMgr.Acquire(ctx, lockKey, metadata)
if err != nil {
    return err
}
defer lockMgr.Release(ctx, lock) // Always defer release
```

### 2. Set Reasonable TTL

```go
// Default: 1 hour (conservative)
lockMgr.SetTTL(1 * time.Hour)

// For quick operations: shorter TTL
lockMgr.SetTTL(10 * time.Minute)

// For long migrations: longer TTL
lockMgr.SetTTL(4 * time.Hour)
```

### 3. Handle Context Cancellation

```go
func deploy(ctx context.Context) error {
    lock, err := lockMgr.Acquire(ctx, lockKey, metadata)
    if err != nil {
        return err
    }
    defer lockMgr.Release(context.Background(), lock) // Use background context for cleanup
    
    // Check context during long operations
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Continue
    }
}
```

### 4. Implement Backoff for Retries

```go
func acquireWithRetry(ctx context.Context, lockKey string) (*Lock, error) {
    backoff := []time.Duration{5*time.Second, 10*time.Second, 30*time.Second, 1*time.Minute}
    
    for i, delay := range backoff {
        lock, err := lockMgr.Acquire(ctx, lockKey, metadata)
        if err == nil {
            return lock, nil
        }
        
        if !errors.Is(err, ErrLockHeld) {
            return nil, err
        }
        
        if i < len(backoff)-1 {
            fmt.Printf("Lock held, retrying in %s...\n", delay)
            time.Sleep(delay)
        }
    }
    
    return nil, fmt.Errorf("failed to acquire lock after %d attempts", len(backoff))
}
```

### 5. Monitor Lock Metrics

```go
func (m *DynamoDBLockManager) Acquire(ctx context.Context, key string, metadata map[string]string) (*Lock, error) {
    start := time.Now()
    
    lock, err := m.acquire(ctx, key, metadata)
    
    // Record metrics
    duration := time.Since(start)
    metrics.RecordLockAcquisitionDuration(key, duration)
    
    if err != nil {
        metrics.IncrementLockAcquisitionFailure(key)
        return nil, err
    }
    
    metrics.IncrementLockAcquisitionSuccess(key)
    return lock, nil
}
```

---

This completes the state management and distributed locking design!



