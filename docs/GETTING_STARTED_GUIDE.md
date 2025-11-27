# Getting Started Guide for Development Teams

This guide walks you through onboarding to the panka system, from initial setup to deploying your first service.

---

## Overview: Two Phases

1. **One-Time Setup** (Platform team does once for the organization)
2. **Team Onboarding** (Each development team does once)
3. **Daily Usage** (Ongoing)

---

## Phase 1: Platform Team Setup (One-Time)

**Who**: Platform/DevOps team
**When**: Once per organization
**Duration**: ~30 minutes

### Step 1: Create AWS Infrastructure

The platform team creates the shared backend infrastructure:

```bash
# Clone panka repository
git clone https://github.com/company/panka.git
cd panka/infrastructure/terraform

# Initialize Terraform
terraform init

# Create S3 bucket and DynamoDB table
terraform apply \
  -var="bucket_name=company-panka-state" \
  -var="table_name=company-panka-locks" \
  -var="region=us-east-1" \
  -var="aws_account_id=123456789012"

# Output:
# ✓ Created S3 bucket: company-panka-state
# ✓ Created DynamoDB table: company-panka-locks
# ✓ Created IAM role: PankaExecutionRole
```

**What this creates:**
- S3 bucket with versioning enabled (for state storage)
- DynamoDB table with TTL enabled (for locking)
- IAM role with required permissions

### Step 2: Create Deployment Repository

Create a central repository for all deployment configurations:

```bash
# Create deployment repository
mkdir -p deployment-repo
cd deployment-repo

# Initialize Git
git init

# Create basic structure
mkdir -p stacks
mkdir -p shared
mkdir -p docs

# Create README
cat > README.md << 'EOF'
# Deployment Repository

This repository contains deployment configurations for all services.

## Backend Configuration

- S3 Bucket: `company-panka-state`
- DynamoDB Table: `company-panka-locks`
- Region: `us-east-1`

## Getting Started

See [Getting Started Guide](docs/GETTING_STARTED.md)
EOF

# Commit and push
git add .
git commit -m "Initial deployment repository"
git remote add origin git@github.com:company/deployment-repo.git
git push -u origin main
```

### Step 3: Document Configuration

Create a configuration guide for teams:

```bash
cat > docs/BACKEND_CONFIG.md << 'EOF'
# Backend Configuration

All teams use the shared panka backend:

## Configuration

Add this to your `~/.panka/config.yaml`:

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

## AWS Permissions

Request access to the `PankaUsers` IAM group for AWS permissions.
EOF
```

### Step 4: Share Configuration

Send to all development teams:

```
📧 Email Template:

Subject: Panka is Ready - Deploy Your Services

Hi Teams,

We've set up the panka system for managing AWS deployments.

Backend Configuration:
- S3 Bucket: company-panka-state
- DynamoDB Table: company-panka-locks
- Region: us-east-1

Getting Started:
1. Install CLI: curl -sSL https://panka.io/install.sh | sh
2. Configure backend: panka init
3. See full guide: https://github.com/company/deployment-repo/docs/GETTING_STARTED.md

Repository:
- Deployment configs: https://github.com/company/deployment-repo

Questions? #panka-help on Slack

- Platform Team
```

---

## Phase 2: Development Team Onboarding

**Who**: Each development team (one-time per team)
**When**: When team wants to start deploying
**Duration**: ~1 hour

Let's follow the **Notifications Team** as they onboard:

### Step 1: Install Panka CLI

Each team member installs the CLI:

```bash
# Install panka
curl -sSL https://panka.io/install.sh | sh

# Verify installation
panka version
# Output: panka version 1.0.0
```

**Alternative installation methods:**

```bash
# macOS with Homebrew
brew install panka

# Download binary directly
wget https://github.com/company/panka/releases/download/v1.0.0/panka-linux-amd64
chmod +x panka-linux-amd64
sudo mv panka-linux-amd64 /usr/local/bin/panka

# Build from source
git clone https://github.com/company/panka.git
cd panka
make build
sudo mv bin/panka /usr/local/bin/
```

### Step 2: Login (Multi-Tenant Setup)

**If your organization uses multi-tenant mode:**

```bash
# Login with tenant credentials
panka login

# Prompts:
? Tenant Name: notifications-team
? Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
? S3 Bucket: company-panka-state
? Region: us-east-1

Authenticating...
✓ Logged in as: notifications-team

Session saved to ~/.panka/session
```

**What the platform team provides:**
- Tenant name (e.g., `notifications-team`)
- Tenant secret (e.g., `ntfy_...`)
- S3 bucket name
- AWS region

**If your organization uses single-tenant mode (legacy):**

```bash
# Configure backend directly
panka init

# Prompts:
? AWS Region: us-east-1
? S3 Bucket for state: company-panka-state
? DynamoDB Table for locks: company-panka-locks
? AWS Profile (leave blank for default): 
✓ Configuration saved to ~/.panka/config.yaml
```

> **Note**: Multi-tenant mode is recommended for organizations with multiple teams. It provides better isolation, security, and cost tracking. See [MULTI_TENANCY.md](MULTI_TENANCY.md) for details.

### Step 3: Configure AWS Credentials

Set up AWS access:

```bash
# Configure AWS CLI
aws configure

# Enter:
AWS Access Key ID: AKIA...
AWS Secret Access Key: ...
Default region: us-east-1
Default output format: json

# Or use AWS SSO
aws sso login --profile panka

# Or use environment variables
export AWS_PROFILE=panka
```

**Verify access:**

```bash
# Test S3 access
aws s3 ls s3://company-panka-state/

# Test DynamoDB access
aws dynamodb describe-table --table-name company-panka-locks

# Both should succeed
```

### Step 4: Clone Deployment Repository

```bash
cd ~/work/

# Clone the deployment repository
git clone git@github.com:company/deployment-repo.git
cd deployment-repo

# Verify structure
ls -la
# Output:
# stacks/
# shared/
# docs/
# README.md
```

### Step 5: Create Your Stack

The Notifications team creates their stack:

```bash
cd deployment-repo

# Create stack directory
mkdir -p stacks/notification-platform
cd stacks/notification-platform

# Initialize stack
panka stack init

# This creates:
# stacks/notification-platform/
# ├── stack.yaml
# ├── infra/
# │   ├── defaults.yaml
# │   ├── networking.yaml
# │   └── security.yaml
# ├── services/
# └── environments/
#     ├── production/
#     ├── staging/
#     └── development/
```

**Edit `stack.yaml`:**

```yaml
apiVersion: core.panka.io/v1
kind: Stack

metadata:
  name: notification-platform
  description: "Email and SMS notification services"
  
  labels:
    team: notifications
    cost-center: engineering
  
  annotations:
    owner: "notifications-team@company.com"
    slack: "#notifications-team"
    repository: "github.com/company/notification-service"

spec:
  provider:
    name: aws
    region: us-east-1
    
  # Backend uses ~/.panka/config.yaml by default
  # Can override per stack if needed:
  # backend:
  #   bucket: custom-bucket
  #   prefix: notification-platform/
```

### Step 6: Define Your First Service

Create a simple service to start:

```bash
# Create service structure
mkdir -p services/email-service/components/{api,database,queue}

# Create service definition
cat > services/email-service/service.yaml << 'EOF'
apiVersion: core.panka.io/v1
kind: Service

metadata:
  name: email-service
  stack: notification-platform
  description: "Email notification service"
  
  labels:
    team: notifications
    tier: backend

spec:
  infrastructure:
    defaults: ./infra/defaults.yaml
EOF
```

**Create API component:**

```bash
cat > services/email-service/components/api/microservice.yaml << 'EOF'
apiVersion: components.panka.io/v1
kind: MicroService

metadata:
  name: api
  service: email-service
  stack: notification-platform
  description: "Email API server"

spec:
  # Container image (from your ECR)
  image:
    repository: 123456789012.dkr.ecr.us-east-1.amazonaws.com/email-api
    tag: "${VERSION}"  # Will be provided at deploy time
  
  # Runtime
  runtime:
    platform: fargate
  
  # Ports
  ports:
    - name: http
      port: 8080
  
  # Environment variables
  environment:
    - name: SERVICE_NAME
      value: email-api
    
    - name: PORT
      value: "8080"
    
    - name: DATABASE_HOST
      valueFrom:
        component: email-service/database
        output: endpoint
    
    - name: QUEUE_URL
      valueFrom:
        component: email-service/queue
        output: url
  
  # Secrets (from AWS Secrets Manager)
  secrets:
    - name: DB_PASSWORD
      secretRef: /stacks/notification-platform/email-service/db-password
      envVar: DATABASE_PASSWORD
    
    - name: SMTP_PASSWORD
      secretRef: /stacks/notification-platform/email-service/smtp-password
      envVar: SMTP_PASSWORD
  
  # Application configs
  configs:
    mountPath: /config
    files:
      - app.yaml
  
  # Health checks
  healthCheck:
    readiness:
      http:
        path: /health/ready
        port: 8080
      periodSeconds: 10
    
    liveness:
      http:
        path: /health/live
        port: 8080
      periodSeconds: 30
  
  # Dependencies
  dependsOn:
    - email-service/database
    - email-service/queue
EOF
```

**Create infrastructure config:**

```bash
cat > services/email-service/components/api/infra.yaml << 'EOF'
apiVersion: infra.panka.io/v1
kind: ComponentInfra

metadata:
  name: api
  service: email-service
  stack: notification-platform

spec:
  # Resource allocation
  resources:
    cpu: 256
    memory: 512
  
  # Scaling
  scaling:
    replicas: 2
    autoscaling:
      enabled: true
      minReplicas: 2
      maxReplicas: 10
      policies:
        - type: targetTracking
          metric: CPUUtilization
          targetValue: 70
  
  # Load balancer
  networking:
    loadBalancer:
      enabled: true
      type: application
EOF
```

**Create app config file:**

```bash
mkdir -p services/email-service/components/api/configs

cat > services/email-service/components/api/configs/app.yaml << 'EOF'
app:
  name: email-api
  environment: ${ENVIRONMENT}

server:
  port: 8080
  timeout: 30s

email:
  provider: smtp
  from: noreply@company.com
  maxRetries: 3

logging:
  level: info
  format: json
EOF
```

**Create database component:**

```bash
cat > services/email-service/components/database/rds.yaml << 'EOF'
apiVersion: components.panka.io/v1
kind: RDS

metadata:
  name: database
  service: email-service
  stack: notification-platform

spec:
  engine:
    type: postgres
    version: "15.4"
  
  instance:
    class: db.t3.small
    storage:
      type: gp3
      allocatedGB: 20
  
  database:
    name: emaildb
    username: dbadmin
    passwordSecret:
      ref: /stacks/notification-platform/email-service/db-password
EOF
```

**Create queue component:**

```bash
cat > services/email-service/components/queue/sqs.yaml << 'EOF'
apiVersion: components.panka.io/v1
kind: SQS

metadata:
  name: queue
  service: email-service
  stack: notification-platform

spec:
  type: standard
  messageRetentionPeriod: 345600
  visibilityTimeout: 300
  
  deadLetterQueue:
    enabled: true
    maxReceiveCount: 3
EOF
```

### Step 7: Create Secrets

Before deploying, create secrets in AWS Secrets Manager:

```bash
# Create database password
aws secretsmanager create-secret \
  --name /stacks/notification-platform/email-service/db-password \
  --secret-string '{"password":"your-secure-password"}' \
  --region us-east-1

# Create SMTP password
aws secretsmanager create-secret \
  --name /stacks/notification-platform/email-service/smtp-password \
  --secret-string '{"password":"smtp-password"}' \
  --region us-east-1
```

### Step 8: Validate Configuration

```bash
# Go to deployment repo
cd ~/work/deployment-repo

# Validate stack
panka validate --stack notification-platform

# Output:
✓ Stack configuration is valid
✓ All services validated
✓ All components validated
✓ No circular dependencies
✓ All references resolved
```

**If errors:**

```bash
# Common issues:

❌ Error: Missing required field 'image.repository'
   File: services/email-service/components/api/microservice.yaml
   → Add image.repository field

❌ Error: Invalid reference 'email-service/cache'
   Component 'cache' not found in service 'email-service'
   → Remove invalid dependency or create cache component

❌ Error: Circular dependency detected
   api → database → api
   → Fix dependency chain
```

### Step 9: Build and Push Container Image

Before deploying, build your application:

```bash
# In your application repository
cd ~/work/email-service/

# Build Docker image
docker build -t email-api:v1.0.0 .

# Tag for ECR
docker tag email-api:v1.0.0 \
  123456789012.dkr.ecr.us-east-1.amazonaws.com/email-api:v1.0.0

# Login to ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  123456789012.dkr.ecr.us-east-1.amazonaws.com

# Push to ECR
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/email-api:v1.0.0
```

### Step 10: Deploy to Development

First deployment to development environment:

```bash
# Go back to deployment repo
cd ~/work/deployment-repo

# Plan deployment (dry-run)
panka plan \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.0

# Output shows what will be created:
┌─────────────────────────────────────────────────────────┐
│ Deployment Plan: notification-platform (development)    │
├─────────────────────────────────────────────────────────┤
│                                                          │
│ Wave 1 (parallel):                                       │
│   + email-service/database (RDS)         CREATE         │
│   + email-service/queue (SQS)            CREATE         │
│                                                          │
│ Wave 2 (after wave 1):                                  │
│   + email-service/api (MicroService)     CREATE         │
│                                                          │
├─────────────────────────────────────────────────────────┤
│ Summary:                                                 │
│   + 3 to create                                          │
│   Estimated duration: ~8 minutes                         │
│   Estimated cost: $45/month                              │
└─────────────────────────────────────────────────────────┘

Continue? (yes/no):
```

**Deploy:**

```bash
# Apply the plan
panka apply \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.0

# What happens:
# 1. Acquiring lock...
# 2. Loading state...
# 3. Building dependency graph...
# 4. Executing wave 1...
#    ✓ Creating RDS database... (5m 23s)
#    ✓ Creating SQS queue... (12s)
# 5. Executing wave 2...
#    ✓ Creating ECS service... (2m 45s)
#    ✓ Creating ALB... (1m 30s)
#    ✓ Registering targets... (45s)
# 6. Running health checks...
#    ✓ API is healthy
# 7. Saving state...
# 8. Releasing lock...
#
# ✓ Deployment successful! (8m 35s)
#
# Outputs:
#   api_url: https://dev-email-api.company.com
#   database_endpoint: email-db-dev.abc123.us-east-1.rds.amazonaws.com
```

### Step 11: Verify Deployment

```bash
# Check status
panka status \
  --stack notification-platform \
  --environment development

# Output:
┌──────────────────────────────────────────────────────────┐
│ Stack: notification-platform (development)               │
├──────────────────────────────────────────────────────────┤
│ Service: email-service                                   │
│   ✓ api        MicroService    2/2 running    Healthy   │
│   ✓ database   RDS             available      Healthy   │
│   ✓ queue      SQS             active         Healthy   │
│                                                          │
│ Last deployed: 5 minutes ago                             │
│ Deployed by: alice@company.com                           │
└──────────────────────────────────────────────────────────┘

# View logs
panka logs \
  --component email-service/api \
  --environment development \
  --follow

# Test the API
curl https://dev-email-api.company.com/health
# {"status":"healthy"}
```

### Step 12: Commit to Git

```bash
cd ~/work/deployment-repo

# Add your stack
git add stacks/notification-platform/

# Commit
git commit -m "Add notification-platform stack

- Add email-service with API, database, and queue
- Configure for development environment
- Initial version: v1.0.0"

# Push
git push origin main

# Create PR for team review
gh pr create \
  --title "Add notification platform stack" \
  --body "Initial deployment configuration for email service"
```

---

## Phase 3: Daily Usage

### Scenario 1: Deploy New Version

You've fixed a bug and want to deploy v1.0.1:

```bash
# 1. Build and push new image
cd ~/work/email-service/
docker build -t email-api:v1.0.1 .
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/email-api:v1.0.1

# 2. Deploy
cd ~/work/deployment-repo/
panka apply \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.1

# Panka detects only image tag changed
# Rolling update with zero downtime
```

### Scenario 2: Update Configuration

You need to change a config value:

```bash
# Edit config file
vim stacks/notification-platform/services/email-service/components/api/configs/app.yaml

# Change:
email:
  maxRetries: 5  # Changed from 3

# Deploy (same version, just config change)
panka apply \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.1

# Panka restarts containers with new config
```

### Scenario 3: Scale Up

Traffic is increasing:

```bash
# Edit infra config
vim stacks/notification-platform/services/email-service/components/api/infra.yaml

# Change:
scaling:
  replicas: 5        # From 2
  autoscaling:
    minReplicas: 5   # From 2
    maxReplicas: 20  # From 10

# Commit and deploy
git add .
git commit -m "Scale email-service API"
panka apply --stack notification-platform --environment development --var VERSION=v1.0.1
```

### Scenario 4: Add New Component

Add a caching layer:

```bash
# Create cache component
cat > stacks/notification-platform/services/email-service/components/cache/elasticache.yaml << 'EOF'
apiVersion: components.panka.io/v1
kind: ElastiCacheRedis

metadata:
  name: cache
  service: email-service
  stack: notification-platform

spec:
  engine:
    version: "7.0"
  
  cluster:
    mode: replication-group
    nodeType: cache.t3.micro
    numNodes: 2
EOF

# Update API to use cache
# Edit: services/email-service/components/api/microservice.yaml
# Add environment variable:
environment:
  - name: REDIS_HOST
    valueFrom:
      component: email-service/cache
      output: endpoint

# Add dependency:
dependsOn:
  - email-service/database
  - email-service/queue
  - email-service/cache  # New

# Deploy
panka apply --stack notification-platform --environment development --var VERSION=v1.0.1
```

### Scenario 5: Promote to Staging

After testing in dev, promote to staging:

```bash
# Create staging overrides (if needed)
mkdir -p stacks/notification-platform/environments/staging/services/email-service/components/api/

cat > stacks/notification-platform/environments/staging/services/email-service/components/api/infra.yaml << 'EOF'
apiVersion: infra.panka.io/v1
kind: ComponentInfra

metadata:
  name: api
  service: email-service
  stack: notification-platform
  environment: staging

spec:
  resources:
    cpu: 512      # Bigger than dev
    memory: 1024
  
  scaling:
    replicas: 3
    autoscaling:
      minReplicas: 3
      maxReplicas: 15
EOF

# Deploy to staging
panka apply \
  --stack notification-platform \
  --environment staging \
  --var VERSION=v1.0.1
```

### Scenario 6: Promote to Production

After staging verification:

```bash
# Production might require approval
panka apply \
  --stack notification-platform \
  --environment production \
  --var VERSION=v1.0.1

# Output:
⚠ Production Deployment Requires Approval

Stack: notification-platform
Environment: production
Version: v1.0.1

Changes:
  + email-service/api (new deployment)
  + email-service/database (new)
  + email-service/cache (new)
  + email-service/queue (new)

Estimated cost: $245/month

Approve? (yes/no): yes

# Deployment proceeds...
```

---

## Team Collaboration

### Multiple Team Members

**Alice** (backend developer):
```bash
# Alice deploys API changes
cd ~/work/deployment-repo
panka apply --stack notification-platform --environment dev --var VERSION=v1.0.2
```

**Bob** (trying to deploy at the same time):
```bash
# Bob tries to deploy
panka apply --stack notification-platform --environment dev --var VERSION=v1.0.3

# Output:
⚠ Stack is locked
  Locked by: alice@company.com
  Since: 2 minutes ago
  Operation: deploying v1.0.2

Waiting for lock... (Ctrl+C to cancel)
```

**After Alice finishes:**
```bash
# Bob's deployment proceeds automatically
✓ Lock acquired
Deploying...
```

### CI/CD Integration

**Setup GitHub Actions:**

```yaml
# .github/workflows/deploy-dev.yml
name: Deploy to Development

on:
  push:
    branches: [develop]
    paths:
      - 'stacks/notification-platform/**'

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
      
      # Configure AWS (assuming OIDC)
      - name: Configure AWS
        uses: aws-actions/configure-aws-credentials@v2
        with:
          role-to-assume: arn:aws:iam::123456789012:role/GithubActionsPanka
          aws-region: us-east-1
      
      # Panka uses AWS credentials from environment
      # Backend config can come from:
      # 1. ~/.panka/config.yaml (if exists in runner)
      # 2. stack.yaml override
      # 3. Environment variables
      
      - name: Deploy
        run: |
          panka apply \
            --stack notification-platform \
            --environment development \
            --var VERSION=${{ github.sha }} \
            --auto-approve
        env:
          # Optional: Override backend config via env vars
          PANKA_BACKEND_BUCKET: company-panka-state
          PANKA_BACKEND_REGION: us-east-1
          PANKA_LOCK_TABLE: company-panka-locks
```

---

## Troubleshooting

### Issue: "Failed to acquire lock"

```bash
# Check who has the lock
panka locks show --stack notification-platform --environment development

# Output:
Lock Status:
  Locked: Yes
  Lock ID: 550e8400-e29b-41d4-a716-446655440000
  Locked by: bob@company.com
  Locked at: 2024-01-15 14:30:00 (15 minutes ago)
  Last heartbeat: 2024-01-15 14:30:00 (15 minutes ago)
  Status: ⚠ STALE (no heartbeat)

# If stale, force unlock
panka unlock --stack notification-platform --environment development --force
```

### Issue: "State not found"

```bash
# First deployment for this stack/environment
# This is normal - panka will create initial state

# If state should exist but doesn't:
# Check S3 bucket
aws s3 ls s3://company-panka-state/stacks/notification-platform/development/
```

### Issue: "Component validation failed"

```bash
# Run validation to see detailed errors
panka validate --stack notification-platform

# Fix errors in YAML files
# Common issues:
# - Missing required fields
# - Invalid references
# - Typos in component names
```

---

## Best Practices

### 1. Use Git Branches

```bash
# Feature branch for changes
git checkout -b feature/add-cache
# Make changes
panka apply --stack notification-platform --environment dev
# Test
git commit -am "Add cache to email service"
git push origin feature/add-cache
# Create PR
```

### 2. Test in Lower Environments First

```
dev → staging → production
```

Always deploy and test in dev before promoting.

### 3. Use Semantic Versioning

```bash
v1.0.0 → v1.0.1  # Bug fix
v1.0.0 → v1.1.0  # New feature
v1.0.0 → v2.0.0  # Breaking change
```

### 4. Keep Configs in Sync

Application code and deployment configs should be versioned together:

```bash
# In your service repo
git tag v1.0.1

# In deployment repo
panka apply --var VERSION=v1.0.1
```

### 5. Monitor After Deployment

```bash
# Watch logs
panka logs --component email-service/api --follow

# Check metrics
panka metrics --component email-service/api --since 1h

# Verify health
panka status --stack notification-platform --environment production
```

---

## Summary

### What You Did

1. ✅ Platform team created shared backend (S3 + DynamoDB)
2. ✅ You installed panka CLI
3. ✅ You configured backend
4. ✅ You created your stack
5. ✅ You defined your service in YAML
6. ✅ You deployed to dev, staging, production
7. ✅ You integrated with CI/CD

### What You Have Now

- ✅ Panka CLI on your machine
- ✅ Stack configuration in Git
- ✅ Automated deployments
- ✅ State management (S3)
- ✅ Locking (DynamoDB)
- ✅ CI/CD integration

### Next Steps

- Add more services to your stack
- Set up monitoring and alerting
- Create runbooks
- Onboard more team members

---

## Getting Help

- **Documentation**: All docs in deployment-repo/docs/
- **Slack**: #panka-help
- **Platform Team**: platform-team@company.com
- **Office Hours**: Wednesdays 3-4 PM

---

**Welcome to panka! 🚀**


