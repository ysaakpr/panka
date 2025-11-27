# Deployer CLI Architecture

The deployer is a **command-line tool** (similar to Pulumi or Terraform) that teams use to deploy their applications. There is no backend service - it's just a CLI binary that reads YAML, manages state in S3, uses DynamoDB for locking, and orchestrates deployments via Pulumi.

---

## Core Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                   │
│                         USER'S MACHINE / CI                       │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                                                             │ │
│  │                     deployer CLI                            │ │
│  │                   (Single Binary)                           │ │
│  │                                                             │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │ │
│  │  │   Parser     │  │  Reconciler  │  │   Executor   │    │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘    │ │
│  │         │                 │                   │            │ │
│  └─────────┼─────────────────┼───────────────────┼────────────┘ │
│            │                 │                   │              │
└────────────┼─────────────────┼───────────────────┼──────────────┘
             │                 │                   │
             ▼                 ▼                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                                                                   │
│                         AWS Cloud                                 │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │      S3      │  │  DynamoDB    │  │   Pulumi Backend     │  │
│  │  (State)     │  │  (Locks)     │  │   (ECS/RDS/etc.)     │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

**Key Point**: The deployer CLI runs on the user's machine or in CI/CD. There is no "deployer service" running in the cloud.

---

## Initial Setup

### Step 1: Install CLI

```bash
# Install deployer CLI
curl -sSL https://deployer.io/install.sh | sh

# Or download binary
wget https://github.com/company/deployer/releases/download/v1.0.0/deployer-linux-amd64
chmod +x deployer-linux-amd64
sudo mv deployer-linux-amd64 /usr/local/bin/deployer

# Verify
deployer version
```

### Step 2: Configure Backend (One-Time Setup)

User provides their own S3 bucket and DynamoDB table:

```bash
# Initialize deployer with backend configuration
deployer init

# Interactive prompts:
? AWS Region: us-east-1
? S3 Bucket for state: company-deployer-state
? DynamoDB Table for locks: company-deployer-locks
? AWS Profile (optional): default

# Saves configuration to ~/.deployer/config.yaml
```

**Configuration File: `~/.deployer/config.yaml`**

```yaml
version: v1

backend:
  type: s3
  region: us-east-1
  bucket: company-deployer-state
  
locks:
  type: dynamodb
  region: us-east-1
  table: company-deployer-locks

aws:
  profile: default
  region: us-east-1
```

### Step 3: Create Infrastructure (One-Time per Organization)

Users need to create the S3 bucket and DynamoDB table once:

```bash
# Option A: Use deployer to create infrastructure
deployer backend create \
  --bucket company-deployer-state \
  --table company-deployer-locks \
  --region us-east-1

# Option B: Use Terraform (provided in repo)
cd infrastructure/terraform
terraform init
terraform apply \
  -var="bucket_name=company-deployer-state" \
  -var="table_name=company-deployer-locks"

# Option C: Create manually in AWS Console
# (See docs/BACKEND_SETUP.md)
```

---

## Stack Management

### Create a Stack

A **stack** is just a directory with YAML files:

```bash
# Create new stack
mkdir -p my-deployment-repo/stacks/user-platform
cd my-deployment-repo/stacks/user-platform

# Initialize stack
deployer stack init

# Creates:
# stacks/user-platform/
# ├── stack.yaml
# ├── infra/
# ├── services/
# └── environments/
```

### Stack Configuration: `stack.yaml`

```yaml
apiVersion: core.deployer.io/v1
kind: Stack

metadata:
  name: user-platform
  description: "User platform services"

spec:
  provider:
    name: aws
    region: us-east-1
  
  # Backend config can be overridden per stack
  # backend:
  #   bucket: custom-bucket
  #   prefix: user-platform/
```

---

## Usage Workflow

### Local Development

```bash
cd ~/work/my-deployment-repo/

# 1. Validate configuration
deployer validate --stack user-platform

# 2. Plan deployment (dry-run)
deployer plan \
  --stack user-platform \
  --environment dev \
  --var VERSION=v1.0.0

# 3. Deploy
deployer apply \
  --stack user-platform \
  --environment dev \
  --var VERSION=v1.0.0

# 4. Check status
deployer status \
  --stack user-platform \
  --environment dev
```

### CI/CD Integration

**.github/workflows/deploy.yml**

```yaml
name: Deploy

on:
  workflow_dispatch:
    inputs:
      stack:
        description: 'Stack name'
        required: true
      environment:
        description: 'Environment'
        required: true
      version:
        description: 'Version to deploy'
        required: true

jobs:
  deploy:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      # Install deployer CLI
      - name: Install Deployer
        run: |
          curl -sSL https://deployer.io/install.sh | sh
          deployer version
      
      # Configure AWS credentials
      - name: Configure AWS
        uses: aws-actions/configure-aws-credentials@v2
        with:
          role-to-assume: arn:aws:iam::ACCOUNT:role/DeployerRole
          aws-region: us-east-1
      
      # Deployer uses default AWS credentials
      # Backend config from repo or ~/.deployer/config.yaml
      
      # Deploy
      - name: Deploy Stack
        run: |
          deployer apply \
            --stack ${{ inputs.stack }} \
            --environment ${{ inputs.environment }} \
            --var VERSION=${{ inputs.version }} \
            --auto-approve
      
      # Verify
      - name: Verify Deployment
        run: |
          deployer status \
            --stack ${{ inputs.stack }} \
            --environment ${{ inputs.environment }}
```

---

## How It Works

### Execution Flow

```
1. User runs: deployer apply --stack user-platform --environment production

2. CLI Process:
   ├── Read ~/.deployer/config.yaml (backend config)
   ├── Scan stacks/user-platform/ directory
   ├── Parse all YAML files
   ├── Apply environment overlays
   ├── Validate schemas
   ├── Build dependency graph
   │
   ├── Connect to DynamoDB
   ├── Acquire lock (atomic write)
   │   └── If locked, wait or fail
   │
   ├── Connect to S3
   ├── Load current state
   │
   ├── Compute diff (current vs desired)
   ├── Generate execution plan
   ├── Display plan to user
   ├── Wait for approval (unless --auto-approve)
   │
   ├── Execute via Pulumi
   │   ├── Create/update AWS resources
   │   ├── Run health checks
   │   └── Capture outputs
   │
   ├── Save new state to S3
   ├── Release lock in DynamoDB
   └── Exit

3. CLI exits (nothing keeps running)
```

### State Storage

**S3 Bucket Structure:**

```
s3://company-deployer-state/
└── stacks/
    └── user-platform/
        ├── production/
        │   ├── state.json
        │   ├── history/
        │   │   ├── 2024-01-15-10-30-00.json
        │   │   └── 2024-01-15-14-20-00.json
        │   └── pulumi/
        │       └── .pulumi/
        │
        ├── staging/
        │   └── state.json
        │
        └── development/
            └── state.json
```

**Lock Entry in DynamoDB:**

```
Table: company-deployer-locks

Item:
{
  "lockKey": "stack:user-platform:env:production",
  "lockId": "550e8400-e29b-41d4-a716-446655440000",
  "lockedBy": "alice@company.com",
  "lockedAt": 1705329600,
  "expiresAt": 1705333200,
  "lastHeartbeat": 1705330800,
  "metadata": {
    "hostname": "alice-laptop",
    "pid": 12345,
    "version": "1.0.0"
  }
}
```

---

## CLI Configuration

### Configuration Hierarchy

Priority (highest to lowest):

1. **Command-line flags**
   ```bash
   deployer apply --backend-bucket my-bucket --stack user-platform
   ```

2. **Environment variables**
   ```bash
   export DEPLOYER_BACKEND_BUCKET=my-bucket
   deployer apply --stack user-platform
   ```

3. **Stack-level config** (`stack.yaml`)
   ```yaml
   spec:
     backend:
       bucket: my-bucket
   ```

4. **User config** (`~/.deployer/config.yaml`)
   ```yaml
   backend:
     bucket: default-bucket
   ```

5. **System defaults**

### Configuration File: `~/.deployer/config.yaml`

```yaml
version: v1

# Backend configuration
backend:
  type: s3
  region: us-east-1
  bucket: company-deployer-state
  prefix: ""  # Optional prefix for all state files
  
# Lock configuration
locks:
  type: dynamodb
  region: us-east-1
  table: company-deployer-locks
  ttl: 3600  # Lock TTL in seconds (default: 1 hour)
  heartbeat: 30  # Heartbeat interval in seconds

# AWS configuration
aws:
  profile: default
  region: us-east-1
  # Assume role if needed
  # assumeRole:
  #   roleArn: arn:aws:iam::123456789012:role/DeployerRole
  #   sessionName: deployer

# Pulumi configuration
pulumi:
  backend: s3  # Use same S3 bucket as deployer state
  # Or use Pulumi Cloud
  # backend: app.pulumi.com
  # accessToken: ${PULUMI_ACCESS_TOKEN}

# CLI preferences
preferences:
  colorOutput: true
  progressBar: true
  autoApprove: false  # Default behavior for --auto-approve
  parallelism: 10     # Max parallel operations

# Logging
logging:
  level: info  # debug | info | warn | error
  file: ~/.deployer/logs/deployer.log
  maxSize: 100  # MB
  maxBackups: 10
  maxAge: 30  # days
```

---

## CLI Commands

### Global Flags

```bash
# All commands support these flags
--verbose, -v          Verbose output
--quiet, -q            Minimal output
--config FILE          Config file path (default: ~/.deployer/config.yaml)
--backend-bucket NAME  Override S3 bucket
--backend-region NAME  Override AWS region
--lock-table NAME      Override DynamoDB table
--no-color             Disable colored output
```

### Core Commands

```bash
# Initialize deployer
deployer init

# Create backend infrastructure
deployer backend create --bucket NAME --table NAME

# Stack operations
deployer stack init
deployer stack list
deployer stack validate

# Deployment
deployer plan     [--stack NAME] [--environment ENV]
deployer apply    [--stack NAME] [--environment ENV] [--var KEY=VALUE]
deployer destroy  [--stack NAME] [--environment ENV]

# Status & Info
deployer status   [--stack NAME] [--environment ENV]
deployer show     [--stack NAME] [--component NAME]
deployer outputs  [--stack NAME] [--environment ENV]
deployer graph    [--stack NAME] [--output FILE]

# Logs & Monitoring
deployer logs     [--component NAME] [--follow] [--since DURATION]
deployer metrics  [--component NAME] [--since DURATION]

# State Management
deployer state list
deployer state show    [--stack NAME] [--environment ENV]
deployer state pull    [--stack NAME] [--environment ENV]
deployer state push    [--stack NAME] [--environment ENV]
deployer state rm      [--stack NAME] [--environment ENV] [--resource ID]

# Lock Management
deployer locks list
deployer locks show    [--stack NAME] [--environment ENV]
deployer unlock        [--stack NAME] [--environment ENV] [--force]

# Drift Detection
deployer drift detect    [--stack NAME] [--environment ENV]
deployer drift remediate [--stack NAME] [--environment ENV]

# Rollback
deployer rollback [--stack NAME] [--environment ENV] [--to-version VERSION]
deployer history  [--stack NAME] [--environment ENV]

# Utilities
deployer validate [--stack NAME]
deployer fmt      [--stack NAME]
deployer version
deployer help
```

---

## Architecture Benefits

### Simple Deployment Model

✅ **No backend service to maintain**
- No servers to run
- No APIs to secure
- No service to scale
- No uptime to monitor

✅ **Standard CLI tool**
- Install once
- Works anywhere (dev machine, CI/CD)
- Familiar workflow (like Terraform/Pulumi)

✅ **User controls infrastructure**
- Users provide S3 bucket
- Users provide DynamoDB table
- Users control costs
- Users control access

✅ **Git-based workflow**
- YAML files in Git
- Version controlled
- Code review process
- Audit trail

### How Teams Use It

**Team 1: Notifications**
```bash
# In their repo
cd ~/work/notification-service/deployment/
deployer apply --stack notification-platform --environment production
```

**Team 2: Payments**
```bash
# In their repo
cd ~/work/payment-service/deployment/
deployer apply --stack payment-platform --environment production
```

Both teams:
- Use same deployer CLI binary
- Use same S3 bucket (different prefixes)
- Use same DynamoDB table (different lock keys)
- Work independently

---

## Multi-Tenant Considerations

### Single Organization

All teams share:
- One S3 bucket: `company-deployer-state`
- One DynamoDB table: `company-deployer-locks`

State isolation via S3 prefixes:
```
s3://company-deployer-state/
├── stacks/user-platform/production/state.json
├── stacks/payment-platform/production/state.json
└── stacks/notification-platform/production/state.json
```

Lock isolation via lock keys:
```
DynamoDB items:
- stack:user-platform:env:production
- stack:payment-platform:env:production
- stack:notification-platform:env:production
```

### Multiple Organizations

Each organization has:
- Their own S3 bucket
- Their own DynamoDB table
- Their own `~/.deployer/config.yaml`

---

## Security Model

### IAM Permissions

**User/CI needs:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DeployerState",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::company-deployer-state",
        "arn:aws:s3:::company-deployer-state/*"
      ]
    },
    {
      "Sid": "DeployerLocks",
      "Effect": "Allow",
      "Action": [
        "dynamodb:PutItem",
        "dynamodb:GetItem",
        "dynamodb:DeleteItem",
        "dynamodb:UpdateItem"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/company-deployer-locks"
    },
    {
      "Sid": "DeployResources",
      "Effect": "Allow",
      "Action": [
        "ecs:*",
        "rds:*",
        "elasticache:*",
        "s3:*",
        "sqs:*"
      ],
      "Resource": "*"
    }
  ]
}
```

### Authentication Methods

**Option 1: AWS Profile (Local)**
```bash
aws configure --profile deployer
export AWS_PROFILE=deployer
deployer apply --stack user-platform
```

**Option 2: IAM Role (CI/CD)**
```yaml
# GitHub Actions
- uses: aws-actions/configure-aws-credentials@v2
  with:
    role-to-assume: arn:aws:iam::ACCOUNT:role/DeployerRole
```

**Option 3: Environment Variables**
```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
deployer apply --stack user-platform
```

---

## Comparison with Similar Tools

### Like Terraform

✅ **Similar:**
- CLI tool
- State in S3
- Lock in DynamoDB
- Declarative YAML/HCL

❌ **Different:**
- Uses Pulumi under the hood
- Higher-level abstractions (MicroService, not ECS primitives)
- Opinionated about structure (Stack → Service → Component)

### Like Pulumi

✅ **Similar:**
- Uses Pulumi for orchestration
- State management
- Resource graph

❌ **Different:**
- YAML-based (not code)
- Purpose-built for application deployment
- Simpler for app teams (no programming needed)

### Like Kubernetes Helm

✅ **Similar:**
- YAML-based
- Templating and overlays
- Package management concept (stacks)

❌ **Different:**
- AWS-focused (not Kubernetes)
- Broader scope (not just containers)
- State managed externally (S3)

---

## Installation

### Binary Installation

```bash
# Linux
curl -sSL https://deployer.io/install.sh | sh

# macOS
brew install deployer

# Windows
choco install deployer

# Docker
docker run -v ~/.deployer:/root/.deployer deployer/cli:latest apply --stack user-platform
```

### Build from Source

```bash
git clone https://github.com/company/deployer.git
cd deployer
make build
sudo mv bin/deployer /usr/local/bin/
```

---

## Summary

The deployer is a **CLI tool**, not a service. Users:

1. Install the `deployer` binary
2. Configure backend (S3 + DynamoDB) once
3. Define stacks in YAML
4. Run `deployer apply` from CI/CD or locally
5. CLI handles everything (parsing, state, locking, deployment)

**No backend service. No servers. Just a CLI tool.** ✅

This is the correct architecture!



