# Architecture Clarification: CLI-First Design

## Critical Correction ⚠️

**The deployer is a CLI tool, not a backend service.**

This document clarifies the correct architecture based on user feedback.

---

## What Deployer Is

✅ **Deployer is a command-line binary** (like `terraform`, `pulumi`, `kubectl`)
- Single executable file
- Runs on user's machine or in CI/CD
- Exits after completing its work
- No persistent process

✅ **Users control the infrastructure**
- Users provide S3 bucket name
- Users provide DynamoDB table name
- Users create these resources once
- Deployer uses them for state and locking

✅ **Git-based workflow**
- YAML files in Git repository
- Version controlled configurations
- Standard PR/review process
- Audit trail via Git history

---

## What Deployer Is NOT

❌ **NOT a backend service**
- No API server running
- No web service to maintain
- No load balancers
- No service-to-service communication

❌ **NOT a SaaS platform**
- No deployer.io cloud service
- No managed infrastructure
- No hosted control plane
- No subscription required

❌ **NOT Kubernetes-based**
- No cluster to manage
- No operator pattern
- No CRDs or controllers
- Direct AWS resource management

---

## Correct Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   Developer's Laptop / CI Runner                  │
│                                                                   │
│  $ deployer apply --stack user-platform --environment production │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                                                           │   │
│  │                  deployer CLI Binary                      │   │
│  │            (Runs once, then exits)                        │   │
│  │                                                           │   │
│  │  1. Parse YAML files from disk                           │   │
│  │  2. Connect to S3/DynamoDB (user-provided)              │   │
│  │  3. Acquire lock                                         │   │
│  │  4. Load state                                           │   │
│  │  5. Compute changes                                      │   │
│  │  6. Execute via Pulumi                                   │   │
│  │  7. Save state                                           │   │
│  │  8. Release lock                                         │   │
│  │  9. Exit                                                 │   │
│  │                                                           │   │
│  └──────────────────────────────────────────────────────────┘   │
│                           │                                       │
└───────────────────────────┼───────────────────────────────────────┘
                            │
                            ▼
              ┌─────────────────────────────┐
              │       AWS Resources         │
              │    (User's AWS Account)     │
              ├─────────────────────────────┤
              │                             │
              │  • S3 Bucket                │
              │    └─ State files           │
              │                             │
              │  • DynamoDB Table           │
              │    └─ Lock entries          │
              │                             │
              │  • ECS, RDS, S3, SQS, etc.  │
              │    └─ Deployed resources    │
              │                             │
              └─────────────────────────────┘
```

---

## User Journey

### One-Time Setup (Per User)

```bash
# 1. Install CLI
curl -sSL https://deployer.io/install.sh | sh

# 2. Configure backend
deployer init
? AWS Region: us-east-1
? S3 Bucket for state: company-deployer-state
? DynamoDB Table for locks: company-deployer-locks
? AWS Profile: default

# Saves to ~/.deployer/config.yaml

# 3. Verify
deployer version
```

### One-Time Setup (Per Organization)

Create the backend infrastructure once:

```bash
# Using deployer
deployer backend create \
  --bucket company-deployer-state \
  --table company-deployer-locks \
  --region us-east-1

# Or using Terraform (provided)
cd infrastructure/terraform
terraform apply
```

**Creates:**
- S3 bucket with versioning
- DynamoDB table with TTL
- IAM role/policies

### Daily Usage

```bash
# Developer workflow
cd ~/work/my-service/deployment/

# Edit YAML files
vim stacks/user-platform/services/my-service/components/api/microservice.yaml

# Deploy
deployer apply --stack user-platform --environment dev

# CLI runs, deploys, exits
```

### CI/CD Usage

```yaml
# .github/workflows/deploy.yml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Deployer
        run: curl -sSL https://deployer.io/install.sh | sh
      
      - name: Configure AWS
        uses: aws-actions/configure-aws-credentials@v2
        with:
          role-to-assume: arn:aws:iam::ACCOUNT:role/DeployerRole
      
      # Backend config from ~/.deployer/config.yaml or repo
      - name: Deploy
        run: |
          deployer apply \
            --stack user-platform \
            --environment production \
            --var VERSION=v1.0.0 \
            --auto-approve
```

---

## Configuration

### User Configuration: `~/.deployer/config.yaml`

```yaml
version: v1

# Backend configuration (user-provided)
backend:
  type: s3
  region: us-east-1
  bucket: company-deployer-state  # User provides this
  
# Lock configuration (user-provided)
locks:
  type: dynamodb
  region: us-east-1
  table: company-deployer-locks    # User provides this
  
# AWS configuration
aws:
  profile: default
  region: us-east-1
```

### Stack Configuration: `stack.yaml`

```yaml
apiVersion: core.deployer.io/v1
kind: Stack

metadata:
  name: user-platform

spec:
  provider:
    name: aws
    region: us-east-1
  
  # Can override backend per stack
  # backend:
  #   bucket: custom-bucket
  #   prefix: user-platform/
```

---

## Key Differences from Original Design

| Aspect | ❌ Original (Incorrect) | ✅ Corrected |
|--------|------------------------|--------------|
| **Architecture** | Backend service | CLI tool |
| **Deployment** | API calls to service | Run CLI binary |
| **State Storage** | Deployer manages | User provides S3 bucket |
| **Locking** | Deployer manages | User provides DynamoDB table |
| **Process Model** | Always running | Run and exit |
| **Installation** | Deploy service | Install binary |
| **Configuration** | Service config | User config file |
| **Scalability** | Service scalability concerns | No concerns (CLI) |
| **Cost** | Service infrastructure | Only S3 + DynamoDB (~$3/mo) |

---

## Advantages of CLI Approach

### Simplicity
- ✅ No backend service to maintain
- ✅ No APIs to secure
- ✅ No uptime concerns
- ✅ No scaling challenges

### User Control
- ✅ Users own the S3 bucket
- ✅ Users own the DynamoDB table
- ✅ Users control costs
- ✅ Users control access policies

### Familiar Workflow
- ✅ Like Terraform: `terraform apply`
- ✅ Like Pulumi: `pulumi up`
- ✅ Like kubectl: `kubectl apply`
- ✅ Standard tool pattern

### CI/CD Integration
- ✅ Easy to install in CI
- ✅ Just another binary
- ✅ No service dependencies
- ✅ Works in any CI/CD system

### Portability
- ✅ Runs anywhere (laptop, CI, bastion)
- ✅ No network dependencies (except AWS)
- ✅ Offline validation possible
- ✅ Air-gapped environments possible

---

## How It Compares

### Like Terraform

```bash
# Terraform workflow
terraform init
terraform plan
terraform apply

# Deployer workflow
deployer init
deployer plan --stack user-platform
deployer apply --stack user-platform
```

**Similar:**
- CLI tool
- State in S3
- Locks in DynamoDB
- Declarative configuration

**Different:**
- YAML vs HCL
- Higher-level abstractions
- Opinionated structure
- Uses Pulumi internally

### Like Pulumi

```bash
# Pulumi workflow
pulumi login s3://my-bucket
pulumi up

# Deployer workflow
deployer init  # Configure S3 bucket
deployer apply --stack user-platform
```

**Similar:**
- Uses Pulumi for orchestration
- State management
- Resource graph

**Different:**
- YAML-based (not code)
- No programming needed
- Purpose-built for app deployment
- Simpler for app teams

---

## Implementation Impact

### What Doesn't Change

✅ **Core functionality**
- Parsing YAML files
- Building dependency graphs
- State management concepts
- Lock management concepts
- Pulumi integration
- Component translators

✅ **User experience**
- YAML definitions
- Stack/Service/Component model
- Deployment workflows
- CLI commands

### What Changes

🔄 **Execution model**
- No API server
- Direct execution
- Process starts and exits
- No persistent workers

🔄 **Configuration**
- User provides backend config
- Config file on user's machine
- No server-side config

🔄 **Deployment**
- No service to deploy
- Just distribute binary
- Update via package managers

🔄 **Documentation**
- Emphasize CLI nature
- Installation instructions
- Backend setup guide
- No service maintenance docs

---

## Updated Documentation

The following documents have been updated to reflect the CLI architecture:

✅ **Created:**
- `CLI_ARCHITECTURE.md` - Complete CLI design

✅ **Updated:**
- `ARCHITECTURE.md` - Added CLI clarification
- `README.md` - Emphasized CLI nature
- `USER_WORKFLOWS.md` - Added CLI setup steps
- `INDEX.md` - Added CLI_ARCHITECTURE.md link

✅ **Still Valid:**
- `STATE_AND_LOCKING.md` - Concepts unchanged
- `E2E_IMPLEMENTATION_AND_TESTING_PLAN.md` - Implementation still valid
- `IMPLEMENTATION_PLAN.md` - Milestones still valid
- All component designs - Unchanged

---

## Summary

**Deployer is a CLI tool (like Terraform or Pulumi), not a backend service.**

**Users:**
1. Install the `deployer` binary
2. Provide S3 bucket and DynamoDB table names
3. Define stacks in YAML files
4. Run `deployer apply` from anywhere
5. CLI handles everything and exits

**No backend. No service. Just a CLI tool.** ✅

This is the correct architecture going forward.

---

**Last Updated**: November 26, 2024
**Status**: Clarified and Corrected



