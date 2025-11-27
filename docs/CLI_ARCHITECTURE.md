# Panka CLI Architecture

The panka is a **command-line tool** (similar to Pulumi or Terraform) that teams use to deploy their applications. There is no backend service - it's just a CLI binary that reads YAML, manages state in S3, uses DynamoDB for locking, and orchestrates deployments via Pulumi.

---

## Core Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                   │
│                         USER'S MACHINE / CI                       │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                                                             │ │
│  │                     panka CLI                            │ │
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

**Key Point**: The panka CLI runs on the user's machine or in CI/CD. There is no "panka service" running in the cloud.

---

## Initial Setup

### Step 1: Install CLI

```bash
# Install panka CLI
curl -sSL https://panka.io/install.sh | sh

# Or download binary
wget https://github.com/company/panka/releases/download/v1.0.0/panka-linux-amd64
chmod +x panka-linux-amd64
sudo mv panka-linux-amd64 /usr/local/bin/panka

# Verify
panka version
```

### Step 2: Configure Backend (One-Time Setup)

User provides their own S3 bucket and DynamoDB table:

```bash
# Initialize panka with backend configuration
panka init

# Interactive prompts:
? AWS Region: us-east-1
? S3 Bucket for state: company-panka-state
? DynamoDB Table for locks: company-panka-locks
? AWS Profile (optional): default

# Saves configuration to ~/.panka/config.yaml
```

**Configuration File: `~/.panka/config.yaml`**

```yaml
version: v1

backend:
  type: s3
  region: us-east-1
  bucket: company-panka-state
  
locks:
  type: dynamodb
  region: us-east-1
  table: company-panka-locks

aws:
  profile: default
  region: us-east-1
```

### Step 3: Create Infrastructure (One-Time per Organization)

Users need to create the S3 bucket and DynamoDB table once:

```bash
# Option A: Use panka to create infrastructure
panka backend create \
  --bucket company-panka-state \
  --table company-panka-locks \
  --region us-east-1

# Option B: Use Terraform (provided in repo)
cd infrastructure/terraform
terraform init
terraform apply \
  -var="bucket_name=company-panka-state" \
  -var="table_name=company-panka-locks"

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
panka stack init

# Creates:
# stacks/user-platform/
# ├── stack.yaml
# ├── infra/
# ├── services/
# └── environments/
```

### Stack Configuration: `stack.yaml`

```yaml
apiVersion: core.panka.io/v1
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
panka validate --stack user-platform

# 2. Plan deployment (dry-run)
panka plan \
  --stack user-platform \
  --environment dev \
  --var VERSION=v1.0.0

# 3. Deploy
panka apply \
  --stack user-platform \
  --environment dev \
  --var VERSION=v1.0.0

# 4. Check status
panka status \
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
      
      # Install panka CLI
      - name: Install Panka
        run: |
          curl -sSL https://panka.io/install.sh | sh
          panka version
      
      # Configure AWS credentials
      - name: Configure AWS
        uses: aws-actions/configure-aws-credentials@v2
        with:
          role-to-assume: arn:aws:iam::ACCOUNT:role/PankaRole
          aws-region: us-east-1
      
      # Panka uses default AWS credentials
      # Backend config from repo or ~/.panka/config.yaml
      
      # Deploy
      - name: Deploy Stack
        run: |
          panka apply \
            --stack ${{ inputs.stack }} \
            --environment ${{ inputs.environment }} \
            --var VERSION=${{ inputs.version }} \
            --auto-approve
      
      # Verify
      - name: Verify Deployment
        run: |
          panka status \
            --stack ${{ inputs.stack }} \
            --environment ${{ inputs.environment }}
```

---

## How It Works

### Execution Flow

```
1. User runs: panka apply --stack user-platform --environment production

2. CLI Process:
   ├── Read ~/.panka/config.yaml (backend config)
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
s3://company-panka-state/
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
Table: company-panka-locks

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
   panka apply --backend-bucket my-bucket --stack user-platform
   ```

2. **Environment variables**
   ```bash
   export PANKA_BACKEND_BUCKET=my-bucket
   panka apply --stack user-platform
   ```

3. **Stack-level config** (`stack.yaml`)
   ```yaml
   spec:
     backend:
       bucket: my-bucket
   ```

4. **User config** (`~/.panka/config.yaml`)
   ```yaml
   backend:
     bucket: default-bucket
   ```

5. **System defaults**

### Configuration File: `~/.panka/config.yaml`

```yaml
version: v1

# Backend configuration
backend:
  type: s3
  region: us-east-1
  bucket: company-panka-state
  prefix: ""  # Optional prefix for all state files
  
# Lock configuration
locks:
  type: dynamodb
  region: us-east-1
  table: company-panka-locks
  ttl: 3600  # Lock TTL in seconds (default: 1 hour)
  heartbeat: 30  # Heartbeat interval in seconds

# AWS configuration
aws:
  profile: default
  region: us-east-1
  # Assume role if needed
  # assumeRole:
  #   roleArn: arn:aws:iam::123456789012:role/PankaRole
  #   sessionName: panka

# Pulumi configuration
pulumi:
  backend: s3  # Use same S3 bucket as panka state
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
  file: ~/.panka/logs/panka.log
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
--config FILE          Config file path (default: ~/.panka/config.yaml)
--backend-bucket NAME  Override S3 bucket
--backend-region NAME  Override AWS region
--lock-table NAME      Override DynamoDB table
--no-color             Disable colored output
```

### Core Commands

```bash
# Initialize panka
panka init

# Create backend infrastructure
panka backend create --bucket NAME --table NAME

# Stack operations
panka stack init
panka stack list
panka stack validate

# Deployment
panka plan     [--stack NAME] [--environment ENV]
panka apply    [--stack NAME] [--environment ENV] [--var KEY=VALUE]
panka destroy  [--stack NAME] [--environment ENV]

# Status & Info
panka status   [--stack NAME] [--environment ENV]
panka show     [--stack NAME] [--component NAME]
panka outputs  [--stack NAME] [--environment ENV]
panka graph    [--stack NAME] [--output FILE]

# Logs & Monitoring
panka logs     [--component NAME] [--follow] [--since DURATION]
panka metrics  [--component NAME] [--since DURATION]

# State Management
panka state list
panka state show    [--stack NAME] [--environment ENV]
panka state pull    [--stack NAME] [--environment ENV]
panka state push    [--stack NAME] [--environment ENV]
panka state rm      [--stack NAME] [--environment ENV] [--resource ID]

# Lock Management
panka locks list
panka locks show    [--stack NAME] [--environment ENV]
panka unlock        [--stack NAME] [--environment ENV] [--force]

# Drift Detection
panka drift detect    [--stack NAME] [--environment ENV]
panka drift remediate [--stack NAME] [--environment ENV]

# Rollback
panka rollback [--stack NAME] [--environment ENV] [--to-version VERSION]
panka history  [--stack NAME] [--environment ENV]

# Utilities
panka validate [--stack NAME]
panka fmt      [--stack NAME]
panka version
panka help
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
panka apply --stack notification-platform --environment production
```

**Team 2: Payments**
```bash
# In their repo
cd ~/work/payment-service/deployment/
panka apply --stack payment-platform --environment production
```

Both teams:
- Use same panka CLI binary
- Use same S3 bucket (different prefixes)
- Use same DynamoDB table (different lock keys)
- Work independently

---

## Multi-Tenant Considerations

### Single Organization

All teams share:
- One S3 bucket: `company-panka-state`
- One DynamoDB table: `company-panka-locks`

State isolation via S3 prefixes:
```
s3://company-panka-state/
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
- Their own `~/.panka/config.yaml`

---

## Security Model

### IAM Permissions

**User/CI needs:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PankaState",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::company-panka-state",
        "arn:aws:s3:::company-panka-state/*"
      ]
    },
    {
      "Sid": "PankaLocks",
      "Effect": "Allow",
      "Action": [
        "dynamodb:PutItem",
        "dynamodb:GetItem",
        "dynamodb:DeleteItem",
        "dynamodb:UpdateItem"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/company-panka-locks"
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
aws configure --profile panka
export AWS_PROFILE=panka
panka apply --stack user-platform
```

**Option 2: IAM Role (CI/CD)**
```yaml
# GitHub Actions
- uses: aws-actions/configure-aws-credentials@v2
  with:
    role-to-assume: arn:aws:iam::ACCOUNT:role/PankaRole
```

**Option 3: Environment Variables**
```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
panka apply --stack user-platform
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
curl -sSL https://panka.io/install.sh | sh

# macOS
brew install panka

# Windows
choco install panka

# Docker
docker run -v ~/.panka:/root/.panka panka/cli:latest apply --stack user-platform
```

### Build from Source

```bash
git clone https://github.com/company/panka.git
cd panka
make build
sudo mv bin/panka /usr/local/bin/
```

---

## Summary

The panka is a **CLI tool**, not a service. Users:

1. Install the `panka` binary
2. Configure backend (S3 + DynamoDB) once
3. Define stacks in YAML
4. Run `panka apply` from CI/CD or locally
5. CLI handles everything (parsing, state, locking, deployment)

**No backend service. No servers. Just a CLI tool.** ✅

This is the correct architecture!



