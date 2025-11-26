# End User Workflow Summary

This document provides a comprehensive overview of how application development teams interact with the deployer system on a day-to-day basis.

---

## Your Role as an Application Development Team

As an application development team, you are responsible for:

1. **Your application code** (in your service repository)
2. **Your service configuration** (in the deployment repository)
3. **Deploying your service** (via GitHub Actions or CLI)
4. **Monitoring your service** (logs, metrics, alerts)

You are **NOT** responsible for:
- Infrastructure setup (VPC, networking, security groups)
- AWS account management
- Secrets rotation
- Cross-stack networking
- Cost optimization at infrastructure level

The **Platform Team** handles all infrastructure concerns. You just define what you need, and the deployer handles provisioning it.

---

## The Two Repositories You Work With

### 1. Your Service Repository (Application Code)
```
github.com/company/notification-service/
├── cmd/
│   └── api/
│       └── main.go           # Your application code
├── pkg/
│   ├── handlers/
│   ├── models/
│   └── services/
├── Dockerfile                 # How to build your container
├── go.mod
└── .github/
    └── workflows/
        └── build.yml          # Builds & pushes to ECR
```

**What you do here:**
- Write application code
- Build Docker images
- Push to ECR
- Tag releases

### 2. Deployment Repository (Infrastructure as Code)
```
github.com/company/deployment-repo/
└── stacks/
    └── user-platform/
        └── services/
            └── notification-service/
                ├── service.yaml           # Service definition
                └── components/
                    ├── api/
                    │   ├── microservice.yaml   # What to deploy
                    │   ├── infra.yaml          # How much resources
                    │   └── configs/            # App configs
                    └── database/
                        └── rds.yaml            # Database config
```

**What you do here:**
- Define your service components
- Configure resources (CPU, memory, replicas)
- Set environment variables
- Mount configuration files
- Trigger deployments

---

## Daily Workflows

### Workflow 1: Deploy a New Version (Most Common)

**Scenario:** You've finished a feature and want to deploy it to production.

#### Step 1: Build and Tag Your Application

```bash
# In your service repo
cd ~/work/notification-service/

# Ensure tests pass
go test ./...

# Build Docker image
docker build -t notification-api:v1.2.0 .

# Tag for ECR
docker tag notification-api:v1.2.0 \
  123456789012.dkr.ecr.us-east-1.amazonaws.com/notification-api:v1.2.0

# Push to ECR
aws ecr get-login-password | docker login --username AWS --password-stdin \
  123456789012.dkr.ecr.us-east-1.amazonaws.com

docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/notification-api:v1.2.0
```

**Or use GitHub Actions:**
```yaml
# In your service repo: .github/workflows/build.yml
name: Build and Push

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build and push
        run: |
          docker build -t notification-api:${{ github.ref_name }} .
          docker tag notification-api:${{ github.ref_name }} \
            ${{ secrets.ECR_REGISTRY }}/notification-api:${{ github.ref_name }}
          docker push ${{ secrets.ECR_REGISTRY }}/notification-api:${{ github.ref_name }}
```

#### Step 2: Deploy to Development

```bash
# Go to GitHub: company/deployment-repo
# Navigate to Actions → "Deploy Stack"
# Fill in:
#   Stack: user-platform
#   Service: notification-service  (optional - deploys only your service)
#   Environment: development
#   Version: v1.2.0
# Click "Run workflow"
```

**Or use CLI:**
```bash
cd ~/work/deployment-repo/

deployer apply \
  --stack user-platform \
  --service notification-service \
  --environment development \
  --var VERSION=v1.2.0
```

**What happens:**
```
1. Deployer acquires lock ────────────┐
2. Loads current state                 │ (Automatic)
3. Computes changes needed             │
4. Generates deployment plan           │
5. Deploys to AWS via Pulumi ─────────┘
6. Runs health checks
7. Updates state
8. Releases lock
9. Sends Slack notification
```

#### Step 3: Test in Development

```bash
# Check deployment status
deployer status \
  --service notification-service \
  --environment development

# View logs
deployer logs \
  --component notification-service/api \
  --environment development \
  --follow

# View metrics
deployer metrics \
  --component notification-service/api \
  --environment development

# Test your endpoints
curl https://dev-api.company.com/notifications/health
```

#### Step 4: Promote to Staging

```bash
# Deploy to staging (same version)
deployer apply \
  --stack user-platform \
  --service notification-service \
  --environment staging \
  --var VERSION=v1.2.0

# Run integration tests
deployer test \
  --service notification-service \
  --environment staging
```

#### Step 5: Promote to Production

```bash
# Deploy to production
deployer apply \
  --stack user-platform \
  --service notification-service \
  --environment production \
  --var VERSION=v1.2.0

# ⚠ For production, manual approval is required
# You'll see:
┌──────────────────────────────────────────────────────────┐
│ Production Deployment Approval Required                   │
├──────────────────────────────────────────────────────────┤
│ Stack: user-platform                                      │
│ Service: notification-service                             │
│ Version: v1.2.0                                           │
│                                                           │
│ Changes:                                                  │
│   • api: Update image v1.1.0 → v1.2.0                    │
│                                                           │
│ Estimated cost change: +$0.00/month                       │
│                                                           │
│ Approvers:                                                │
│   - notifications-team-lead                               │
│   - platform-team                                         │
│                                                           │
│ Approve this deployment? (yes/no):                       │
└──────────────────────────────────────────────────────────┘
```

#### Step 6: Monitor Production Deployment

```bash
# Watch deployment progress
deployer status \
  --service notification-service \
  --environment production \
  --follow

# Real-time output:
┌──────────────────────────────────────────────────────────┐
│ Deployment Progress: notification-service                 │
├──────────────────────────────────────────────────────────┤
│ Wave 1: Dependencies                                      │
│   ✓ database          Available         (30s)            │
│   ✓ cache             Available         (45s)            │
│   ✓ queue             Active            (15s)            │
│                                                           │
│ Wave 2: Application                                       │
│   ⟳ api               Rolling update    (2/5 running)    │
│     └─ Health checks: Passing                            │
│                                                           │
│ Elapsed: 2m 15s                                           │
└──────────────────────────────────────────────────────────┘

# After deployment completes:
✓ Deployment successful!
  Duration: 4m 32s
  Version: v1.2.0
  Resources updated: 1
  
  Next steps:
  - Monitor metrics: deployer metrics --component notification-service/api
  - View logs: deployer logs --component notification-service/api --follow
  - Dashboard: https://grafana.company.com/d/notification-service
```

#### Step 7: Rollback if Needed

If something goes wrong:

```bash
# Automatic rollback triggers:
# - Error rate > 5%
# - P99 latency > 2000ms
# - Health checks failing

# Manual rollback:
deployer rollback \
  --service notification-service \
  --environment production

# Rollback to specific version:
deployer rollback \
  --service notification-service \
  --environment production \
  --to-version v1.1.0
```

---

### Workflow 2: Change Configuration (No Code Change)

**Scenario:** You need to increase retry count in your app config.

#### Step 1: Update Configuration File

```bash
cd ~/work/deployment-repo/

# Edit app config
vim stacks/user-platform/services/notification-service/components/api/configs/app.yaml
```

```yaml
# Before:
email:
  maxRetries: 3
  retryDelay: 5s

# After:
email:
  maxRetries: 5      # ← Changed
  retryDelay: 10s    # ← Changed
```

#### Step 2: Commit and Deploy

```bash
git checkout -b update-notification-retry
git add stacks/user-platform/services/notification-service/components/api/configs/
git commit -m "Increase email retry count and delay"
git push origin update-notification-retry

# Create PR
gh pr create --title "Update notification retry config" \
  --body "Increase retry count from 3 to 5 and delay from 5s to 10s"

# After review and merge, deploy:
deployer apply \
  --stack user-platform \
  --service notification-service \
  --environment production \
  --var VERSION=v1.2.0  # Same version, just config change
```

**What happens:**
- Deployer detects only config changed
- Restarts containers with new config
- No new image is pulled
- Faster deployment (~1-2 minutes)

---

### Workflow 3: Scale Your Service

**Scenario:** Your service is getting more traffic and needs more resources.

#### Step 1: Update Infrastructure Config

```bash
cd ~/work/deployment-repo/

# Edit infra config
vim stacks/user-platform/services/notification-service/components/api/infra.yaml
```

```yaml
# Before:
spec:
  resources:
    cpu: 256
    memory: 512
  
  scaling:
    replicas: 2
    autoscaling:
      minReplicas: 2
      maxReplicas: 10

# After:
spec:
  resources:
    cpu: 512       # ← Doubled
    memory: 1024   # ← Doubled
  
  scaling:
    replicas: 5    # ← Increased
    autoscaling:
      minReplicas: 5    # ← Increased
      maxReplicas: 20   # ← Doubled
```

#### Step 2: Deploy

```bash
git checkout -b scale-notification-api
git add stacks/user-platform/services/notification-service/components/api/infra.yaml
git commit -m "Scale notification API: increase resources and replicas"
git push origin scale-notification-api

# Create PR and merge

# Deploy
deployer apply \
  --stack user-platform \
  --service notification-service \
  --environment production \
  --var VERSION=v1.2.0
```

**Cost estimate before deployment:**
```
┌──────────────────────────────────────────────────────────┐
│ Cost Impact Analysis                                      │
├──────────────────────────────────────────────────────────┤
│ Current monthly cost: $245.00                             │
│ New monthly cost: $612.50                                 │
│ Increase: +$367.50/month (+150%)                          │
│                                                           │
│ Breakdown:                                                │
│   • ECS Tasks (5x): $490.00/month                        │
│   • ALB: $22.50/month                                    │
│   • RDS: $100.00/month                                   │
│                                                           │
│ Continue? (yes/no):                                      │
└──────────────────────────────────────────────────────────┘
```

---

### Workflow 4: Add a New Component

**Scenario:** Your service needs a Redis cache for performance.

#### Step 1: Add Cache Component

```bash
cd ~/work/deployment-repo/

mkdir -p stacks/user-platform/services/notification-service/components/cache/

# Create cache definition
cat > stacks/user-platform/services/notification-service/components/cache/elasticache.yaml << 'EOF'
apiVersion: components.deployer.io/v1
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
EOF
```

#### Step 2: Update API to Use Cache

```bash
# Edit API component
vim stacks/user-platform/services/notification-service/components/api/microservice.yaml
```

```yaml
# Add environment variable:
environment:
  - name: CACHE_HOST
    valueFrom:
      component: notification-service/cache
      output: endpoint
  
  - name: CACHE_PORT
    value: "6379"
  
  - name: CACHE_ENABLED
    value: "true"

# Add dependency:
dependsOn:
  - notification-service/database
  - notification-service/cache  # ← New
```

#### Step 3: Update Application Config

```bash
# Edit app config
vim stacks/user-platform/services/notification-service/components/api/configs/app.yaml
```

```yaml
cache:
  enabled: true
  ttl: 3600
  maxSize: 10000
```

#### Step 4: Update Application Code

```bash
# In your service repo
cd ~/work/notification-service/

# Add cache support to your application code
# ... (your code changes)

# Build new version
docker build -t notification-api:v1.3.0 .
docker push ...
```

#### Step 5: Deploy Everything

```bash
cd ~/work/deployment-repo/

git checkout -b add-redis-cache
git add stacks/user-platform/services/notification-service/
git commit -m "Add Redis cache to notification service"
git push origin add-redis-cache

# Create PR, review, merge

# Deploy
deployer apply \
  --stack user-platform \
  --service notification-service \
  --environment production \
  --var VERSION=v1.3.0  # ← New version with cache support
```

**Deployment plan:**
```
┌──────────────────────────────────────────────────────────┐
│ Deployment Plan                                           │
├──────────────────────────────────────────────────────────┤
│ Wave 1 (new dependency):                                  │
│   + cache (ElastiCacheRedis)    CREATE    (~8 minutes)   │
│                                                           │
│ Wave 2 (after cache is ready):                           │
│   ⚠ api (MicroService)           UPDATE                  │
│     - image: v1.2.0 → v1.3.0                             │
│     - environment: +CACHE_HOST, +CACHE_PORT              │
│                                                           │
│ Estimated cost: +$85/month (ElastiCache)                  │
└──────────────────────────────────────────────────────────┘
```

---

## Day-to-Day Operations

### Check Service Health

```bash
# Quick status check
deployer status \
  --service notification-service \
  --environment production

Output:
┌──────────────────────────────────────────────────────────┐
│ Service: notification-service (production)                │
├──────────────────────────────────────────────────────────┤
│ Component     Type            Status      Health         │
├──────────────────────────────────────────────────────────┤
│ api           MicroService    5/5 running  ✓ Healthy     │
│ database      RDS             available    ✓ Healthy     │
│ cache         ElastiCache     available    ✓ Healthy     │
│ queue         SQS             active       ✓ Healthy     │
│                                                           │
│ Last deployed: 2024-01-15 14:30 (2 hours ago)            │
│ Version: v1.3.0                                           │
│ Deployed by: alice@company.com                            │
└──────────────────────────────────────────────────────────┘
```

### View Logs

```bash
# Stream logs
deployer logs \
  --component notification-service/api \
  --environment production \
  --follow

# Filter errors
deployer logs \
  --component notification-service/api \
  --environment production \
  --filter "ERROR" \
  --since 1h

# Search logs
deployer logs \
  --component notification-service/api \
  --environment production \
  --search "email sent" \
  --since 24h
```

### Check Metrics

```bash
deployer metrics \
  --component notification-service/api \
  --environment production \
  --since 1h

Output:
┌──────────────────────────────────────────────────────────┐
│ Metrics: notification-service/api (last 1 hour)          │
├──────────────────────────────────────────────────────────┤
│                                                           │
│ Request Metrics:                                          │
│   Total requests:     125,432                             │
│   Success rate:       99.98%                              │
│   Error rate:         0.02%                               │
│                                                           │
│ Latency:                                                  │
│   P50:               45ms                                 │
│   P95:               156ms                                │
│   P99:               320ms                                │
│   Max:               1,243ms                              │
│                                                           │
│ Resource Usage:                                           │
│   CPU:               45% (avg) / 78% (max)               │
│   Memory:            62% (avg) / 85% (max)               │
│                                                           │
│ Autoscaling:                                              │
│   Current replicas:  5                                    │
│   Min replicas:      5                                    │
│   Max replicas:      20                                   │
│   Target CPU:        70%                                  │
│                                                           │
│ ✓ All metrics within normal range                        │
└──────────────────────────────────────────────────────────┘
```

### View Deployment History

```bash
deployer history \
  --service notification-service \
  --environment production \
  --limit 10

Output:
┌──────────────────────────────────────────────────────────┐
│ Deployment History: notification-service (production)     │
├──────────────────────────────────────────────────────────┤
│ Version  Date       Time      By              Duration   │
├──────────────────────────────────────────────────────────┤
│ v1.3.0   Jan 15    14:30     alice@company    4m 32s ✓  │
│ v1.2.0   Jan 14    10:15     bob@company      4m 18s ✓  │
│ v1.1.9   Jan 13    16:45     alice@company    Rolled back│
│ v1.1.8   Jan 12    11:20     bob@company      5m 12s ✓  │
│ v1.1.7   Jan 11    09:30     alice@company    4m 55s ✓  │
└──────────────────────────────────────────────────────────┘
```

### Detect Drift

```bash
# Check for manual changes
deployer drift detect \
  --service notification-service \
  --environment production

Output:
┌──────────────────────────────────────────────────────────┐
│ Drift Detection: notification-service                     │
├──────────────────────────────────────────────────────────┤
│                                                           │
│ ⚠ MEDIUM: api (MicroService)                             │
│   Configuration drift detected                            │
│                                                           │
│   Desired state:                                          │
│     replicas: 5                                           │
│     cpu: 512                                              │
│     memory: 1024                                          │
│                                                           │
│   Actual state:                                           │
│     replicas: 3          ← Drift detected                │
│     cpu: 512                                              │
│     memory: 1024                                          │
│                                                           │
│   Likely cause: Manual scaling via AWS console           │
│   Detected: 2024-01-15 16:45:00                          │
│                                                           │
│ ✓ All other components match desired state               │
│                                                           │
│ Actions:                                                  │
│   [Remediate] - Restore to desired state (5 replicas)    │
│   [Acknowledge] - Mark as expected (update config)        │
│   [Ignore] - Suppress future alerts                      │
└──────────────────────────────────────────────────────────┘

# Fix drift
deployer drift remediate \
  --service notification-service \
  --environment production
```

---

## Troubleshooting Common Issues

### Issue 1: Deployment Failed

```bash
# Check what failed
deployer status \
  --service notification-service \
  --environment production

# Get detailed error
deployer show \
  --component notification-service/api \
  --environment production \
  --show-events

# View logs around failure time
deployer logs \
  --component notification-service/api \
  --environment production \
  --since 30m \
  --filter "ERROR"

# Rollback if needed
deployer rollback \
  --service notification-service \
  --environment production
```

### Issue 2: Health Checks Failing

```bash
# Check health status
deployer health \
  --component notification-service/api \
  --environment production

# Exec into container
deployer exec \
  --component notification-service/api \
  --environment production \
  --command "/bin/sh"

# Inside container:
$ curl http://localhost:8080/health
$ netstat -tuln | grep 8080
$ cat /config/app.yaml
```

### Issue 3: Can't Deploy (Lock Held)

```bash
# Check lock status
deployer state locks \
  --stack user-platform \
  --environment production

# If lock is stale, force unlock
deployer unlock \
  --stack user-platform \
  --environment production \
  --force
```

---

## Team Collaboration

### Multiple Teams in Same Stack

Your team (`notification-service`) and another team (`user-service`) are both in `user-platform` stack:

```bash
# Team A: Deploys user-service
deployer apply \
  --stack user-platform \
  --service user-service \
  --environment production

# Team B: Deploys notification-service (can run in parallel)
deployer apply \
  --stack user-platform \
  --service notification-service \
  --environment production
```

**The deployer ensures:**
- Both teams can deploy simultaneously (service-level locking)
- No conflicts between deployments
- Dependency resolution across services

### Cross-Service Dependencies

If your service depends on another team's service:

```yaml
# In your microservice.yaml
environment:
  - name: USER_SERVICE_URL
    valueFrom:
      component: user-service/api  # ← Reference other team's service
      output: url
```

**Coordination:**
- Slack the other team if you need their output
- Document dependencies in annotations
- Test integration in staging first

---

## Best Practices

### 1. Always Test in Lower Environments First
```
dev → staging → production
```
Never skip environments!

### 2. Use Semantic Versioning
```
v1.0.0 → v1.0.1  (bug fixes)
v1.0.0 → v1.1.0  (new features, backward compatible)
v1.0.0 → v2.0.0  (breaking changes)
```

### 3. Keep Deployments Small and Frequent
- Deploy often (multiple times per day)
- Small changes are easier to rollback
- Faster to identify issues

### 4. Monitor After Deployment
- Check metrics for 15-30 minutes after deployment
- Watch error rates and latency
- Be ready to rollback

### 5. Document Your Service
```yaml
metadata:
  annotations:
    runbook: "https://wiki.company.com/notification-service/runbook"
    dashboard: "https://grafana.company.com/d/notification-service"
    oncall: "https://pagerduty.com/services/notification-service"
```

---

## Quick Reference Card

```bash
# Most Common Commands

# Deploy
deployer apply --stack STACK --service SERVICE --environment ENV --var VERSION=X

# Check status
deployer status --service SERVICE --environment ENV

# View logs
deployer logs --component COMPONENT --environment ENV --follow

# Rollback
deployer rollback --service SERVICE --environment ENV

# Check history
deployer history --service SERVICE --environment ENV

# Detect drift
deployer drift detect --service SERVICE --environment ENV
```

---

## Getting Help

### Self-Service
- Documentation: https://docs.company.com/deployer
- Runbooks: https://wiki.company.com/deployer
- FAQs: https://wiki.company.com/deployer/faq

### Community
- Slack: #deployer-help
- Office Hours: Wednesdays 3-4 PM (Platform Team)

### Support
- Email: platform-team@company.com
- On-call (emergencies): https://pagerduty.com/teams/platform
- GitHub Issues: https://github.com/company/deployer/issues

---

**Summary: As an app team, you define WHAT you want (in YAML), and the deployer handles HOW to provision it (on AWS). Simple!**

