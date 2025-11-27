# User Workflows for Application Development Teams

This guide explains how application development teams use the panka system for managing their services.

---

## Table of Contents

1. [Overview](#overview)
2. [Initial Setup](#initial-setup)
3. [Common Workflows](#common-workflows)
4. [Day-to-Day Operations](#day-to-day-operations)
5. [Team Collaboration](#team-collaboration)
6. [Troubleshooting](#troubleshooting)

---

## Overview

As an application development team, you will:
1. Define your service and components in YAML files
2. Store them in the deployment repository
3. Trigger deployments via GitHub Actions
4. Monitor your service health
5. Rollback if issues occur

**You DON'T need to:**
- Understand AWS infrastructure details (platform team manages this)
- Write Terraform/Pulumi code
- Manage state or locks
- Configure networking or security (inherited from stack defaults)

---

## Initial Setup

### Step 1: Install Panka CLI

```bash
# Install panka
curl -sSL https://panka.io/install.sh | sh

# Verify installation
panka version
```

### Step 2: Configure Backend (First Time Only)

```bash
# Initialize panka
panka init

# Interactive prompts:
? AWS Region: us-east-1
? S3 Bucket for state: company-panka-state
? DynamoDB Table for locks: company-panka-locks
? AWS Profile (optional): default

# This creates: ~/.panka/config.yaml
```

**Note**: Your platform team will provide the S3 bucket and DynamoDB table names.

### Step 3: Get Access to Deployment Repository

```bash
# Clone the deployment repository
git clone git@github.com:company/deployment-repo.git
cd deployment-repo
```

### Step 4: Understand the Structure

Your team's service lives in a specific stack:

```
deployment-repo/
└── stacks/
    └── user-platform/              # Your stack
        ├── services/
        │   └── user-service/       # Your service
        │       ├── service.yaml
        │       └── components/
        │           ├── api/
        │           ├── worker/
        │           ├── database/
        │           └── cache/
        │
        └── environments/
            ├── production/
            ├── staging/
            └── development/
```

### Step 3: Install Panka CLI (Optional)

For local validation and testing:

```bash
# Install panka CLI
curl -sSL https://panka.io/install.sh | sh

# Verify installation
panka --version

# Configure AWS credentials
aws configure --profile panka
```

---

## Common Workflows

### Workflow 1: Deploy New Service (First Time)

**Scenario**: Your team is creating a new service called "notification-service"

#### Step 1: Create Service Structure

```bash
cd deployment-repo/stacks/user-platform/services/

# Create your service directory
mkdir -p notification-service/components/{api,database,queue}
```

#### Step 2: Define Service

Create `notification-service/service.yaml`:

```yaml
apiVersion: core.panka.io/v1
kind: Service

metadata:
  name: notification-service
  stack: user-platform
  description: "Email and SMS notification service"
  
  labels:
    team: notifications
    criticality: high
  
  annotations:
    repository: "github.com/company/notification-service"
    owner: "notifications-team@company.com"
    slack: "#notifications-team"

spec:
  # Service-level defaults (optional)
  infrastructure:
    defaults: ./infra/defaults.yaml
```

#### Step 3: Define Components

Create `notification-service/components/api/microservice.yaml`:

```yaml
apiVersion: components.panka.io/v1
kind: MicroService

metadata:
  name: api
  service: notification-service
  stack: user-platform
  description: "Notification API server"

spec:
  # Your container image
  image:
    repository: 123456789012.dkr.ecr.us-east-1.amazonaws.com/notification-api
    tag: "${VERSION}"
  
  # Runtime (Fargate is default)
  runtime:
    platform: fargate
  
  # How to start your app
  container:
    command: ["/app/notification-api"]
    args:
      - "--config=/config/app.yaml"
  
  # Ports your app listens on
  ports:
    - name: http
      port: 8080
  
  # Environment variables your app needs
  environment:
    - name: SERVICE_NAME
      value: notification-api
    
    - name: DATABASE_HOST
      valueFrom:
        component: notification-service/database
        output: endpoint
    
    - name: QUEUE_URL
      valueFrom:
        component: notification-service/queue
        output: url
  
  # Secrets (managed by platform team in AWS Secrets Manager)
  secrets:
    - name: DB_PASSWORD
      secretRef: /stacks/user-platform/notification-service/db-password
      envVar: DATABASE_PASSWORD
    
    - name: SMTP_PASSWORD
      secretRef: /stacks/user-platform/notification-service/smtp-password
      envVar: SMTP_PASSWORD
  
  # Your app configuration files
  configs:
    mountPath: /config
    files:
      - app.yaml
      - logging.yaml
  
  # Health check endpoints
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
    - notification-service/database
    - notification-service/queue
```

Create `notification-service/components/api/infra.yaml`:

```yaml
apiVersion: infra.panka.io/v1
kind: ComponentInfra

metadata:
  name: api
  service: notification-service
  stack: user-platform

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
```

Create `notification-service/components/database/rds.yaml`:

```yaml
apiVersion: components.panka.io/v1
kind: RDS

metadata:
  name: database
  service: notification-service
  stack: user-platform

spec:
  engine:
    type: postgres
    version: "15.4"
  
  instance:
    class: db.t3.medium
    storage:
      type: gp3
      allocatedGB: 50
  
  database:
    name: notificationdb
    username: dbadmin
    passwordSecret:
      ref: /stacks/user-platform/notification-service/db-password
```

Create `notification-service/components/queue/sqs.yaml`:

```yaml
apiVersion: components.panka.io/v1
kind: SQS

metadata:
  name: queue
  service: notification-service
  stack: user-platform

spec:
  type: standard
  messageRetentionPeriod: 345600
  visibilityTimeout: 300
  
  deadLetterQueue:
    enabled: true
    maxReceiveCount: 3
```

#### Step 4: Add Application Configs

Create `notification-service/components/api/configs/app.yaml`:

```yaml
# Your application-specific configuration
app:
  name: notification-api
  version: 1.0.0
  environment: ${ENVIRONMENT}

server:
  host: 0.0.0.0
  port: 8080
  readTimeout: 30s
  writeTimeout: 30s

email:
  provider: smtp
  from: noreply@company.com

sms:
  provider: twilio
```

#### Step 5: Request Secrets from Platform Team

Create a ticket or PR for platform team to create secrets in AWS Secrets Manager:

```
Secrets needed:
1. /stacks/user-platform/notification-service/db-password
2. /stacks/user-platform/notification-service/smtp-password
```

#### Step 6: Create PR and Review

```bash
git checkout -b add-notification-service
git add stacks/user-platform/services/notification-service/
git commit -m "Add notification service"
git push origin add-notification-service

# Create PR
gh pr create --title "Add notification service" \
  --body "Adding new notification service with email/SMS support"
```

#### Step 7: Validation Runs Automatically

GitHub Actions will automatically:
- Validate YAML syntax
- Check schema compliance
- Run security policies
- Estimate costs
- Generate deployment plan

#### Step 8: Deploy to Development First

Once PR is merged:

```bash
# Go to GitHub Actions
# Run workflow: "Deploy Stack"
# Inputs:
#   Stack: user-platform
#   Service: notification-service  # (optional, deploys only your service)
#   Environment: development
#   Version: v1.0.0
```

Or use CLI locally:

```bash
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment development \
  --var VERSION=v1.0.0
```

#### Step 9: Test in Development

```bash
# Check service status
panka show \
  --stack user-platform \
  --component notification-service/api \
  --environment development

# View logs
panka logs \
  --stack user-platform \
  --component notification-service/api \
  --environment development \
  --follow
```

#### Step 10: Promote to Staging, then Production

```bash
# Deploy to staging
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment staging \
  --var VERSION=v1.0.0

# After testing, deploy to production
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment production \
  --var VERSION=v1.0.0
```

---

### Workflow 2: Update Existing Service (Deploy New Version)

**Scenario**: You've built a new version of your API and want to deploy it.

#### Step 1: Build and Push Container Image

```bash
# Build your app
cd ~/projects/notification-service
docker build -t notification-api:v1.1.0 .

# Tag for ECR
docker tag notification-api:v1.1.0 \
  123456789012.dkr.ecr.us-east-1.amazonaws.com/notification-api:v1.1.0

# Push to ECR
aws ecr get-login-password | docker login --username AWS --password-stdin \
  123456789012.dkr.ecr.us-east-1.amazonaws.com
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/notification-api:v1.1.0
```

#### Step 2: Deploy New Version

**Option A: Via GitHub Actions (Recommended)**

```
Go to GitHub Actions → Deploy Stack
Inputs:
  Stack: user-platform
  Service: notification-service
  Environment: production
  Version: v1.1.0
```

**Option B: Via CLI**

```bash
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment production \
  --var VERSION=v1.1.0
```

#### Step 3: Monitor Deployment

```bash
# Watch deployment progress
panka status \
  --stack user-platform \
  --environment production \
  --follow

# View metrics
panka metrics \
  --component notification-service/api \
  --environment production
```

#### Step 4: Verify Deployment

```bash
# Check health
panka health \
  --component notification-service/api \
  --environment production

# Run smoke tests
panka test \
  --service notification-service \
  --environment production
```

#### Step 5: Rollback if Issues

If something goes wrong:

```bash
panka rollback \
  --stack user-platform \
  --service notification-service \
  --environment production
```

---

### Workflow 3: Update Configuration Only

**Scenario**: You need to change an environment variable or config file.

#### Step 1: Update Configuration

Edit `notification-service/components/api/configs/app.yaml`:

```yaml
app:
  name: notification-api
  version: 1.0.0

email:
  provider: smtp
  from: noreply@company.com
  maxRetries: 5  # ← Changed from 3 to 5
```

Or update environment variable in `notification-service/components/api/microservice.yaml`:

```yaml
environment:
  - name: LOG_LEVEL
    value: debug  # ← Changed from info to debug
```

#### Step 2: Commit and Deploy

```bash
git checkout -b update-notification-config
git add stacks/user-platform/services/notification-service/
git commit -m "Update notification config: increase retry count"
git push origin update-notification-config

# Create PR and merge after review

# Deploy
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment production \
  --var VERSION=v1.1.0  # Same version, just config change
```

**Note**: Panka will detect only config changed and will restart containers with new config.

---

### Workflow 4: Scale Service (Change Resources)

**Scenario**: Your service needs more CPU/memory or more replicas.

#### Step 1: Update Infrastructure Config

Edit `notification-service/components/api/infra.yaml`:

```yaml
spec:
  resources:
    cpu: 512      # ← Changed from 256
    memory: 1024  # ← Changed from 512
  
  scaling:
    replicas: 5   # ← Changed from 2
    
    autoscaling:
      minReplicas: 5   # ← Changed from 2
      maxReplicas: 20  # ← Changed from 10
```

#### Step 2: Deploy (Config changes typically go directly)

```bash
git checkout -b scale-notification-api
git add stacks/user-platform/services/notification-service/components/api/infra.yaml
git commit -m "Scale notification API: increase resources"
git push origin scale-notification-api

# Create PR
# After approval, merge and deploy
```

**Production Override** (if you only want to scale in production):

Edit `environments/production/services/notification-service/components/api/infra.yaml`:

```yaml
apiVersion: infra.panka.io/v1
kind: ComponentInfra

metadata:
  name: api
  service: notification-service
  stack: user-platform
  environment: production

spec:
  # Only override what's different in production
  resources:
    cpu: 1024
    memory: 2048
  
  scaling:
    minReplicas: 10
    maxReplicas: 50
```

---

### Workflow 5: Add New Component (e.g., Cache)

**Scenario**: Your service needs a Redis cache for performance.

#### Step 1: Add Cache Component

Create `notification-service/components/cache/elasticache.yaml`:

```yaml
apiVersion: components.panka.io/v1
kind: ElastiCacheRedis

metadata:
  name: cache
  service: notification-service
  stack: user-platform

spec:
  engine:
    version: "7.0"
  
  cluster:
    mode: replication-group
    nodeType: cache.t3.medium
    numNodes: 2
  
  security:
    atRestEncryption: true
    transitEncryption: true
```

#### Step 2: Update API to Use Cache

Edit `notification-service/components/api/microservice.yaml`:

```yaml
environment:
  # Add cache endpoint
  - name: CACHE_HOST
    valueFrom:
      component: notification-service/cache
      output: endpoint
  
  - name: CACHE_PORT
    value: "6379"

# Add dependency
dependsOn:
  - notification-service/database
  - notification-service/queue
  - notification-service/cache  # ← New dependency
```

#### Step 3: Update Application Config

Edit `notification-service/components/api/configs/app.yaml`:

```yaml
cache:
  enabled: true
  ttl: 3600
```

#### Step 4: Deploy

```bash
git checkout -b add-notification-cache
git add stacks/user-platform/services/notification-service/
git commit -m "Add Redis cache to notification service"
git push origin add-notification-cache

# Create PR, review, merge, deploy
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment production \
  --var VERSION=v1.1.0
```

---

## Day-to-Day Operations

### Check Service Status

```bash
# Check all components in your service
panka status \
  --service notification-service \
  --environment production

Output:
┌──────────────────────────────────────────────────────────────┐
│ Service: notification-service (production)                    │
├──────────────────────────────────────────────────────────────┤
│ ✓ api          MicroService    5/5 running    Healthy       │
│ ✓ database     RDS             available      Healthy       │
│ ✓ cache        ElastiCache     available      Healthy       │
│ ✓ queue        SQS             active         Healthy       │
└──────────────────────────────────────────────────────────────┘
```

### View Logs

```bash
# View logs for your API
panka logs \
  --component notification-service/api \
  --environment production \
  --follow \
  --since 1h

# Filter logs
panka logs \
  --component notification-service/api \
  --environment production \
  --filter "ERROR" \
  --since 24h
```

### View Metrics

```bash
# View metrics
panka metrics \
  --component notification-service/api \
  --environment production \
  --since 1h

Output:
┌──────────────────────────────────────────────────────────────┐
│ Metrics: notification-service/api (last 1 hour)              │
├──────────────────────────────────────────────────────────────┤
│ Request Count:    125,432                                    │
│ Error Rate:       0.02%                                      │
│ P50 Latency:      45ms                                       │
│ P99 Latency:      320ms                                      │
│ CPU Usage:        45%                                        │
│ Memory Usage:     62%                                        │
└──────────────────────────────────────────────────────────────┘
```

### Check Deployment History

```bash
# View deployment history
panka history \
  --service notification-service \
  --environment production

Output:
┌──────────────────────────────────────────────────────────────┐
│ Deployment History: notification-service (production)        │
├──────────────────────────────────────────────────────────────┤
│ v1.1.0  2024-01-15 14:30  alice@company.com  Success  5m32s │
│ v1.0.9  2024-01-14 10:15  bob@company.com    Success  4m18s │
│ v1.0.8  2024-01-13 16:45  alice@company.com  Rolled back    │
│ v1.0.7  2024-01-12 11:20  bob@company.com    Success  4m55s │
└──────────────────────────────────────────────────────────────┘
```

### Detect Configuration Drift

```bash
# Check if anyone made manual changes in AWS console
panka drift detect \
  --service notification-service \
  --environment production

Output:
┌──────────────────────────────────────────────────────────────┐
│ Drift Detection: notification-service (production)           │
├──────────────────────────────────────────────────────────────┤
│ ⚠ MEDIUM: api (MicroService)                                │
│   Desired replicas: 5                                        │
│   Actual replicas: 3                                         │
│   → Someone manually scaled down the service                 │
│                                                               │
│ ✓ All other components match desired state                  │
└──────────────────────────────────────────────────────────────┘

# Fix drift
panka drift remediate \
  --service notification-service \
  --environment production
```

---

## Team Collaboration

### Multiple Teams Working on Same Stack

**Scenario**: Team A works on `user-service`, Team B works on `notification-service`

Both services are in the `user-platform` stack. Teams can work independently:

```bash
# Team A deploys user-service
panka apply \
  --stack user-platform \
  --service user-service \
  --environment production

# Team B deploys notification-service (can run concurrently)
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment production
```

**Locking ensures safety:**
- If both teams try to deploy the same service simultaneously, one will wait
- Different services can deploy in parallel
- Stack-level lock prevents conflicting changes

### Cross-Service Dependencies

**Scenario**: Your service depends on another team's service

```yaml
# In your microservice.yaml
environment:
  - name: USER_SERVICE_URL
    valueFrom:
      component: user-service/api  # ← Reference another team's service
      output: url
```

**Communication:**
- Coordinate with other team via Slack/email
- Document dependencies in service.yaml annotations
- Test integration in staging before production

### Environment Promotion

**Best Practice**: Always deploy in order: dev → staging → production

```bash
# Step 1: Deploy to dev
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment development \
  --var VERSION=v1.2.0

# Step 2: Test thoroughly in dev

# Step 3: Promote to staging
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment staging \
  --var VERSION=v1.2.0

# Step 4: Test in staging

# Step 5: Promote to production (requires approval)
panka apply \
  --stack user-platform \
  --service notification-service \
  --environment production \
  --var VERSION=v1.2.0
```

---

## Troubleshooting

### Deployment Failed

```bash
# Check deployment status
panka status --service notification-service --environment production

# View detailed error
panka show \
  --component notification-service/api \
  --environment production \
  --show-events

# View logs around failure time
panka logs \
  --component notification-service/api \
  --environment production \
  --since 30m

# If needed, rollback
panka rollback \
  --service notification-service \
  --environment production
```

### Health Checks Failing

```bash
# Check health check status
panka health \
  --component notification-service/api \
  --environment production

Output:
┌──────────────────────────────────────────────────────────────┐
│ Health: notification-service/api                             │
├──────────────────────────────────────────────────────────────┤
│ Readiness:  ✗ Failing                                        │
│   Last check: 2024-01-15 14:35:22                           │
│   Error: Connection refused to http://localhost:8080/health │
│                                                               │
│ Liveness:   ✓ Passing                                        │
└──────────────────────────────────────────────────────────────┘

# Check if app is actually listening on the right port
panka exec \
  --component notification-service/api \
  --environment production \
  --command "netstat -tuln"
```

### Service Not Accessible

```bash
# Check load balancer status
panka show \
  --component notification-service/api \
  --environment production \
  --show-loadbalancer

# Check security groups
panka show \
  --component notification-service/api \
  --environment production \
  --show-networking

# Test connectivity from within VPC
panka exec \
  --component notification-service/api \
  --environment production \
  --command "curl http://localhost:8080/health"
```

### Database Connection Issues

```bash
# Check database status
panka status \
  --component notification-service/database \
  --environment production

# Verify connection string
panka show \
  --component notification-service/database \
  --environment production

# Check if API has correct endpoint
panka show \
  --component notification-service/api \
  --environment production \
  --show-environment | grep DATABASE
```

### Deployment Stuck

```bash
# Check if lock is held
panka state locks \
  --stack user-platform \
  --environment production

Output:
┌──────────────────────────────────────────────────────────────┐
│ Active Locks                                                 │
├──────────────────────────────────────────────────────────────┤
│ Stack: user-platform                                         │
│ Environment: production                                      │
│ Locked by: github-actions-run-12345                         │
│ Locked at: 2024-01-15 13:30:00 (2 hours ago)               │
│ Status: Stale (no heartbeat for 1 hour)                    │
└──────────────────────────────────────────────────────────────┘

# If lock is stale, force unlock
panka unlock \
  --stack user-platform \
  --environment production \
  --force
```

### Need Help from Platform Team

When to escalate to platform team:
- Infrastructure issues (networking, security groups, IAM)
- Stack-level configuration problems
- Resource quota limits
- AWS service issues
- Secrets management
- Cost optimization

Contact: `#platform-team` on Slack or `platform-team@company.com`

---

## Best Practices

### 1. Always Test in Lower Environments First
```
dev → staging → production
```

### 2. Use Semantic Versioning
```
v1.0.0 → v1.0.1 (patch - bug fixes)
v1.0.0 → v1.1.0 (minor - new features, backward compatible)
v1.0.0 → v2.0.0 (major - breaking changes)
```

### 3. Keep Application Config Separate from Infrastructure
- Application settings → `configs/app.yaml`
- Infrastructure settings → `infra.yaml`

### 4. Document Your Service
Add clear metadata to your service:

```yaml
metadata:
  description: "Clear description of what this service does"
  annotations:
    runbook: "https://wiki.company.com/notification-service/runbook"
    dashboard: "https://grafana.company.com/d/notification-service"
    owner: "notifications-team@company.com"
    slack: "#notifications-team"
```

### 5. Monitor Your Service
- Set up alerts for error rates, latency, etc.
- Check metrics regularly
- Run smoke tests after deployments

### 6. Small, Frequent Deployments
Better than large, infrequent ones:
- Easier to rollback
- Faster to identify issues
- Lower risk

### 7. Use Git Branching
```bash
# Feature branch
git checkout -b feature/add-sms-support

# Make changes
git commit -m "Add SMS support to notification service"

# Create PR for review
gh pr create

# After approval, merge and deploy
```

---

## Quick Reference

### Common Commands

```bash
# Deploy service
panka apply --stack STACK --service SERVICE --environment ENV --var VERSION=X

# Check status
panka status --service SERVICE --environment ENV

# View logs
panka logs --component COMPONENT --environment ENV --follow

# Rollback
panka rollback --service SERVICE --environment ENV

# Show history
panka history --service SERVICE --environment ENV

# Detect drift
panka drift detect --service SERVICE --environment ENV

# Unlock stuck deployment
panka unlock --stack STACK --environment ENV --force
```

### File Locations

```
Your service files:
  stacks/{stack}/services/{your-service}/
    ├── service.yaml
    ├── components/
    │   └── {component}/
    │       ├── {type}.yaml (microservice.yaml, rds.yaml, etc.)
    │       ├── infra.yaml
    │       └── configs/
    └── secrets.yaml

Environment overrides:
  stacks/{stack}/environments/{env}/services/{your-service}/
    └── components/
        └── {component}/
            ├── {type}.yaml
            └── infra.yaml
```

### Getting Help

- Documentation: `https://docs.company.com/panka`
- Slack: `#panka-help`
- Platform Team: `platform-team@company.com`
- On-call: `https://pagerduty.com/teams/platform`

---

## Summary

As an application development team:

1. **Define** your service in YAML (application concerns)
2. **Configure** infrastructure needs (platform team helps)
3. **Deploy** via GitHub Actions or CLI
4. **Monitor** via metrics and logs
5. **Rollback** if issues occur
6. **Iterate** with small, frequent deployments

You own your application code and configuration. Platform team owns infrastructure, networking, and security policies.

**The panka system handles all the complexity of AWS deployments for you!**

