# End-to-End Implementation and Testing Plan

This document provides a complete implementation and testing strategy for the panka system, from initial setup through production deployment.

---

## Table of Contents

1. [Phase 0: Prerequisites & Setup](#phase-0-prerequisites--setup)
2. [Phase 1: Core Infrastructure](#phase-1-core-infrastructure)
3. [Phase 2: State & Lock Management](#phase-2-state--lock-management)
4. [Phase 3: YAML Parser & Validator](#phase-3-yaml-parser--validator)
5. [Phase 4: Dependency Resolution](#phase-4-dependency-resolution)
6. [Phase 5: Reconciliation Engine](#phase-5-reconciliation-engine)
7. [Phase 6: Pulumi Integration](#phase-6-pulumi-integration)
8. [Phase 7: Component Implementations](#phase-7-component-implementations)
9. [Phase 8: CLI & User Experience](#phase-8-cli--user-experience)
10. [Phase 9: Advanced Features](#phase-9-advanced-features)
11. [Phase 10: Production Readiness](#phase-10-production-readiness)
12. [Testing Strategy](#testing-strategy)
13. [Deployment & Rollout Plan](#deployment--rollout-plan)

---

## Phase 0: Prerequisites & Setup

### Week 0: Days 1-5

#### Implementation Tasks

**Day 1-2: Project Initialization**

```bash
# Initialize Go module
mkdir -p ~/work/panka
cd ~/work/panka
go mod init github.com/company/panka

# Create project structure
mkdir -p cmd/panka
mkdir -p pkg/{state,lock,parser,graph,reconciler,executor,pulumi,components}
mkdir -p internal/{aws,config,logger,metrics}
mkdir -p test/{unit,integration,e2e,fixtures}
mkdir -p docs
mkdir -p scripts
mkdir -p examples

# Create main.go
cat > cmd/panka/main.go << 'EOF'
package main

import (
    "fmt"
    "os"
)

func main() {
    fmt.Println("Panka v0.1.0")
    os.Exit(0)
}
EOF

# Build and verify
go build -o bin/panka ./cmd/panka
./bin/panka
```

**Day 3: CI/CD Setup**

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install dependencies
        run: go mod download
      
      - name: Run linter
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
      
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.txt
      
      - name: Build
        run: go build -v ./cmd/panka

  integration-test:
    runs-on: ubuntu-latest
    services:
      localstack:
        image: localstack/localstack:latest
        env:
          SERVICES: s3,dynamodb,ecs,rds
        ports:
          - 4566:4566
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run integration tests
        run: go test -v -tags=integration ./test/integration/...
        env:
          AWS_ENDPOINT: http://localhost:4566
```

**Day 4-5: Development Environment Setup**

Create `Makefile`:

```makefile
.PHONY: build test lint clean install dev

# Build settings
BINARY_NAME=panka
VERSION?=0.1.0
BUILD_DIR=bin
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Build the binary
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/panka

# Install the binary
install:
	go install $(LDFLAGS) ./cmd/panka

# Run tests
test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run integration tests
test-integration:
	go test -v -tags=integration ./test/integration/...

# Run e2e tests
test-e2e:
	go test -v -tags=e2e ./test/e2e/...

# Run all tests
test-all: test test-integration test-e2e

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.txt

# Start development environment (LocalStack)
dev:
	docker-compose up -d

# Stop development environment
dev-stop:
	docker-compose down

# Generate mocks
mocks:
	mockgen -source=pkg/state/interface.go -destination=pkg/state/mocks/mock_state.go
	mockgen -source=pkg/lock/interface.go -destination=pkg/lock/mocks/mock_lock.go

# Run pre-commit checks
pre-commit: fmt lint test

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golang/mock/mockgen@latest
```

Create `docker-compose.yml` for local development:

```yaml
version: '3.8'

services:
  localstack:
    image: localstack/localstack:latest
    ports:
      - "4566:4566"
    environment:
      - SERVICES=s3,dynamodb,ecs,rds,elasticache,sqs,sns
      - DEBUG=1
      - DATA_DIR=/tmp/localstack/data
      - DOCKER_HOST=unix:///var/run/docker.sock
    volumes:
      - "./test/localstack:/docker-entrypoint-initaws.d"
      - "/var/run/docker.sock:/var/run/docker.sock"
  
  postgres:
    image: postgres:15
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_USER=testuser
      - POSTGRES_PASSWORD=testpass
      - POSTGRES_DB=testdb
    volumes:
      - postgres-data:/var/lib/postgresql/data

volumes:
  postgres-data:
```

#### Testing Tasks

**Unit Tests:**

```go
// test/setup_test.go
package test

import (
    "testing"
)

func TestProjectSetup(t *testing.T) {
    t.Run("binary builds successfully", func(t *testing.T) {
        // This test verifies that the project compiles
        // If we got here, the build succeeded
    })
}
```

**Acceptance Criteria:**
- ✅ Project compiles successfully
- ✅ CI/CD pipeline runs
- ✅ Development environment starts
- ✅ Makefile commands work
- ✅ Code passes linter

---

## Phase 1: Core Infrastructure

### Week 1: Days 1-5

#### Implementation Tasks

**AWS Infrastructure Setup**

Create `infrastructure/terraform/main.tf`:

```hcl
terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  
  backend "s3" {
    bucket         = "company-terraform-state"
    key            = "panka/infrastructure/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-state-lock"
  }
}

provider "aws" {
  region = var.aws_region
  
  default_tags {
    tags = {
      Project     = "panka"
      ManagedBy   = "terraform"
      Environment = var.environment
    }
  }
}

# S3 Bucket for State
resource "aws_s3_bucket" "panka_state" {
  bucket = "${var.prefix}-panka-state-${var.environment}"
  
  tags = {
    Name = "panka-state"
  }
}

resource "aws_s3_bucket_versioning" "panka_state" {
  bucket = aws_s3_bucket.panka_state.id
  
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "panka_state" {
  bucket = aws_s3_bucket.panka_state.id
  
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "panka_state" {
  bucket = aws_s3_bucket.panka_state.id
  
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

resource "aws_s3_bucket_public_access_block" "panka_state" {
  bucket = aws_s3_bucket.panka_state.id
  
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# DynamoDB Table for Locks
resource "aws_dynamodb_table" "panka_locks" {
  name         = "${var.prefix}-panka-locks-${var.environment}"
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
    Name = "panka-state-locks"
  }
}

# IAM Role for Panka
resource "aws_iam_role" "panka_execution" {
  name = "${var.prefix}-panka-execution-${var.environment}"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      },
      {
        Effect = "Allow"
        Principal = {
          Federated = "arn:aws:iam::${var.aws_account_id}:oidc-provider/token.actions.githubusercontent.com"
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          }
          StringLike = {
            "token.actions.githubusercontent.com:sub" = "repo:${var.github_org}/${var.github_repo}:*"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "panka_execution" {
  name = "panka-execution-policy"
  role = aws_iam_role.panka_execution.id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket",
          "s3:GetBucketVersioning",
          "s3:GetObjectVersion"
        ]
        Resource = [
          aws_s3_bucket.panka_state.arn,
          "${aws_s3_bucket.panka_state.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:DeleteItem",
          "dynamodb:UpdateItem",
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = aws_dynamodb_table.panka_locks.arn
      },
      {
        Effect = "Allow"
        Action = [
          "ecs:*",
          "ecr:*",
          "rds:*",
          "elasticache:*",
          "s3:*",
          "sqs:*",
          "sns:*",
          "elasticloadbalancing:*",
          "ec2:Describe*",
          "ec2:CreateTags",
          "logs:*",
          "cloudwatch:*",
          "secretsmanager:*",
          "kms:*",
          "iam:PassRole",
          "iam:GetRole",
          "iam:ListRolePolicies"
        ]
        Resource = "*"
      }
    ]
  })
}

# Outputs
output "state_bucket" {
  value = aws_s3_bucket.panka_state.id
}

output "lock_table" {
  value = aws_dynamodb_table.panka_locks.id
}

output "execution_role_arn" {
  value = aws_iam_role.panka_execution.arn
}
```

Create `infrastructure/terraform/variables.tf`:

```hcl
variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "aws_account_id" {
  description = "AWS account ID"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "prefix" {
  description = "Resource name prefix"
  type        = string
  default     = "company"
}

variable "github_org" {
  description = "GitHub organization"
  type        = string
}

variable "github_repo" {
  description = "GitHub repository name"
  type        = string
}
```

**Deploy Infrastructure:**

```bash
cd infrastructure/terraform

# Initialize
terraform init

# Plan
terraform plan \
  -var="aws_account_id=123456789012" \
  -var="environment=dev" \
  -var="github_org=company" \
  -var="github_repo=panka"

# Apply
terraform apply \
  -var="aws_account_id=123456789012" \
  -var="environment=dev" \
  -var="github_org=company" \
  -var="github_repo=panka"
```

#### Testing Tasks

**Integration Tests:**

```bash
# Test S3 bucket
aws s3 ls s3://company-panka-state-dev/

# Test DynamoDB table
aws dynamodb describe-table --table-name company-panka-locks-dev

# Test IAM role
aws iam get-role --role-name company-panka-execution-dev
```

**Acceptance Criteria:**
- ✅ S3 bucket created with versioning
- ✅ DynamoDB table created with TTL
- ✅ IAM role created with correct policies
- ✅ Resources tagged correctly
- ✅ Public access blocked on S3

---

## Phase 2: State & Lock Management

### Week 2-3: Days 1-10

#### Implementation Tasks

**Day 1-3: State Manager Implementation**

Create `pkg/state/interface.go`:

```go
package state

import (
    "context"
    "time"
)

// StateManager manages deployment state
type StateManager interface {
    Load(ctx context.Context, stack, environment string) (*State, error)
    Save(ctx context.Context, state *State) error
    History(ctx context.Context, stack, environment string, limit int) ([]*Deployment, error)
    Backup(ctx context.Context, stack, environment string) error
    Restore(ctx context.Context, stack, environment string, timestamp time.Time) error
}

// State represents the deployment state
type State struct {
    Version           string                 `json:"version"`
    FormatVersion     string                 `json:"format_version"`
    Metadata          *StateMetadata         `json:"metadata"`
    Resources         map[string]*Resource   `json:"resources"`
    Outputs           map[string]interface{} `json:"outputs"`
    DeploymentHistory []*Deployment          `json:"deployment_history"`
    Checksums         map[string]string      `json:"checksums"`
}

// StateMetadata contains state metadata
type StateMetadata struct {
    Stack         string    `json:"stack"`
    Environment   string    `json:"environment"`
    LastUpdated   time.Time `json:"last_updated"`
    LastUpdatedBy string    `json:"last_updated_by"`
    DeploymentID  string    `json:"deployment_id"`
    GitCommit     string    `json:"git_commit"`
    GitBranch     string    `json:"git_branch"`
}

// Resource represents a deployed resource
type Resource struct {
    Kind          string                 `json:"kind"`
    APIVersion    string                 `json:"api_version"`
    Status        string                 `json:"status"`
    Lifecycle     *ResourceLifecycle     `json:"lifecycle"`
    DesiredConfig map[string]interface{} `json:"desired_config"`
    ActualState   map[string]interface{} `json:"actual_state"`
    Outputs       map[string]interface{} `json:"outputs"`
    Pulumi        *PulumiInfo            `json:"pulumi"`
    Dependencies  []string               `json:"dependencies"`
    Health        *HealthStatus          `json:"health"`
}

// ResourceLifecycle contains resource lifecycle info
type ResourceLifecycle struct {
    CreatedAt   time.Time `json:"created_at"`
    LastUpdated time.Time `json:"last_updated"`
    Version     int       `json:"version"`
}

// PulumiInfo contains Pulumi-specific info
type PulumiInfo struct {
    URN  string   `json:"urn,omitempty"`
    URNs []string `json:"urns,omitempty"`
    ID   string   `json:"id,omitempty"`
}

// HealthStatus represents resource health
type HealthStatus struct {
    Status    string    `json:"status"`
    LastCheck time.Time `json:"last_check"`
    Checks    map[string]*HealthCheck `json:"checks,omitempty"`
}

// HealthCheck represents a single health check
type HealthCheck struct {
    Status    string    `json:"status"`
    LastCheck time.Time `json:"last_check"`
}

// Deployment represents a deployment event
type Deployment struct {
    DeploymentID     string    `json:"deployment_id"`
    Timestamp        time.Time `json:"timestamp"`
    TriggeredBy      string    `json:"triggered_by"`
    GitCommit        string    `json:"git_commit"`
    Version          string    `json:"version"`
    Status           string    `json:"status"`
    DurationSeconds  int       `json:"duration_seconds"`
    ResourcesChanged struct {
        Created int `json:"created"`
        Updated int `json:"updated"`
        Deleted int `json:"deleted"`
    } `json:"resources_changed"`
}
```

Create `pkg/state/s3/state_manager.go`:

```go
package s3

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/company/panka/pkg/state"
)

type S3StateManager struct {
    client *s3.Client
    bucket string
}

func NewS3StateManager(client *s3.Client, bucket string) *S3StateManager {
    return &S3StateManager{
        client: client,
        bucket: bucket,
    }
}

func (m *S3StateManager) Load(ctx context.Context, stack, environment string) (*state.State, error) {
    key := m.stateKey(stack, environment)
    
    result, err := m.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(m.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to load state: %w", err)
    }
    defer result.Body.Close()
    
    var st state.State
    if err := json.NewDecoder(result.Body).Decode(&st); err != nil {
        return nil, fmt.Errorf("failed to decode state: %w", err)
    }
    
    return &st, nil
}

func (m *S3StateManager) Save(ctx context.Context, st *state.State) error {
    key := m.stateKey(st.Metadata.Stack, st.Metadata.Environment)
    
    data, err := json.MarshalIndent(st, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal state: %w", err)
    }
    
    // Save current state
    _, err = m.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String(m.bucket),
        Key:         aws.String(key),
        Body:        bytes.NewReader(data),
        ContentType: aws.String("application/json"),
    })
    if err != nil {
        return fmt.Errorf("failed to save state: %w", err)
    }
    
    // Save to history
    historyKey := m.historyKey(st.Metadata.Stack, st.Metadata.Environment, time.Now())
    _, err = m.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String(m.bucket),
        Key:         aws.String(historyKey),
        Body:        bytes.NewReader(data),
        ContentType: aws.String("application/json"),
    })
    if err != nil {
        return fmt.Errorf("failed to save state history: %w", err)
    }
    
    return nil
}

func (m *S3StateManager) stateKey(stack, environment string) string {
    return fmt.Sprintf("stacks/%s/%s/state.json", stack, environment)
}

func (m *S3StateManager) historyKey(stack, environment string, timestamp time.Time) string {
    return fmt.Sprintf("stacks/%s/%s/history/%s.json", 
        stack, environment, timestamp.Format("2006-01-02-15-04-05"))
}

// ... implement other methods
```

**Day 4-7: Lock Manager Implementation**

Create `pkg/lock/interface.go`:

```go
package lock

import (
    "context"
    "time"
)

// LockManager manages distributed locks
type LockManager interface {
    Acquire(ctx context.Context, key string, metadata map[string]string) (*Lock, error)
    Release(ctx context.Context, lock *Lock) error
    Heartbeat(ctx context.Context, lock *Lock) error
    IsLocked(ctx context.Context, key string) (bool, error)
    GetLock(ctx context.Context, key string) (*LockInfo, error)
    ForceUnlock(ctx context.Context, key string) error
    ListLocks(ctx context.Context) ([]*LockInfo, error)
}

// Lock represents an acquired lock
type Lock struct {
    Key           string
    ID            string
    Metadata      map[string]string
    ExpiresAt     time.Time
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

Create `pkg/lock/dynamodb/lock_manager.go`:

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
    "github.com/company/panka/pkg/lock"
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
        ttl:       1 * time.Hour,
    }
}

func (m *DynamoDBLockManager) Acquire(ctx context.Context, key string, metadata map[string]string) (*lock.Lock, error) {
    lockID := uuid.New().String()
    now := time.Now()
    expiresAt := now.Add(m.ttl)
    
    _, err := m.client.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String(m.tableName),
        Item: map[string]types.AttributeValue{
            "lockKey":       &types.AttributeValueMemberS{Value: key},
            "lockId":        &types.AttributeValueMemberS{Value: lockID},
            "lockedBy":      &types.AttributeValueMemberS{Value: metadata["lockedBy"]},
            "lockedAt":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
            "expiresAt":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expiresAt.Unix())},
            "lastHeartbeat": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
        },
        ConditionExpression: aws.String("attribute_not_exists(lockKey) OR expiresAt < :now"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":now": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
        },
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to acquire lock: %w", err)
    }
    
    l := &lock.Lock{
        Key:           key,
        ID:            lockID,
        Metadata:      metadata,
        ExpiresAt:     expiresAt,
        heartbeatStop: make(chan struct{}),
    }
    
    // Start heartbeat
    go m.heartbeatLoop(context.Background(), l)
    
    return l, nil
}

func (m *DynamoDBLockManager) heartbeatLoop(ctx context.Context, l *lock.Lock) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if err := m.Heartbeat(ctx, l); err != nil {
                // Log error but continue
                fmt.Printf("Heartbeat failed: %v\n", err)
            }
        case <-l.heartbeatStop:
            return
        }
    }
}

// ... implement other methods
```

**Day 8-10: Testing & Integration**

Create comprehensive tests:

```go
// pkg/state/s3/state_manager_test.go
package s3

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestS3StateManager_LoadAndSave(t *testing.T) {
    // Setup LocalStack
    client := setupLocalStackS3(t)
    manager := NewS3StateManager(client, "test-bucket")
    
    ctx := context.Background()
    
    // Create test state
    state := &state.State{
        Version:       "1.0.0",
        FormatVersion: "panka-state-v1",
        Metadata: &state.StateMetadata{
            Stack:       "test-stack",
            Environment: "dev",
        },
        Resources: make(map[string]*state.Resource),
    }
    
    // Save
    err := manager.Save(ctx, state)
    require.NoError(t, err)
    
    // Load
    loaded, err := manager.Load(ctx, "test-stack", "dev")
    require.NoError(t, err)
    
    // Assert
    assert.Equal(t, state.Version, loaded.Version)
    assert.Equal(t, state.Metadata.Stack, loaded.Metadata.Stack)
}

// pkg/lock/dynamodb/lock_manager_test.go
package dynamodb

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestDynamoDBLockManager_AcquireAndRelease(t *testing.T) {
    client := setupLocalStackDynamoDB(t)
    manager := NewDynamoDBLockManager(client, "test-locks")
    
    ctx := context.Background()
    
    // Acquire lock
    lock, err := manager.Acquire(ctx, "test-key", map[string]string{
        "lockedBy": "test",
    })
    require.NoError(t, err)
    require.NotNil(t, lock)
    
    // Verify locked
    locked, err := manager.IsLocked(ctx, "test-key")
    require.NoError(t, err)
    assert.True(t, locked)
    
    // Release lock
    err = manager.Release(ctx, lock)
    require.NoError(t, err)
    
    // Verify unlocked
    locked, err = manager.IsLocked(ctx, "test-key")
    require.NoError(t, err)
    assert.False(t, locked)
}

func TestDynamoDBLockManager_Heartbeat(t *testing.T) {
    client := setupLocalStackDynamoDB(t)
    manager := NewDynamoDBLockManager(client, "test-locks")
    
    ctx := context.Background()
    
    lock, err := manager.Acquire(ctx, "test-key", map[string]string{
        "lockedBy": "test",
    })
    require.NoError(t, err)
    
    initialExpiry := lock.ExpiresAt
    
    // Wait a bit
    time.Sleep(2 * time.Second)
    
    // Send heartbeat
    err = manager.Heartbeat(ctx, lock)
    require.NoError(t, err)
    
    // Expiry should be extended
    assert.True(t, lock.ExpiresAt.After(initialExpiry))
}
```

#### Acceptance Criteria
- ✅ State can be saved and loaded from S3
- ✅ State history is preserved
- ✅ State versioning works
- ✅ Locks can be acquired and released
- ✅ Lock heartbeats extend expiry
- ✅ Stale locks can be detected
- ✅ Force unlock works
- ✅ All tests pass with 80%+ coverage

---

## Phase 3: YAML Parser & Validator

### Week 4-5: Days 1-10

#### Implementation Tasks

**Day 1-3: Schema Definitions**

Create `pkg/parser/schema/types.go`:

```go
package schema

// Stack represents a stack definition
type Stack struct {
    APIVersion string        `yaml:"apiVersion"`
    Kind       string        `yaml:"kind"`
    Metadata   Metadata      `yaml:"metadata"`
    Spec       StackSpec     `yaml:"spec"`
}

type StackSpec struct {
    Provider     Provider              `yaml:"provider"`
    Shared       SharedResources       `yaml:"shared,omitempty"`
    Defaults     Defaults              `yaml:"defaults,omitempty"`
    Services     []ServiceRef          `yaml:"services,omitempty"`
    Deployment   DeploymentConfig      `yaml:"deployment,omitempty"`
    Monitoring   MonitoringConfig      `yaml:"monitoring,omitempty"`
    Outputs      map[string]Output     `yaml:"outputs,omitempty"`
}

// Service represents a service definition
type Service struct {
    APIVersion string      `yaml:"apiVersion"`
    Kind       string      `yaml:"kind"`
    Metadata   Metadata    `yaml:"metadata"`
    Spec       ServiceSpec `yaml:"spec"`
}

type ServiceSpec struct {
    Infrastructure InfrastructureRefs `yaml:"infrastructure,omitempty"`
    Components     []ComponentRef     `yaml:"components,omitempty"`
}

// Component types
type MicroService struct {
    APIVersion string           `yaml:"apiVersion"`
    Kind       string           `yaml:"kind"`
    Metadata   Metadata         `yaml:"metadata"`
    Spec       MicroServiceSpec `yaml:"spec"`
}

type MicroServiceSpec struct {
    Image        ImageConfig        `yaml:"image"`
    Runtime      RuntimeConfig      `yaml:"runtime"`
    Container    ContainerConfig    `yaml:"container,omitempty"`
    Ports        []PortConfig       `yaml:"ports"`
    Environment  []EnvVar           `yaml:"environment,omitempty"`
    Secrets      []SecretRef        `yaml:"secrets,omitempty"`
    Configs      ConfigMount        `yaml:"configs,omitempty"`
    Mounts       []Mount            `yaml:"mounts,omitempty"`
    HealthCheck  HealthCheckConfig  `yaml:"healthCheck,omitempty"`
    Resources    ResourceRequirements `yaml:"resources,omitempty"`
    DependsOn    []string           `yaml:"dependsOn,omitempty"`
    Outputs      map[string]string  `yaml:"outputs,omitempty"`
}

// ... more component types (RDS, S3, etc.)
```

**Day 4-6: Parser Implementation**

Create `pkg/parser/parser.go`:

```go
package parser

import (
    "fmt"
    "io/ioutil"
    "path/filepath"
    
    "gopkg.in/yaml.v3"
    "github.com/company/panka/pkg/parser/schema"
)

type Parser struct {
    stackPath string
}

func NewParser(stackPath string) *Parser {
    return &Parser{stackPath: stackPath}
}

func (p *Parser) ParseStack() (*schema.Stack, error) {
    stackFile := filepath.Join(p.stackPath, "stack.yaml")
    
    data, err := ioutil.ReadFile(stackFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read stack file: %w", err)
    }
    
    var stack schema.Stack
    if err := yaml.Unmarshal(data, &stack); err != nil {
        return nil, fmt.Errorf("failed to parse stack: %w", err)
    }
    
    return &stack, nil
}

func (p *Parser) ParseServices() ([]*schema.Service, error) {
    servicesPath := filepath.Join(p.stackPath, "services")
    
    serviceDirs, err := ioutil.ReadDir(servicesPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read services directory: %w", err)
    }
    
    var services []*schema.Service
    for _, dir := range serviceDirs {
        if !dir.IsDir() {
            continue
        }
        
        serviceFile := filepath.Join(servicesPath, dir.Name(), "service.yaml")
        service, err := p.parseServiceFile(serviceFile)
        if err != nil {
            return nil, fmt.Errorf("failed to parse service %s: %w", dir.Name(), err)
        }
        
        services = append(services, service)
    }
    
    return services, nil
}

func (p *Parser) ParseComponents(serviceName string) ([]interface{}, error) {
    componentsPath := filepath.Join(p.stackPath, "services", serviceName, "components")
    
    componentDirs, err := ioutil.ReadDir(componentsPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read components directory: %w", err)
    }
    
    var components []interface{}
    for _, dir := range componentDirs {
        if !dir.IsDir() {
            continue
        }
        
        componentPath := filepath.Join(componentsPath, dir.Name())
        component, err := p.parseComponent(componentPath)
        if err != nil {
            return nil, fmt.Errorf("failed to parse component %s: %w", dir.Name(), err)
        }
        
        components = append(components, component)
    }
    
    return components, nil
}

func (p *Parser) parseComponent(componentPath string) (interface{}, error) {
    // Find component definition file (microservice.yaml, rds.yaml, etc.)
    files, err := ioutil.ReadDir(componentPath)
    if err != nil {
        return nil, err
    }
    
    for _, file := range files {
        if file.IsDir() || file.Name() == "infra.yaml" {
            continue
        }
        
        if filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml" {
            return p.parseComponentFile(filepath.Join(componentPath, file.Name()))
        }
    }
    
    return nil, fmt.Errorf("no component definition found")
}

func (p *Parser) parseComponentFile(filePath string) (interface{}, error) {
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
    
    // First, parse to determine kind
    var meta struct {
        Kind string `yaml:"kind"`
    }
    if err := yaml.Unmarshal(data, &meta); err != nil {
        return nil, err
    }
    
    // Parse based on kind
    switch meta.Kind {
    case "MicroService":
        var ms schema.MicroService
        if err := yaml.Unmarshal(data, &ms); err != nil {
            return nil, err
        }
        return &ms, nil
    
    case "RDS":
        var rds schema.RDS
        if err := yaml.Unmarshal(data, &rds); err != nil {
            return nil, err
        }
        return &rds, nil
    
    // ... other component types
    
    default:
        return nil, fmt.Errorf("unknown component kind: %s", meta.Kind)
    }
}
```

**Day 7-8: Validator Implementation**

Create `pkg/validator/validator.go`:

```go
package validator

import (
    "fmt"
    
    "github.com/company/panka/pkg/parser/schema"
)

type Validator struct {
    errors []error
}

func NewValidator() *Validator {
    return &Validator{
        errors: make([]error, 0),
    }
}

func (v *Validator) ValidateStack(stack *schema.Stack) error {
    v.errors = make([]error, 0)
    
    // Validate API version
    if stack.APIVersion != "core.panka.io/v1" {
        v.addError(fmt.Errorf("invalid apiVersion: %s", stack.APIVersion))
    }
    
    // Validate kind
    if stack.Kind != "Stack" {
        v.addError(fmt.Errorf("invalid kind: %s (expected Stack)", stack.Kind))
    }
    
    // Validate metadata
    if err := v.validateMetadata(stack.Metadata); err != nil {
        v.addError(err)
    }
    
    // Validate provider
    if err := v.validateProvider(stack.Spec.Provider); err != nil {
        v.addError(err)
    }
    
    if len(v.errors) > 0 {
        return fmt.Errorf("validation failed: %v", v.errors)
    }
    
    return nil
}

func (v *Validator) ValidateMicroService(ms *schema.MicroService) error {
    v.errors = make([]error, 0)
    
    // Validate required fields
    if ms.Spec.Image.Repository == "" {
        v.addError(fmt.Errorf("image.repository is required"))
    }
    
    if ms.Spec.Image.Tag == "" {
        v.addError(fmt.Errorf("image.tag is required"))
    }
    
    if len(ms.Spec.Ports) == 0 {
        v.addError(fmt.Errorf("at least one port must be defined"))
    }
    
    // Validate health checks
    if ms.Spec.HealthCheck != nil {
        if err := v.validateHealthCheck(ms.Spec.HealthCheck); err != nil {
            v.addError(err)
        }
    }
    
    // Validate environment variables
    for _, env := range ms.Spec.Environment {
        if env.Name == "" {
            v.addError(fmt.Errorf("environment variable name cannot be empty"))
        }
    }
    
    if len(v.errors) > 0 {
        return fmt.Errorf("microservice validation failed: %v", v.errors)
    }
    
    return nil
}

func (v *Validator) validateHealthCheck(hc *schema.HealthCheckConfig) error {
    if hc.Readiness != nil {
        if hc.Readiness.HTTP != nil {
            if hc.Readiness.HTTP.Path == "" {
                return fmt.Errorf("readiness health check path is required")
            }
            if hc.Readiness.HTTP.Port == 0 {
                return fmt.Errorf("readiness health check port is required")
            }
        }
    }
    return nil
}

func (v *Validator) addError(err error) {
    v.errors = append(v.errors, err)
}
```

**Day 9-10: Environment Overlay Merger**

Create `pkg/parser/merger.go`:

```go
package parser

import (
    "github.com/imdario/mergo"
    "github.com/company/panka/pkg/parser/schema"
)

type Merger struct{}

func NewMerger() *Merger {
    return &Merger{}
}

func (m *Merger) MergeStackWithEnvironment(base *schema.Stack, overlay *schema.Stack) (*schema.Stack, error) {
    // Deep copy base
    merged := *base
    
    // Merge overlay into base using strategic merge
    if err := mergo.Merge(&merged, overlay, mergo.WithOverride); err != nil {
        return nil, err
    }
    
    return &merged, nil
}

func (m *Merger) MergeMicroServiceWithEnvironment(base *schema.MicroService, overlay *schema.MicroService) (*schema.MicroService, error) {
    merged := *base
    
    if err := mergo.Merge(&merged, overlay, mergo.WithOverride, mergo.WithAppendSlice); err != nil {
        return nil, err
    }
    
    return &merged, nil
}
```

#### Testing Tasks

Create comprehensive tests:

```go
// pkg/parser/parser_test.go
func TestParser_ParseStack(t *testing.T) {
    parser := NewParser("../../test/fixtures/stacks/test-stack")
    
    stack, err := parser.ParseStack()
    require.NoError(t, err)
    require.NotNil(t, stack)
    
    assert.Equal(t, "test-stack", stack.Metadata.Name)
    assert.Equal(t, "core.panka.io/v1", stack.APIVersion)
}

// pkg/validator/validator_test.go
func TestValidator_ValidateMicroService(t *testing.T) {
    validator := NewValidator()
    
    tests := []struct {
        name    string
        ms      *schema.MicroService
        wantErr bool
    }{
        {
            name: "valid microservice",
            ms: &schema.MicroService{
                Spec: schema.MicroServiceSpec{
                    Image: schema.ImageConfig{
                        Repository: "test/image",
                        Tag:        "v1.0.0",
                    },
                    Ports: []schema.PortConfig{
                        {Port: 8080},
                    },
                },
            },
            wantErr: false,
        },
        {
            name: "missing image repository",
            ms: &schema.MicroService{
                Spec: schema.MicroServiceSpec{
                    Image: schema.ImageConfig{
                        Tag: "v1.0.0",
                    },
                },
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.ValidateMicroService(tt.ms)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### Acceptance Criteria
- ✅ Stack YAML can be parsed
- ✅ Service YAML can be parsed
- ✅ All component types can be parsed
- ✅ Schema validation works
- ✅ Environment overlays merge correctly
- ✅ Variable interpolation works
- ✅ Cross-references are validated
- ✅ Tests pass with 80%+ coverage

---

## Phase 4: Dependency Resolution

### Week 6: Days 1-5

#### Implementation Tasks

**Day 1-2: Dependency Graph Builder**

Create `pkg/graph/graph.go`:

```go
package graph

import (
    "fmt"
    
    "github.com/company/panka/pkg/parser/schema"
)

type Node struct {
    ID           string
    Kind         string
    Component    interface{}
    Dependencies []string
}

type Graph struct {
    Nodes map[string]*Node
    Edges map[string][]string
}

func NewGraph() *Graph {
    return &Graph{
        Nodes: make(map[string]*Node),
        Edges: make(map[string][]string),
    }
}

func (g *Graph) AddNode(id string, kind string, component interface{}) {
    g.Nodes[id] = &Node{
        ID:           id,
        Kind:         kind,
        Component:    component,
        Dependencies: make([]string, 0),
    }
}

func (g *Graph) AddEdge(from, to string) {
    if _, ok := g.Edges[from]; !ok {
        g.Edges[from] = make([]string, 0)
    }
    g.Edges[from] = append(g.Edges[from], to)
    
    if node, ok := g.Nodes[from]; ok {
        node.Dependencies = append(node.Dependencies, to)
    }
}

func (g *Graph) TopologicalSort() ([][]*Node, error) {
    // Calculate in-degree for each node
    inDegree := make(map[string]int)
    for id := range g.Nodes {
        inDegree[id] = 0
    }
    for _, deps := range g.Edges {
        for _, dep := range deps {
            inDegree[dep]++
        }
    }
    
    // Find nodes with no dependencies (wave 1)
    var waves [][]*Node
    processed := make(map[string]bool)
    
    for {
        var wave []*Node
        for id, degree := range inDegree {
            if degree == 0 && !processed[id] {
                wave = append(wave, g.Nodes[id])
                processed[id] = true
            }
        }
        
        if len(wave) == 0 {
            break
        }
        
        // Reduce in-degree for dependents
        for _, node := range wave {
            for _, dep := range g.Edges[node.ID] {
                inDegree[dep]--
            }
        }
        
        waves = append(waves, wave)
    }
    
    // Check if all nodes were processed (detect cycles)
    if len(processed) != len(g.Nodes) {
        return nil, fmt.Errorf("cycle detected in dependency graph")
    }
    
    return waves, nil
}

func (g *Graph) DetectCycles() ([][]string, error) {
    visited := make(map[string]bool)
    recStack := make(map[string]bool)
    var cycles [][]string
    
    for id := range g.Nodes {
        if !visited[id] {
            if cycle := g.detectCycleDFS(id, visited, recStack, []string{}); len(cycle) > 0 {
                cycles = append(cycles, cycle)
            }
        }
    }
    
    if len(cycles) > 0 {
        return cycles, fmt.Errorf("cycles detected")
    }
    
    return nil, nil
}

func (g *Graph) detectCycleDFS(node string, visited, recStack map[string]bool, path []string) []string {
    visited[node] = true
    recStack[node] = true
    path = append(path, node)
    
    for _, dep := range g.Edges[node] {
        if !visited[dep] {
            if cycle := g.detectCycleDFS(dep, visited, recStack, path); len(cycle) > 0 {
                return cycle
            }
        } else if recStack[dep] {
            // Found cycle
            cycleStart := 0
            for i, n := range path {
                if n == dep {
                    cycleStart = i
                    break
                }
            }
            return append(path[cycleStart:], dep)
        }
    }
    
    recStack[node] = false
    return nil
}
```

**Day 3-4: Dependency Extractor**

Create `pkg/graph/extractor.go`:

```go
package graph

import (
    "fmt"
    "strings"
    
    "github.com/company/panka/pkg/parser/schema"
)

type DependencyExtractor struct {
    graph *Graph
}

func NewDependencyExtractor() *DependencyExtractor {
    return &DependencyExtractor{
        graph: NewGraph(),
    }
}

func (e *DependencyExtractor) ExtractFromStack(stack *schema.Stack, services []*schema.Service, components map[string][]interface{}) (*Graph, error) {
    // Add all components as nodes
    for serviceName, comps := range components {
        for _, comp := range comps {
            id := e.getComponentID(serviceName, comp)
            kind := e.getComponentKind(comp)
            e.graph.AddNode(id, kind, comp)
        }
    }
    
    // Extract explicit dependencies (dependsOn)
    for serviceName, comps := range components {
        for _, comp := range comps {
            fromID := e.getComponentID(serviceName, comp)
            
            deps := e.getExplicitDependencies(comp)
            for _, dep := range deps {
                e.graph.AddEdge(fromID, dep)
            }
        }
    }
    
    // Extract implicit dependencies (valueFrom references)
    for serviceName, comps := range components {
        for _, comp := range comps {
            fromID := e.getComponentID(serviceName, comp)
            
            deps := e.getImplicitDependencies(comp)
            for _, dep := range deps {
                e.graph.AddEdge(fromID, dep)
            }
        }
    }
    
    return e.graph, nil
}

func (e *DependencyExtractor) getComponentID(serviceName string, comp interface{}) string {
    switch c := comp.(type) {
    case *schema.MicroService:
        return fmt.Sprintf("%s/%s", c.Metadata.Service, c.Metadata.Name)
    case *schema.RDS:
        return fmt.Sprintf("%s/%s", c.Metadata.Service, c.Metadata.Name)
    // ... other types
    default:
        return ""
    }
}

func (e *DependencyExtractor) getComponentKind(comp interface{}) string {
    switch comp.(type) {
    case *schema.MicroService:
        return "MicroService"
    case *schema.RDS:
        return "RDS"
    // ... other types
    default:
        return "Unknown"
    }
}

func (e *DependencyExtractor) getExplicitDependencies(comp interface{}) []string {
    switch c := comp.(type) {
    case *schema.MicroService:
        return c.Spec.DependsOn
    // ... other types
    default:
        return nil
    }
}

func (e *DependencyExtractor) getImplicitDependencies(comp interface{}) []string {
    var deps []string
    
    switch c := comp.(type) {
    case *schema.MicroService:
        for _, env := range c.Spec.Environment {
            if env.ValueFrom != nil && env.ValueFrom.Component != "" {
                deps = append(deps, env.ValueFrom.Component)
            }
        }
    // ... other types
    }
    
    return deps
}
```

**Day 5: Testing**

```go
// pkg/graph/graph_test.go
func TestGraph_TopologicalSort(t *testing.T) {
    g := NewGraph()
    
    // Add nodes
    g.AddNode("a", "MicroService", nil)
    g.AddNode("b", "RDS", nil)
    g.AddNode("c", "ElastiCache", nil)
    g.AddNode("d", "MicroService", nil)
    
    // Add edges (a depends on b and c, d depends on a)
    g.AddEdge("a", "b")
    g.AddEdge("a", "c")
    g.AddEdge("d", "a")
    
    waves, err := g.TopologicalSort()
    require.NoError(t, err)
    require.Len(t, waves, 3)
    
    // Wave 1: b and c (no dependencies)
    assert.Len(t, waves[0], 2)
    
    // Wave 2: a (depends on b and c)
    assert.Len(t, waves[1], 1)
    assert.Equal(t, "a", waves[1][0].ID)
    
    // Wave 3: d (depends on a)
    assert.Len(t, waves[2], 1)
    assert.Equal(t, "d", waves[2][0].ID)
}

func TestGraph_DetectCycles(t *testing.T) {
    g := NewGraph()
    
    g.AddNode("a", "MicroService", nil)
    g.AddNode("b", "MicroService", nil)
    g.AddNode("c", "MicroService", nil)
    
    // Create cycle: a -> b -> c -> a
    g.AddEdge("a", "b")
    g.AddEdge("b", "c")
    g.AddEdge("c", "a")
    
    cycles, err := g.DetectCycles()
    assert.Error(t, err)
    assert.NotEmpty(t, cycles)
}
```

#### Acceptance Criteria
- ✅ Dependency graph can be built from components
- ✅ Explicit dependencies (dependsOn) are extracted
- ✅ Implicit dependencies (valueFrom) are extracted
- ✅ Topological sort produces correct deployment waves
- ✅ Cycles are detected and reported
- ✅ Parallel deployment groups are identified
- ✅ Tests pass with 80%+ coverage

---

## Phase 5: Reconciliation Engine

### Week 7-8: Days 1-10

#### Implementation Tasks

**Day 1-3: State Differ**

Create `pkg/reconciler/differ.go`:

```go
package reconciler

import (
    "github.com/company/panka/pkg/state"
    "github.com/company/panka/pkg/parser/schema"
)

type ChangeType string

const (
    ChangeTypeCreate  ChangeType = "CREATE"
    ChangeTypeUpdate  ChangeType = "UPDATE"
    ChangeTypeReplace ChangeType = "REPLACE"
    ChangeTypeDelete  ChangeType = "DELETE"
    ChangeTypeNoOp    ChangeType = "NO_OP"
)

type Change struct {
    ResourceID  string
    Type        ChangeType
    OldState    *state.Resource
    NewState    interface{}
    Differences []Difference
}

type Difference struct {
    Path     string
    OldValue interface{}
    NewValue interface{}
}

type Differ struct{}

func NewDiffer() *Differ {
    return &Differ{}
}

func (d *Differ) ComputeDiff(currentState *state.State, desiredComponents map[string]interface{}) ([]*Change, error) {
    changes := make([]*Change, 0)
    
    // Find creates and updates
    for id, desired := range desiredComponents {
        if current, exists := currentState.Resources[id]; exists {
            // Resource exists - check if changed
            if d.hasChanged(current, desired) {
                changeType := ChangeTypeUpdate
                if d.requiresReplacement(current, desired) {
                    changeType = ChangeTypeReplace
                }
                
                changes = append(changes, &Change{
                    ResourceID:  id,
                    Type:        changeType,
                    OldState:    current,
                    NewState:    desired,
                    Differences: d.getDifferences(current, desired),
                })
            } else {
                changes = append(changes, &Change{
                    ResourceID: id,
                    Type:       ChangeTypeNoOp,
                    OldState:   current,
                    NewState:   desired,
                })
            }
        } else {
            // New resource
            changes = append(changes, &Change{
                ResourceID: id,
                Type:       ChangeTypeCreate,
                NewState:   desired,
            })
        }
    }
    
    // Find deletes
    for id, current := range currentState.Resources {
        if _, exists := desiredComponents[id]; !exists {
            changes = append(changes, &Change{
                ResourceID: id,
                Type:       ChangeTypeDelete,
                OldState:   current,
            })
        }
    }
    
    return changes, nil
}

func (d *Differ) hasChanged(current *state.Resource, desired interface{}) bool {
    // Deep comparison of desired config
    // Implementation depends on component type
    return true // Placeholder
}

func (d *Differ) requiresReplacement(current *state.Resource, desired interface{}) bool {
    // Check if changes require resource replacement
    // e.g., changing RDS instance class might require replacement
    return false // Placeholder
}

func (d *Differ) getDifferences(current *state.Resource, desired interface{}) []Difference {
    // Detailed field-by-field comparison
    return []Difference{} // Placeholder
}
```

**Day 4-6: Plan Generator**

Create `pkg/reconciler/planner.go`:

```go
package reconciler

import (
    "fmt"
    "time"
    
    "github.com/company/panka/pkg/graph"
)

type ExecutionPlan struct {
    Stack          string
    Environment    string
    DeploymentID   string
    Timestamp      time.Time
    Waves          []Wave
    Summary        PlanSummary
    CostEstimate   *CostEstimate
}

type Wave struct {
    Number   int
    Actions  []*Action
    Parallel bool
}

type Action struct {
    ResourceID  string
    Type        ChangeType
    Component   interface{}
    Change      *Change
    EstimatedDuration time.Duration
}

type PlanSummary struct {
    TotalResources int
    ToCreate       int
    ToUpdate       int
    ToReplace      int
    ToDelete       int
    NoOp           int
}

type CostEstimate struct {
    CurrentMonthlyCost  float64
    NewMonthlyCost      float64
    MonthlyCostChange   float64
    PercentChange       float64
    Breakdown          map[string]float64
}

type Planner struct {
    differ *Differ
}

func NewPlanner() *Planner {
    return &Planner{
        differ: NewDiffer(),
    }
}

func (p *Planner) GeneratePlan(
    stack, environment string,
    currentState *state.State,
    desiredComponents map[string]interface{},
    dependencyGraph *graph.Graph,
) (*ExecutionPlan, error) {
    
    // Compute diff
    changes, err := p.differ.ComputeDiff(currentState, desiredComponents)
    if err != nil {
        return nil, err
    }
    
    // Get deployment waves from topological sort
    waves, err := dependencyGraph.TopologicalSort()
    if err != nil {
        return nil, err
    }
    
    // Build execution plan
    plan := &ExecutionPlan{
        Stack:        stack,
        Environment:  environment,
        DeploymentID: generateDeploymentID(),
        Timestamp:    time.Now(),
        Waves:        make([]Wave, 0),
    }
    
    // Create waves with actions
    for waveNum, waveNodes := range waves {
        wave := Wave{
            Number:   waveNum + 1,
            Actions:  make([]*Action, 0),
            Parallel: len(waveNodes) > 1,
        }
        
        for _, node := range waveNodes {
            // Find change for this node
            var change *Change
            for _, c := range changes {
                if c.ResourceID == node.ID {
                    change = c
                    break
                }
            }
            
            if change != nil && change.Type != ChangeTypeNoOp {
                wave.Actions = append(wave.Actions, &Action{
                    ResourceID:        node.ID,
                    Type:              change.Type,
                    Component:         node.Component,
                    Change:            change,
                    EstimatedDuration: p.estimateDuration(change),
                })
            }
        }
        
        if len(wave.Actions) > 0 {
            plan.Waves = append(plan.Waves, wave)
        }
    }
    
    // Generate summary
    plan.Summary = p.generateSummary(changes)
    
    // Estimate cost
    plan.CostEstimate = p.estimateCost(currentState, desiredComponents, changes)
    
    return plan, nil
}

func (p *Planner) generateSummary(changes []*Change) PlanSummary {
    summary := PlanSummary{}
    
    for _, change := range changes {
        summary.TotalResources++
        switch change.Type {
        case ChangeTypeCreate:
            summary.ToCreate++
        case ChangeTypeUpdate:
            summary.ToUpdate++
        case ChangeTypeReplace:
            summary.ToReplace++
        case ChangeTypeDelete:
            summary.ToDelete++
        case ChangeTypeNoOp:
            summary.NoOp++
        }
    }
    
    return summary
}

func (p *Planner) estimateDuration(change *Change) time.Duration {
    // Estimate based on change type and resource kind
    switch change.Type {
    case ChangeTypeCreate:
        return 5 * time.Minute
    case ChangeTypeUpdate:
        return 3 * time.Minute
    case ChangeTypeReplace:
        return 8 * time.Minute
    case ChangeTypeDelete:
        return 2 * time.Minute
    default:
        return 0
    }
}

func (p *Planner) estimateCost(currentState *state.State, desiredComponents map[string]interface{}, changes []*Change) *CostEstimate {
    // Cost estimation logic
    // This would integrate with AWS Pricing API
    return &CostEstimate{
        CurrentMonthlyCost: 100.0,
        NewMonthlyCost:     120.0,
        MonthlyCostChange:  20.0,
        PercentChange:      20.0,
        Breakdown: map[string]float64{
            "ECS":         50.0,
            "RDS":         60.0,
            "ElastiCache": 10.0,
        },
    }
}

func generateDeploymentID() string {
    return fmt.Sprintf("dep-%d", time.Now().Unix())
}
```

**Day 7-8: Plan Formatter**

Create `pkg/reconciler/formatter.go`:

```go
package reconciler

import (
    "fmt"
    "strings"
)

type PlanFormatter struct{}

func NewPlanFormatter() *PlanFormatter {
    return &PlanFormatter{}
}

func (f *PlanFormatter) Format(plan *ExecutionPlan) string {
    var sb strings.Builder
    
    // Header
    sb.WriteString("┌─────────────────────────────────────────────────────────┐\n")
    sb.WriteString(fmt.Sprintf("│ Deployment Plan: %s (%s)\n", plan.Stack, plan.Environment))
    sb.WriteString("├─────────────────────────────────────────────────────────┤\n")
    sb.WriteString("\n")
    
    // Waves
    for _, wave := range plan.Waves {
        parallel := ""
        if wave.Parallel {
            parallel = " (parallel)"
        }
        sb.WriteString(fmt.Sprintf("Wave %d%s:\n", wave.Number, parallel))
        
        for _, action := range wave.Actions {
            symbol := f.getSymbol(action.Type)
            sb.WriteString(fmt.Sprintf("  %s %s (%s) %s\n",
                symbol,
                action.ResourceID,
                f.getKind(action.Component),
                action.Type))
            
            if action.Change != nil && len(action.Change.Differences) > 0 {
                for _, diff := range action.Change.Differences {
                    sb.WriteString(fmt.Sprintf("    - %s: %v → %v\n",
                        diff.Path, diff.OldValue, diff.NewValue))
                }
            }
        }
        sb.WriteString("\n")
    }
    
    // Summary
    sb.WriteString("├─────────────────────────────────────────────────────────┤\n")
    sb.WriteString("Summary:\n")
    sb.WriteString(fmt.Sprintf("  + %d to create\n", plan.Summary.ToCreate))
    sb.WriteString(fmt.Sprintf("  ✓ %d to update\n", plan.Summary.ToUpdate))
    sb.WriteString(fmt.Sprintf("  ⚠ %d to replace\n", plan.Summary.ToReplace))
    sb.WriteString(fmt.Sprintf("  - %d to delete\n", plan.Summary.ToDelete))
    sb.WriteString("\n")
    
    // Cost estimate
    if plan.CostEstimate != nil {
        sb.WriteString(fmt.Sprintf("Estimated cost change: $%.2f/month (%+.1f%%)\n",
            plan.CostEstimate.MonthlyCostChange,
            plan.CostEstimate.PercentChange))
    }
    
    sb.WriteString("└─────────────────────────────────────────────────────────┘\n")
    
    return sb.String()
}

func (f *PlanFormatter) getSymbol(changeType ChangeType) string {
    switch changeType {
    case ChangeTypeCreate:
        return "+"
    case ChangeTypeUpdate:
        return "✓"
    case ChangeTypeReplace:
        return "⚠"
    case ChangeTypeDelete:
        return "-"
    default:
        return " "
    }
}

func (f *PlanFormatter) getKind(component interface{}) string {
    // Type assertion to get kind
    return "Component" // Placeholder
}
```

**Day 9-10: Testing**

```go
// pkg/reconciler/differ_test.go
func TestDiffer_ComputeDiff(t *testing.T) {
    differ := NewDiffer()
    
    currentState := &state.State{
        Resources: map[string]*state.Resource{
            "service/api": {
                Kind: "MicroService",
                DesiredConfig: map[string]interface{}{
                    "image": "test:v1.0.0",
                    "replicas": 3,
                },
            },
        },
    }
    
    desiredComponents := map[string]interface{}{
        "service/api": &schema.MicroService{
            Spec: schema.MicroServiceSpec{
                Image: schema.ImageConfig{
                    Repository: "test",
                    Tag:        "v1.1.0", // Changed
                },
            },
        },
    }
    
    changes, err := differ.ComputeDiff(currentState, desiredComponents)
    require.NoError(t, err)
    require.Len(t, changes, 1)
    
    assert.Equal(t, ChangeTypeUpdate, changes[0].Type)
    assert.Equal(t, "service/api", changes[0].ResourceID)
}

// pkg/reconciler/planner_test.go
func TestPlanner_GeneratePlan(t *testing.T) {
    planner := NewPlanner()
    
    // Setup test data
    currentState := &state.State{
        Resources: make(map[string]*state.Resource),
    }
    
    desiredComponents := map[string]interface{}{
        "service/database": &schema.RDS{},
        "service/api": &schema.MicroService{
            Spec: schema.MicroServiceSpec{
                DependsOn: []string{"service/database"},
            },
        },
    }
    
    graph := graph.NewGraph()
    graph.AddNode("service/database", "RDS", desiredComponents["service/database"])
    graph.AddNode("service/api", "MicroService", desiredComponents["service/api"])
    graph.AddEdge("service/api", "service/database")
    
    plan, err := planner.GeneratePlan("test-stack", "dev", currentState, desiredComponents, graph)
    require.NoError(t, err)
    require.NotNil(t, plan)
    
    // Database should be in wave 1, API in wave 2
    assert.Len(t, plan.Waves, 2)
    assert.Equal(t, 1, plan.Waves[0].Number)
    assert.Equal(t, 2, plan.Waves[1].Number)
}
```

#### Acceptance Criteria
- ✅ State diff correctly identifies creates, updates, deletes
- ✅ Replacement detection works
- ✅ Execution plan generated with correct waves
- ✅ Plan summary is accurate
- ✅ Cost estimation works
- ✅ Plan formatter produces readable output
- ✅ Tests pass with 80%+ coverage

---

## Phase 6: Pulumi Integration

### Week 9-10: Days 1-10

#### Implementation Tasks

**Day 1-3: Pulumi Wrapper**

Create `pkg/pulumi/pulumi.go`:

```go
package pulumi

import (
    "context"
    "fmt"
    
    "github.com/pulumi/pulumi/sdk/v3/go/auto"
    "github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
    "github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type PulumiExecutor struct {
    projectName string
    stackName   string
    workDir     string
}

func NewPulumiExecutor(projectName, stackName, workDir string) *PulumiExecutor {
    return &PulumiExecutor{
        projectName: projectName,
        stackName:   stackName,
        workDir:     workDir,
    }
}

func (e *PulumiExecutor) Preview(ctx context.Context, program pulumi.RunFunc) error {
    stack, err := e.getOrCreateStack(ctx, program)
    if err != nil {
        return err
    }
    
    _, err = stack.Preview(ctx, optpreview.ProgressStreams(os.Stdout))
    return err
}

func (e *PulumiExecutor) Up(ctx context.Context, program pulumi.RunFunc) (auto.UpResult, error) {
    stack, err := e.getOrCreateStack(ctx, program)
    if err != nil {
        return auto.UpResult{}, err
    }
    
    return stack.Up(ctx, optup.ProgressStreams(os.Stdout))
}

func (e *PulumiExecutor) Destroy(ctx context.Context, program pulumi.RunFunc) error {
    stack, err := e.getOrCreateStack(ctx, program)
    if err != nil {
        return err
    }
    
    _, err = stack.Destroy(ctx)
    return err
}

func (e *PulumiExecutor) getOrCreateStack(ctx context.Context, program pulumi.RunFunc) (auto.Stack, error) {
    return auto.UpsertStackInlineSource(ctx, e.stackName, e.projectName, program,
        auto.WorkDir(e.workDir))
}
```

**Day 4-7: Component Translators**

Create `pkg/pulumi/translators/microservice.go`:

```go
package translators

import (
    "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
    "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
    "github.com/company/panka/pkg/parser/schema"
)

type MicroServiceTranslator struct{}

func NewMicroServiceTranslator() *MicroServiceTranslator {
    return &MicroServiceTranslator{}
}

func (t *MicroServiceTranslator) Translate(ctx *pulumi.Context, ms *schema.MicroService, infra *schema.ComponentInfra) error {
    // Create task definition
    taskDef, err := ecs.NewTaskDefinition(ctx, ms.Metadata.Name+"-task", &ecs.TaskDefinitionArgs{
        Family:                  pulumi.String(ms.Metadata.Name),
        Cpu:                     pulumi.String(fmt.Sprintf("%d", infra.Spec.Resources.CPU)),
        Memory:                  pulumi.String(fmt.Sprintf("%d", infra.Spec.Resources.Memory)),
        NetworkMode:             pulumi.String("awsvpc"),
        RequiresCompatibilities: pulumi.StringArray{pulumi.String("FARGATE")},
        ExecutionRoleArn:        pulumi.String("..."),
        TaskRoleArn:             pulumi.String("..."),
        
        ContainerDefinitions: pulumi.String(t.buildContainerDefinitions(ms)),
    })
    if err != nil {
        return err
    }
    
    // Create target group
    targetGroup, err := lb.NewTargetGroup(ctx, ms.Metadata.Name+"-tg", &lb.TargetGroupArgs{
        Port:       pulumi.Int(ms.Spec.Ports[0].Port),
        Protocol:   pulumi.String("HTTP"),
        VpcId:      pulumi.String("..."),
        TargetType: pulumi.String("ip"),
        
        HealthCheck: &lb.TargetGroupHealthCheckArgs{
            Path:                pulumi.String(ms.Spec.HealthCheck.Readiness.HTTP.Path),
            Interval:            pulumi.Int(ms.Spec.HealthCheck.Readiness.PeriodSeconds),
            Timeout:             pulumi.Int(ms.Spec.HealthCheck.Readiness.TimeoutSeconds),
            HealthyThreshold:    pulumi.Int(ms.Spec.HealthCheck.Readiness.SuccessThreshold),
            UnhealthyThreshold:  pulumi.Int(ms.Spec.HealthCheck.Readiness.FailureThreshold),
        },
    })
    if err != nil {
        return err
    }
    
    // Create ECS service
    _, err = ecs.NewService(ctx, ms.Metadata.Name, &ecs.ServiceArgs{
        Cluster:        pulumi.String("..."),
        TaskDefinition: taskDef.Arn,
        DesiredCount:   pulumi.Int(infra.Spec.Scaling.Replicas),
        LaunchType:     pulumi.String("FARGATE"),
        
        NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
            Subnets:        pulumi.StringArray{pulumi.String("...")},
            SecurityGroups: pulumi.StringArray{pulumi.String("...")},
        },
        
        LoadBalancers: ecs.ServiceLoadBalancerArray{
            &ecs.ServiceLoadBalancerArgs{
                TargetGroupArn: targetGroup.Arn,
                ContainerName:  pulumi.String(ms.Metadata.Name),
                ContainerPort:  pulumi.Int(ms.Spec.Ports[0].Port),
            },
        },
    })
    
    return err
}

func (t *MicroServiceTranslator) buildContainerDefinitions(ms *schema.MicroService) string {
    // Build container definition JSON
    return `[...]` // JSON string
}
```

Create `pkg/pulumi/translators/rds.go`:

```go
package translators

import (
    "github.com/pulumi/pulumi-aws/sdk/v6/go/aws/rds"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
    "github.com/company/panka/pkg/parser/schema"
)

type RDSTranslator struct{}

func NewRDSTranslator() *RDSTranslator {
    return &RDSTranslator{}
}

func (t *RDSTranslator) Translate(ctx *pulumi.Context, rdsConfig *schema.RDS) error {
    _, err := rds.NewInstance(ctx, rdsConfig.Metadata.Name, &rds.InstanceArgs{
        Engine:              pulumi.String(rdsConfig.Spec.Engine.Type),
        EngineVersion:       pulumi.String(rdsConfig.Spec.Engine.Version),
        InstanceClass:       pulumi.String(rdsConfig.Spec.Instance.Class),
        AllocatedStorage:    pulumi.Int(rdsConfig.Spec.Instance.Storage.AllocatedGB),
        StorageType:         pulumi.String(rdsConfig.Spec.Instance.Storage.Type),
        DbName:              pulumi.String(rdsConfig.Spec.Database.Name),
        Username:            pulumi.String(rdsConfig.Spec.Database.Username),
        Password:            pulumi.String("..."), // From secrets
        MultiAz:             pulumi.Bool(rdsConfig.Spec.Availability.MultiAZ),
        StorageEncrypted:    pulumi.Bool(rdsConfig.Spec.Encryption.AtRest),
        BackupRetentionPeriod: pulumi.Int(rdsConfig.Spec.Backup.Automated.RetentionDays),
        DbSubnetGroupName:   pulumi.String("..."),
        VpcSecurityGroupIds: pulumi.StringArray{pulumi.String("...")},
        SkipFinalSnapshot:   pulumi.Bool(false),
    })
    
    return err
}
```

**Day 8-10: Executor Integration**

Create `pkg/executor/executor.go`:

```go
package executor

import (
    "context"
    "fmt"
    
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
    "github.com/company/panka/pkg/pulumi"
    "github.com/company/panka/pkg/pulumi/translators"
    "github.com/company/panka/pkg/reconciler"
)

type Executor struct {
    pulumiExecutor *pulumi.PulumiExecutor
    translators    map[string]Translator
}

type Translator interface {
    Translate(ctx *pulumi.Context, component interface{}, infra interface{}) error
}

func NewExecutor(projectName, stackName, workDir string) *Executor {
    return &Executor{
        pulumiExecutor: pulumi.NewPulumiExecutor(projectName, stackName, workDir),
        translators: map[string]Translator{
            "MicroService":      translators.NewMicroServiceTranslator(),
            "RDS":               translators.NewRDSTranslator(),
            "ElastiCacheRedis":  translators.NewElastiCacheTranslator(),
            "S3":                translators.NewS3Translator(),
            "SQS":               translators.NewSQSTranslator(),
        },
    }
}

func (e *Executor) Execute(ctx context.Context, plan *reconciler.ExecutionPlan) error {
    // Create Pulumi program
    program := func(ctx *pulumi.Context) error {
        for _, wave := range plan.Waves {
            for _, action := range wave.Actions {
                if err := e.executeAction(ctx, action); err != nil {
                    return fmt.Errorf("failed to execute action %s: %w", action.ResourceID, err)
                }
            }
        }
        return nil
    }
    
    // Execute via Pulumi
    _, err := e.pulumiExecutor.Up(ctx, program)
    return err
}

func (e *Executor) executeAction(ctx *pulumi.Context, action *reconciler.Action) error {
    kind := e.getKind(action.Component)
    
    translator, ok := e.translators[kind]
    if !ok {
        return fmt.Errorf("no translator for kind: %s", kind)
    }
    
    return translator.Translate(ctx, action.Component, nil)
}

func (e *Executor) getKind(component interface{}) string {
    // Type assertion
    return "Unknown" // Placeholder
}
```

#### Acceptance Criteria
- ✅ Pulumi wrapper functional
- ✅ MicroService translator creates ECS resources
- ✅ RDS translator creates database
- ✅ S3, SQS, ElastiCache translators work
- ✅ Executor orchestrates deployment
- ✅ Integration tests with LocalStack pass
- ✅ Tests pass with 70%+ coverage

---

## Phase 7: Component Implementations

### Week 11-13: Days 1-15

[Implement all remaining component translators similarly to Phase 6]

Component list:
- Worker, CronJob, Lambda (compute)
- DynamoDB, DocumentDB (database)
- ElastiCache variants (cache)
- EFS, EBS (storage)
- SNS, MSK, EventBridge (messaging)
- ALB, NLB, CloudFront, APIGateway (networking)

Each component gets:
- Schema definition
- Translator implementation
- Unit tests
- Integration tests

---

## Phase 8: CLI & User Experience

### Week 14-15: Days 1-10

#### Implementation Tasks

**Day 1-3: CLI Framework**

Create `cmd/panka/main.go`:

```go
package main

import (
    "os"
    
    "github.com/spf13/cobra"
    "github.com/company/panka/internal/cli"
)

func main() {
    rootCmd := cli.NewRootCommand()
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

Create `internal/cli/root.go`:

```go
package cli

import (
    "github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "panka",
        Short: "Panka manages application deployments on AWS",
        Long:  `Panka is a deployment management system for AWS...`,
    }
    
    // Add subcommands
    cmd.AddCommand(NewApplyCommand())
    cmd.AddCommand(NewPlanCommand())
    cmd.AddCommand(NewDestroyCommand())
    cmd.AddCommand(NewValidateCommand())
    cmd.AddCommand(NewStatusCommand())
    cmd.AddCommand(NewLogsCommand())
    cmd.AddCommand(NewHistoryCommand())
    cmd.AddCommand(NewDriftCommand())
    cmd.AddCommand(NewRollbackCommand())
    cmd.AddCommand(NewStateCommand())
    cmd.AddCommand(NewUnlockCommand())
    
    return cmd
}
```

**Day 4-6: Core Commands**

Create `internal/cli/apply.go`:

```go
package cli

import (
    "context"
    "fmt"
    
    "github.com/spf13/cobra"
    "github.com/company/panka/pkg/panka"
)

func NewApplyCommand() *cobra.Command {
    var (
        stack       string
        service     string
        component   string
        environment string
        version     string
        autoApprove bool
    )
    
    cmd := &cobra.Command{
        Use:   "apply",
        Short: "Deploy stack/service/component",
        RunE: func(cmd *cobra.Command, args []string) error {
            d := panka.New()
            
            return d.Apply(context.Background(), &panka.ApplyOptions{
                Stack:       stack,
                Service:     service,
                Component:   component,
                Environment: environment,
                Version:     version,
                AutoApprove: autoApprove,
            })
        },
    }
    
    cmd.Flags().StringVar(&stack, "stack", "", "Stack name (required)")
    cmd.Flags().StringVar(&service, "service", "", "Service name (optional)")
    cmd.Flags().StringVar(&component, "component", "", "Component name (optional)")
    cmd.Flags().StringVar(&environment, "environment", "", "Environment (required)")
    cmd.Flags().StringVar(&version, "var VERSION", "", "Version to deploy (required)")
    cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip approval prompt")
    
    cmd.MarkFlagRequired("stack")
    cmd.MarkFlagRequired("environment")
    
    return cmd
}
```

**Day 7-10: Progress UI & Output**

```go
// internal/cli/ui/progress.go
package ui

import (
    "fmt"
    "time"
    
    "github.com/schollz/progressbar/v3"
)

type ProgressBar struct {
    bar *progressbar.ProgressBar
}

func NewProgressBar(max int, description string) *ProgressBar {
    bar := progressbar.NewOptions(max,
        progressbar.OptionSetDescription(description),
        progressbar.OptionSetWidth(50),
        progressbar.OptionShowCount(),
        progressbar.OptionSetTheme(progressbar.Theme{
            Saucer:        "=",
            SaucerHead:    ">",
            SaucerPadding: " ",
            BarStart:      "[",
            BarEnd:        "]",
        }),
    )
    
    return &ProgressBar{bar: bar}
}

func (p *ProgressBar) Increment() {
    p.bar.Add(1)
}
```

---

## Phase 9: Advanced Features

### Week 16-17: Days 1-10

#### Implementation Tasks

**Drift Detection**
**Policy Validation (OPA)**
**Multi-Region Support**
**Auto-Rollback**
**Advanced Metrics**

[Similar detailed implementation as previous phases]

---

## Phase 10: Production Readiness

### Week 18: Days 1-5

#### Tasks

**Day 1-2: Performance Testing**
- Load testing with 1000+ resources
- Concurrent deployment testing
- Lock contention testing

**Day 3: Security Audit**
- IAM permissions review
- Secrets management audit
- Network security review

**Day 4: Documentation**
- Complete all documentation
- Create video tutorials
- Write runbooks

**Day 5: Production Deployment**
- Deploy infrastructure
- Migrate pilot team
- Monitor and iterate

---

## Testing Strategy

### Unit Testing

**Coverage Target: 80%+**

```bash
# Run unit tests
make test

# With coverage
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# View coverage
go tool cover -html=coverage.txt
```

### Integration Testing

**Using LocalStack**

```bash
# Start LocalStack
make dev

# Run integration tests
make test-integration
```

**Test Categories:**
- State Manager + S3
- Lock Manager + DynamoDB
- Parser + Validator
- Graph Builder
- Pulumi Integration

### End-to-End Testing

**Test Scenarios:**

1. **Deploy Simple Stack**
```bash
# Test: Deploy single MicroService with RDS
panka apply --stack test-simple --environment dev
```

2. **Deploy Complex Stack**
```bash
# Test: Deploy multi-service stack with dependencies
panka apply --stack test-complex --environment dev
```

3. **Update Deployment**
```bash
# Test: Update existing deployment
panka apply --stack test-simple --environment dev --var VERSION=v1.1.0
```

4. **Rollback**
```bash
# Test: Rollback to previous version
panka rollback --stack test-simple --environment dev
```

5. **Drift Detection**
```bash
# Test: Detect and remediate drift
panka drift detect --stack test-simple --environment dev
panka drift remediate --stack test-simple --environment dev
```

6. **Concurrent Deployments**
```bash
# Test: Multiple teams deploying simultaneously
panka apply --stack stack1 --service service-a --environment dev &
panka apply --stack stack1 --service service-b --environment dev &
wait
```

### Performance Testing

**Load Test Script:**

```go
// test/performance/load_test.go
func TestConcurrentDeployments(t *testing.T) {
    numStacks := 10
    numConcurrent := 5
    
    semaphore := make(chan struct{}, numConcurrent)
    var wg sync.WaitGroup
    
    for i := 0; i < numStacks; i++ {
        wg.Add(1)
        go func(stackNum int) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            // Deploy stack
            err := panka.Apply(ctx, &panka.ApplyOptions{
                Stack: fmt.Sprintf("perf-test-%d", stackNum),
                Environment: "dev",
            })
            
            assert.NoError(t, err)
        }(i)
    }
    
    wg.Wait()
}
```

### Security Testing

**Checklist:**
- [ ] IAM permissions follow least privilege
- [ ] Secrets never logged
- [ ] State files encrypted
- [ ] TLS for all connections
- [ ] Input validation on all APIs
- [ ] No hardcoded credentials
- [ ] Security scanning in CI/CD

### User Acceptance Testing

**Test with Real Teams:**

1. **Week 16**: Internal platform team
2. **Week 17**: Pilot team (1 service)
3. **Week 18**: Expanded pilot (3 teams, 10 services)

**Feedback Metrics:**
- Time to deploy new service
- Number of support requests
- User satisfaction score
- Adoption rate

---

## Deployment & Rollout Plan

### Phase 1: Infrastructure Setup (Week 18, Day 1)

```bash
# Deploy AWS infrastructure
cd infrastructure/terraform
terraform apply

# Verify
aws s3 ls
aws dynamodb list-tables
```

### Phase 2: Internal Testing (Week 18, Day 2)

```bash
# Deploy platform team's services
panka apply --stack platform-internal --environment dev
panka apply --stack platform-internal --environment staging
```

### Phase 3: Pilot Rollout (Week 18, Day 3-4)

**Pilot Team: Notifications**

```bash
# Migrate notification-service
panka apply --stack notification-platform --environment dev
panka apply --stack notification-platform --environment staging
panka apply --stack notification-platform --environment production
```

**Success Criteria:**
- Deployment completes successfully
- No production incidents
- Team provides positive feedback

### Phase 4: Full Rollout (Week 18, Day 5+)

**Rollout Schedule:**
- Week 19: 3 more teams
- Week 20: 5 more teams
- Week 21: All remaining teams

**Communication Plan:**
- Announcement email (Week 17)
- Training sessions (Week 18-19)
- Office hours (Daily during rollout)
- Documentation site live
- Slack support channel

---

## Success Metrics

### Technical Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Deployment Success Rate | >99% | CloudWatch metrics |
| Average Deployment Time | <5 min | Timing logs |
| Lock Contention Rate | <1% | DynamoDB metrics |
| State Consistency | 100% | Validation checks |
| Test Coverage | >80% | Go test coverage |

### User Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Time to Deploy New Service | <30 min | User surveys |
| Support Requests | <5/week | Ticket system |
| User Satisfaction | >4/5 | Quarterly survey |
| Adoption Rate | 100% of teams | Usage tracking |
| Documentation Clarity | >4/5 | Feedback form |

### Business Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Deployment Frequency | 10x increase | Git analytics |
| MTTR (Mean Time to Recovery) | <10 min | Incident logs |
| Infrastructure Costs | Neutral or lower | AWS Cost Explorer |
| Developer Productivity | +20% | Sprint velocity |

---

## Risk Management

### Risk Matrix

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| State corruption | Low | High | S3 versioning, backups, validation |
| Lock failures | Medium | Medium | TTL cleanup, force-unlock |
| Pulumi integration issues | Medium | High | Extensive testing, fallback plan |
| Performance issues | Low | Medium | Load testing, optimization |
| Security vulnerabilities | Low | High | Security audit, penetration testing |
| Adoption resistance | Medium | Medium | Training, documentation, support |

---

## Conclusion

This implementation plan provides a comprehensive roadmap for building the panka system from scratch to production-ready deployment. The phased approach ensures:

1. **Solid Foundation**: Core infrastructure first
2. **Iterative Development**: Each phase builds on previous
3. **Continuous Testing**: Tests at every phase
4. **Early Validation**: Pilot rollout before full deployment
5. **User Focus**: Documentation and UX throughout

**Total Timeline: 18 weeks**

**Team Required:**
- 2-3 Backend Engineers (Go)
- 1 DevOps/Platform Engineer
- 1 QA Engineer (part-time)
- 1 Technical Writer (part-time)

**Next Steps:**
1. Review and approve plan
2. Provision development infrastructure
3. Set up project repository
4. Kick off Phase 0
