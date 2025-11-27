# Deployer - Complete Overview

Complete guide to the multi-tenant AWS deployment CLI tool.

---

## What is Deployer?

Deployer is a **CLI tool** (like Terraform/Pulumi) that enables development teams to deploy and manage AWS applications using simple YAML files, with **multi-tenant** capabilities for organizational scale.

### Key Characteristics

- **CLI Tool** - No backend service to run or maintain
- **Multi-Tenant** - Isolated environments for each team
- **YAML-Based** - Declarative configuration
- **State Management** - S3 with versioning
- **Distributed Locking** - DynamoDB for concurrency control
- **Pulumi Backend** - Extensible to Terraform

---

## Architecture

### Two-Mode CLI

```
┌──────────────────────────────────────────────────────────────┐
│                      DEPLOYER CLI                             │
│                                                               │
│  Mode 1: ADMIN MODE                                          │
│  • Platform team logs in with admin credentials              │
│  • Creates and manages tenants                               │
│  • Monitors all activity                                     │
│  • Commands: tenant init, list, show, rotate, etc.          │
│                                                               │
│  Mode 2: TENANT MODE                                         │
│  • Dev teams log in with tenant credentials                  │
│  • Deploy their stacks                                       │
│  • Isolated state and locks                                  │
│  • Commands: apply, status, logs, etc.                      │
│                                                               │
└──────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────────┐
│                AWS: SHARED INFRASTRUCTURE                     │
│                                                               │
│  S3: company-deployer-state                                  │
│  ├── tenants.yaml              ← Registry of all tenants    │
│  └── tenants/                                                │
│      ├── notifications-team/   ← Tenant 1 (isolated)        │
│      ├── payments-team/        ← Tenant 2 (isolated)        │
│      └── analytics-team/       ← Tenant 3 (isolated)        │
│                                                               │
│  DynamoDB: company-deployer-locks                            │
│  ├── tenant:notifications-team:...  ← Tenant 1 locks        │
│  ├── tenant:payments-team:...       ← Tenant 2 locks        │
│  └── tenant:analytics-team:...      ← Tenant 3 locks        │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### Multi-Tenancy Features

#### 1. Admin Mode
- Login: `deployer admin login`
- Create tenants with isolated namespaces
- Generate secure credentials (e.g., `ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG`)
- Manage tenant lifecycle (suspend, activate, delete)
- Rotate credentials
- Monitor all tenant activity
- Track costs per tenant

#### 2. Tenant Mode
- Login: `deployer login` (with tenant credentials)
- Deploy stacks within tenant namespace
- Completely isolated from other tenants
- Can view own tenant details and usage
- Cannot access admin functions

#### 3. State Isolation
```
S3 Structure per Tenant:

tenants/<tenant-name>/
├── tenant.yaml                     ← Tenant config
└── v1/                             ← Version namespace
    └── stacks/
        └── <stack-name>/
            ├── production/
            │   ├── state.json
            │   └── history/
            ├── staging/
            └── development/
```

#### 4. Lock Isolation
```
DynamoDB Lock Keys:

tenant:<tenant-name>:stack:<stack-name>:env:<environment>

Examples:
tenant:notifications-team:stack:notification-platform:env:production
tenant:payments-team:stack:payment-platform:env:staging
```

#### 5. Credential Management
- Format: `<prefix>_<32-random-chars>`
- Storage: Bcrypt hash in `tenants.yaml`
- Never stored in plain text
- Admin can rotate anytime
- Teams re-authenticate after rotation

---

## The Complete Workflow

### Platform Team (One-Time Setup)

```bash
# 1. Deploy AWS infrastructure
cd deployer/infrastructure/terraform
terraform apply \
  -var="bucket_name=company-deployer-state" \
  -var="table_name=company-deployer-locks"
# Creates S3 bucket + DynamoDB table + Admin credentials

# 2. Install CLI
curl -sSL https://deployer.io/install.sh | sh

# 3. Login as admin
deployer admin login
? S3 Bucket: company-deployer-state
? Region: us-east-1
? Admin Password: ••••••••••••
✓ Logged in as Administrator

# 4. Create tenants
deployer tenant init
? Tenant Name: notifications-team
? Monthly cost limit: 5000
✓ Tenant created
  Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG

# 5. Share credentials with teams
# (via 1Password, AWS Secrets Manager, secure Slack DM, etc.)
```

### Development Teams (One-Time Per Team)

```bash
# 1. Install CLI
curl -sSL https://deployer.io/install.sh | sh

# 2. Login with tenant credentials
deployer login
? Tenant: notifications-team
? Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
? Bucket: company-deployer-state
? Region: us-east-1
✓ Logged in as: notifications-team

# 3. Clone deployment repo
git clone git@github.com:company/deployment-repo.git
cd deployment-repo

# 4. Define stack in YAML
# (Create stack.yaml, service.yaml, component YAMLs)

# 5. Deploy
deployer apply --stack notification-platform --environment dev --var VERSION=v1.0.0
✓ Deployment successful! (8m 35s)
```

### Daily Operations

```bash
# Deploy new version
deployer apply --stack my-stack --var VERSION=v1.0.1

# Check status
deployer status --stack my-stack --environment production

# View logs
deployer logs --component my-service/api --follow

# Rollback if issues
deployer rollback --stack my-stack --environment production

# View tenant details
deployer tenant details

# View usage
deployer tenant usage
```

---

## YAML Structure

### Stack Definition

```yaml
apiVersion: core.deployer.io/v1
kind: Stack

metadata:
  name: notification-platform
  description: "Email and SMS notifications"
  labels:
    team: notifications

spec:
  provider:
    name: aws
    region: us-east-1
  
  infrastructure:
    defaults: ./infra/defaults.yaml
    networking: ./infra/networking.yaml
    security: ./infra/security.yaml
```

### Service Definition

```yaml
apiVersion: core.deployer.io/v1
kind: Service

metadata:
  name: email-service
  stack: notification-platform

spec:
  infrastructure:
    defaults: ./infra/defaults.yaml
```

### Component Definitions

**Application Config** (`microservice.yaml`):
```yaml
apiVersion: components.deployer.io/v1
kind: MicroService

metadata:
  name: api
  service: email-service

spec:
  image:
    repository: 123456789012.dkr.ecr.us-east-1.amazonaws.com/email-api
    tag: "${VERSION}"
  
  ports:
    - name: http
      port: 8080
  
  environment:
    - name: DATABASE_HOST
      valueFrom:
        component: email-service/database
        output: endpoint
  
  secrets:
    - name: DB_PASSWORD
      secretRef: /stacks/notification-platform/email-service/db-password
  
  dependsOn:
    - email-service/database
```

**Infrastructure Config** (`infra.yaml`):
```yaml
apiVersion: infra.deployer.io/v1
kind: ComponentInfra

metadata:
  name: api
  service: email-service

spec:
  resources:
    cpu: 512
    memory: 1024
  
  scaling:
    replicas: 3
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 10
  
  networking:
    loadBalancer:
      enabled: true
      type: application
```

**Database** (`rds.yaml`):
```yaml
apiVersion: components.deployer.io/v1
kind: RDS

metadata:
  name: database
  service: email-service

spec:
  engine:
    type: postgres
    version: "15.4"
  
  instance:
    class: db.t3.small
    storage:
      type: gp3
      allocatedGB: 20
```

### Repository Structure

```
deployment-repo/
├── stacks/
│   └── notification-platform/
│       ├── stack.yaml
│       ├── infra/
│       │   ├── defaults.yaml
│       │   ├── networking.yaml
│       │   └── security.yaml
│       ├── services/
│       │   └── email-service/
│       │       ├── service.yaml
│       │       └── components/
│       │           ├── api/
│       │           │   ├── microservice.yaml
│       │           │   ├── infra.yaml
│       │           │   └── configs/
│       │           │       └── app.yaml
│       │           ├── database/
│       │           │   └── rds.yaml
│       │           └── queue/
│       │               └── sqs.yaml
│       └── environments/
│           ├── production/
│           ├── staging/
│           └── development/
```

---

## CLI Commands

### Admin Commands

```bash
# Authentication
deployer admin login                    # Login as admin
deployer admin logout                   # Logout

# Tenant Management
deployer tenant init                    # Create new tenant
deployer tenant list                    # List all tenants
deployer tenant show <tenant>           # Show tenant details
deployer tenant stats                   # Tenant statistics

# Credential Management
deployer tenant rotate <tenant>         # Rotate credentials

# Lifecycle
deployer tenant suspend <tenant>        # Suspend tenant
deployer tenant activate <tenant>       # Activate tenant
deployer tenant delete <tenant>         # Delete tenant

# Monitoring
deployer admin monitor                  # Real-time activity
deployer admin costs                    # Cost analysis
```

### Tenant Commands

```bash
# Authentication
deployer login                          # Login as tenant
deployer logout                         # Logout

# Tenant Operations
deployer tenant details                 # View tenant details
deployer tenant usage                   # Usage statistics

# Stack Operations
deployer stack init                     # Create stack
deployer validate --stack <name>        # Validate config
deployer plan --stack <name> --env <env> --var VERSION=<ver>
deployer apply --stack <name> --env <env> --var VERSION=<ver>
deployer status --stack <name> --env <env>
deployer logs --component <name> --follow
deployer rollback --stack <name> --env <env>
deployer destroy --stack <name> --env <env>

# History and State
deployer history --stack <name>         # Deployment history
deployer drift --stack <name>           # Detect drift
deployer outputs --stack <name>         # View outputs
```

---

## Benefits

### For Platform Team

✅ **Easy Setup**: One `terraform apply`, done
✅ **No Maintenance**: No backend service to run
✅ **Tenant Management**: Create tenants in seconds
✅ **Cost Tracking**: Per-tenant cost visibility
✅ **Monitoring**: Real-time activity dashboard
✅ **Security**: Isolated state, credential rotation
✅ **Scalability**: Unlimited tenants, no infrastructure changes
✅ **Low Cost**: ~$3/month + usage

### For Development Teams

✅ **Simple Onboarding**: Login once, use forever
✅ **YAML-Based**: No Terraform/Pulumi coding
✅ **Isolated**: Can't see or affect other teams
✅ **Self-Service**: Deploy anytime, no approvals
✅ **Fast**: 5-minute deployments
✅ **Safe**: Automatic rollback on failures
✅ **CI/CD Friendly**: Run in GitHub Actions
✅ **Consistent**: Same process for all environments

### For Organization

✅ **Standardization**: All teams use same tool
✅ **Visibility**: Platform sees all tenants
✅ **Cost Control**: Limits per tenant
✅ **Compliance**: Audit trail in Git + S3
✅ **Faster Delivery**: 10x more deployments
✅ **Lower Risk**: Automatic rollback
✅ **Better Reliability**: Consistent deployments

---

## Documentation

### Quick Start (Read First!)

1. **[MULTI_TENANT_QUICKSTART.md](MULTI_TENANT_QUICKSTART.md)** - Multi-tenant setup guide
2. **[QUICKSTART.md](QUICKSTART.md)** - 5-minute overview
3. **[HOW_TEAMS_USE_DEPLOYER.md](HOW_TEAMS_USE_DEPLOYER.md)** - Visual walkthrough
4. **[GETTING_STARTED_GUIDE.md](docs/GETTING_STARTED_GUIDE.md)** - Complete step-by-step

### For Platform Administrators

1. **[MULTI_TENANCY.md](docs/MULTI_TENANCY.md)** - Multi-tenant architecture
2. **[PLATFORM_ADMIN_GUIDE.md](docs/PLATFORM_ADMIN_GUIDE.md)** - Admin operations

### For Developers

1. **[USER_WORKFLOWS.md](docs/USER_WORKFLOWS.md)** - Common workflows
2. **[END_USER_SUMMARY.md](docs/END_USER_SUMMARY.md)** - Quick reference
3. **[SUMMARY_FOR_TEAMS.md](SUMMARY_FOR_TEAMS.md)** - Team summary

### Architecture & Implementation

1. **[CLI_ARCHITECTURE.md](docs/CLI_ARCHITECTURE.md)** - CLI design
2. **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System architecture
3. **[STATE_AND_LOCKING.md](docs/STATE_AND_LOCKING.md)** - State & locking
4. **[E2E_IMPLEMENTATION_AND_TESTING_PLAN.md](docs/E2E_IMPLEMENTATION_AND_TESTING_PLAN.md)** - Implementation plan

### Navigation

1. **[INDEX.md](INDEX.md)** - Complete documentation index
2. **[README.md](README.md)** - Project overview

---

## Cost Estimate

### AWS Infrastructure (Monthly)

```
S3 Bucket:
- Storage: $0.023/GB × 10 GB = $0.23
- Requests: $0.005/1000 × 10K = $0.05
Total S3: ~$0.30/month

DynamoDB:
- On-demand: $1.25/million writes
- Locks: ~1000/day = 30K/month
- Cost: $1.25 × 0.03 = $0.04
Total DynamoDB: ~$0.05/month

Secrets Manager (Admin credentials):
- $0.40/secret/month = $0.40

───────────────────────────────
Base Cost: ~$1/month

With backups, logging, monitoring: ~$3/month
```

### Resource Costs (Per Tenant)

Application resources (ECS, RDS, etc.) billed normally to your AWS account. Deployer just manages them.

Example tenant:
- ECS Fargate: $145/month
- RDS PostgreSQL: $128/month
- ElastiCache: $42/month
- Total: ~$315/month per stack

---

## Comparison

### vs. Terraform

| Feature | Deployer | Terraform |
|---------|----------|-----------|
| Backend | CLI tool | CLI tool |
| Config | YAML | HCL |
| State | S3 | S3 |
| Locking | DynamoDB | DynamoDB |
| Multi-tenant | Built-in | Manual |
| Application focus | Yes | Infrastructure |
| Learning curve | Low | Medium |

**Use Deployer when:**
- You want multi-tenancy out of the box
- You prefer YAML over HCL
- You want application-focused abstractions

**Use Terraform when:**
- You need maximum flexibility
- You're managing infrastructure, not applications
- You have existing Terraform code

### vs. Pulumi

| Feature | Deployer | Pulumi |
|---------|----------|--------|
| Backend | CLI tool | CLI tool |
| Config | YAML | Code (Go/Python/TS) |
| State | S3 | Pulumi Cloud/S3 |
| Multi-tenant | Built-in | Manual |
| Application focus | Yes | Yes |
| Learning curve | Low | Medium-High |

**Use Deployer when:**
- You want declarative YAML
- You want multi-tenancy built-in
- You want simplicity over flexibility

**Use Pulumi when:**
- You want to code your infrastructure
- You need complex logic
- You want Pulumi's ecosystem

### vs. AWS CDK

| Feature | Deployer | AWS CDK |
|---------|----------|---------|
| Backend | CLI tool | CLI + CloudFormation |
| Config | YAML | Code (TypeScript/Python) |
| Multi-tenant | Built-in | Manual |
| Vendor | AWS-agnostic design | AWS-only |

**Deployer uses Pulumi under the hood, so it gets:**
- Fast deployments (parallel operations)
- Rich state management
- Extensibility

---

## Next Steps

### For Platform Team

1. Read [MULTI_TENANCY.md](docs/MULTI_TENANCY.md)
2. Read [PLATFORM_ADMIN_GUIDE.md](docs/PLATFORM_ADMIN_GUIDE.md)
3. Deploy infrastructure: `terraform apply`
4. Create first tenant: `deployer tenant init`
5. Share credentials with a team
6. Monitor: `deployer admin monitor`

### For Development Teams

1. Read [MULTI_TENANT_QUICKSTART.md](MULTI_TENANT_QUICKSTART.md)
2. Read [GETTING_STARTED_GUIDE.md](docs/GETTING_STARTED_GUIDE.md)
3. Install CLI: `curl -sSL deployer.io/install.sh | sh`
4. Login: `deployer login`
5. Define your stack in YAML
6. Deploy: `deployer apply`

### For Implementers

1. Read [CLI_ARCHITECTURE.md](docs/CLI_ARCHITECTURE.md)
2. Read [ARCHITECTURE.md](docs/ARCHITECTURE.md)
3. Read [E2E_IMPLEMENTATION_AND_TESTING_PLAN.md](docs/E2E_IMPLEMENTATION_AND_TESTING_PLAN.md)
4. Start coding!

---

## Summary

**Deployer** is a multi-tenant CLI tool for deploying AWS applications using YAML.

**One Platform Team** sets up:
- S3 bucket
- DynamoDB table
- Creates tenants

**Multiple Development Teams** use:
- Same CLI tool
- Tenant credentials
- YAML configs
- Git workflow

**Result**:
- ✅ Complete isolation per team
- ✅ Centralized management
- ✅ Simple for everyone
- ✅ Scalable to unlimited teams
- ✅ Low cost (~$3/month base)

**Get Started**: [MULTI_TENANT_QUICKSTART.md](MULTI_TENANT_QUICKSTART.md)

---

**Built with ❤️ for modern DevOps teams**

