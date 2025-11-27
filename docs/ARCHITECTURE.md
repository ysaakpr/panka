# Panka Architecture

## Overview

Panka is a Golang-based deployment management system for managing application deployments on AWS using ECS/Fargate/EKS with Pulumi as the backend orchestrator.

## Initial Setup

### Step 1: Install CLI

```bash
# Install panka
curl -sSL https://panka.io/install.sh | sh

# Or download from releases
# Or build from source
```

### Step 2: Configure Backend

User provides their own infrastructure:

```bash
# Initialize panka (interactive)
panka init

# Prompts for:
# - AWS Region
# - S3 Bucket name (for state)
# - DynamoDB Table name (for locks)
# - AWS Profile (optional)

# Creates: ~/.panka/config.yaml
```

### Step 3: Create Backend Infrastructure (One-Time)

```bash
# Option A: Use panka to create
panka backend create \
  --bucket company-panka-state \
  --table company-panka-locks

# Option B: Use Terraform (provided)
cd infrastructure/terraform
terraform apply

# Option C: Create manually in AWS Console
```

**Required Resources:**
- S3 bucket (with versioning enabled)
- DynamoDB table (with TTL enabled)
- IAM permissions for CLI user/role

See [CLI_ARCHITECTURE.md](CLI_ARCHITECTURE.md) for complete details.

---

## Core Concepts

### Stack
A **stack** is the unit of deployment. It represents a complete environment (production, staging, development) containing multiple services.

### Service
A **service** is a logical grouping of related components (e.g., an API, its database, cache, and workers).

### Component
A **component** is a single deployable unit - can be:
- Container-based: MicroService, Worker, CronJob, Lambda
- Database: RDS, DynamoDB, DocumentDB
- Cache: ElastiCacheRedis, ElastiCacheMemcached, AWSMemoryDB
- Storage: S3, EFS, EBS
- Messaging: SQS, SNS, Kafka, MSK
- Networking: ALB, NLB, CloudFront, APIGateway

## System Architecture

**Important**: Panka is a **CLI tool** (like Terraform or Pulumi), not a backend service.

```
┌─────────────────────────────────────────────────────────────────┐
│                   USER'S MACHINE / CI RUNNER                      │
│                                                                   │
│                      panka CLI (Binary)                        │
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐       │
│  │  Discovery   │───▶│ Reconciler   │───▶│  Executor   │       │
│  │   Engine     │    │   Engine     │    │   Engine    │       │
│  └──────────────┘    └──────────────┘    └─────────────┘       │
│         │                   │                    │               │
│         ▼                   ▼                    ▼               │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐       │
│  │   Resource   │    │    State     │    │   Pulumi    │       │
│  │   Parser     │    │   Manager    │    │   Backend   │       │
│  └──────────────┘    └──────────────┘    └─────────────┘       │
│                             │                                     │
└─────────────────────────────┼─────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │   AWS Services   │
                    │  (User Provides) │
                    ├──────────────────┤
                    │ • S3 (State)     │
                    │ • DynamoDB (Lock)│
                    │ • ECS/Fargate    │
                    │ • RDS            │
                    │ • ElastiCache    │
                    │ • SQS, S3, etc.  │
                    └──────────────────┘
```

**Key Points**:
- CLI runs on user's machine or in CI/CD
- No "panka service" running in the cloud
- Users provide their own S3 bucket and DynamoDB table
- CLI exits after deployment completes

## API Groups

### `core.panka.io/v1`
- Stack
- Service

### `infra.panka.io/v1`
- InfraDefaults
- ServiceInfraDefaults
- ComponentInfra
- Networking
- Security
- Observability
- Compliance

### `components.panka.io/v1`
All deployable components:
- MicroService, Worker, CronJob, Lambda, EC2Instance
- RDS, DynamoDB, DocumentDB
- ElastiCacheRedis, ElastiCacheMemcached, AWSMemoryDB
- S3, EFS, EBS
- SQS, SNS, Kafka, MSK, EventBridge
- ALB, NLB, CloudFront, APIGateway

## State Management

### State Backend: S3
```
s3://company-panka-state/
├── stacks/
│   └── {stack-name}/
│       └── {environment}/
│           ├── state.json           # Current state
│           ├── history/             # State history
│           └── pulumi/              # Pulumi state
```

### Lock Backend: DynamoDB
```
Table: panka-state-locks

Primary Key: lockKey (String)
  Format: "stack:{stack-name}:env:{environment}"
  Example: "stack:user-platform:env:production"

Attributes:
- lockKey (String, Primary Key)
- lockId (String) - UUID of lock holder
- lockedAt (Number) - Unix timestamp
- lockedBy (String) - User/system identifier
- expiresAt (Number) - Unix timestamp for TTL
- metadata (Map) - Additional context
  - deployment_id
  - git_commit
  - ci_run_id
```

### Lock Granularity

```
Level 1: Stack-level lock (default)
  lockKey: "stack:user-platform:env:production"
  - Safest, simplest
  - Only one deployment per stack at a time

Level 2: Service-level lock (optional)
  lockKey: "stack:user-platform:env:production:service:user-service"
  - Multiple services can deploy concurrently
  - More complex dependency management

Level 3: Component-level lock (advanced)
  lockKey: "stack:user-platform:env:production:component:user-service/api"
  - Maximum parallelism
  - Most complex dependency tracking
```

## Execution Flow

### 1. Discovery Phase
```
Input: --stack user-platform --environment production

├── Scan directory structure recursively
├── Parse all YAML files (stack, services, components, infra)
├── Apply environment overlays
├── Resolve variables and references
├── Validate schemas and policies
└── Build ResourceGraph with dependencies
```

### 2. Dependency Resolution
```
├── Extract explicit dependencies (dependsOn)
├── Extract implicit dependencies (valueFrom)
├── Build dependency graph (DAG)
├── Detect cycles
├── Perform topological sort
└── Generate deployment waves (parallel execution groups)
```

### 3. State Reconciliation
```
├── Acquire distributed lock (DynamoDB)
├── Load current state from S3
├── Compute diff (desired vs current)
├── Generate execution plan
├── Display plan to user
├── Wait for approval (if required)
└── Proceed to execution
```

### 4. Execution
```
For each Wave:
  ├── Pre-wave validation
  ├── Execute resources in parallel
  │   ├── Pre-deployment hooks
  │   ├── Translate to Pulumi
  │   ├── Execute via Pulumi
  │   ├── Post-deployment verification
  │   ├── Update state
  │   └── Handle failures
  ├── Post-wave validation
  └── Proceed to next wave

├── Final verification
├── Save state
├── Release lock
└── Send notifications
```

## Reconciliation Loop

Runs periodically (default: every 5 minutes) or on-demand:

```
├── Discover current state (query AWS)
├── Load desired state (parse YAML)
├── Detect drift
├── Generate drift report
├── Auto-remediate (if enabled)
└── Send alerts for critical drift
```

## Rollback Strategy

### Automatic Rollback Triggers
- CloudWatch alarms
- Metric thresholds (error rate, latency)
- Health check failures
- Deployment timeout

### Rollback Process
```
├── Identify last known good state
├── Generate rollback plan (reverse order)
├── Execute rollback
│   ├── CREATED resources → DELETE
│   ├── UPDATED resources → RESTORE
│   ├── REPLACED resources → RECREATE old
│   └── DELETED resources → RECREATE
├── Verify state
├── Update state
└── Send notifications
```

## Security

### IAM Roles
- **PankaExecutionRole**: Used by panka CLI to manage AWS resources
- **TaskExecutionRole**: Used by ECS tasks to pull images and access secrets
- **TaskRole**: Used by running containers to access AWS services

### Secrets Management
- All secrets stored in AWS Secrets Manager
- Never stored in YAML files (only references)
- Rotation enabled by default
- Audit logging for all access

### Encryption
- State files encrypted at rest (S3-SSE)
- Secrets encrypted (Secrets Manager KMS)
- Database encryption at rest (RDS, DynamoDB)
- Transit encryption (TLS/SSL)

## Observability

### Logging
- Structured JSON logs
- Centralized in CloudWatch
- Correlation IDs for tracing
- Deployment audit trail

### Metrics
- Deployment duration
- Success/failure rates
- Resource counts
- Drift detection frequency
- Cost tracking

### Alerting
- Deployment failures
- Drift detection
- Security policy violations
- Cost threshold breaches

## Disaster Recovery

### State Backup
- Automatic versioning in S3
- History retained for 90 days
- Point-in-time recovery enabled
- Cross-region replication (optional)

### Lock Recovery
- TTL-based automatic cleanup (1 hour default)
- Manual force-unlock capability
- Stale lock detection
- Lock takeover for expired locks

