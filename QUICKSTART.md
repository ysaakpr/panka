# Deployer QuickStart

Get started with deployer in 3 simple phases.

---

## The 3-Phase Journey

```
┌─────────────────────────────────────────────────────────────────┐
│ Phase 1: PLATFORM TEAM SETUP (Once per organization)            │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Platform team creates:                                          │
│  ✓ S3 bucket: company-deployer-state                            │
│  ✓ DynamoDB table: company-deployer-locks                       │
│  ✓ Deployment repository                                         │
│                                                                   │
│  Duration: 30 minutes                                            │
│  Who: Platform/DevOps team                                       │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘

                            ↓

┌─────────────────────────────────────────────────────────────────┐
│ Phase 2: DEVELOPER ONBOARDING (Once per team)                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Each developer:                                                 │
│  1. Install CLI: curl -sSL deployer.io/install.sh | sh         │
│  2. Configure: deployer init                                     │
│  3. Define service in YAML                                       │
│  4. Deploy: deployer apply                                       │
│                                                                   │
│  Duration: 1 hour                                                │
│  Who: Each development team                                      │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘

                            ↓

┌─────────────────────────────────────────────────────────────────┐
│ Phase 3: DAILY USAGE (Ongoing)                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Developers:                                                     │
│  • deployer apply --var VERSION=v1.0.1  (deploy new version)   │
│  • deployer status                       (check health)          │
│  • deployer logs --follow                (view logs)             │
│  • deployer rollback                     (if issues)             │
│                                                                   │
│  Duration: 5 minutes per deployment                              │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Platform Team (30 minutes)

### What Platform Team Does

```bash
# 1. Create AWS infrastructure
cd deployer/infrastructure/terraform
terraform apply \
  -var="bucket_name=company-deployer-state" \
  -var="table_name=company-deployer-locks"

# 2. Create deployment repository
mkdir deployment-repo
cd deployment-repo
git init
mkdir -p stacks shared docs

# 3. Share config with teams
cat > docs/BACKEND_CONFIG.md << 'EOF'
Backend Configuration:
- S3 Bucket: company-deployer-state
- DynamoDB Table: company-deployer-locks
- Region: us-east-1
EOF
```

### What Gets Created

```
AWS Resources:
├── S3 Bucket: company-deployer-state
│   └── For storing deployment state
│
├── DynamoDB Table: company-deployer-locks
│   └── For distributed locking
│
└── IAM Role: DeployerExecutionRole
    └── With required permissions

Git Repository:
deployment-repo/
├── stacks/       (teams add their stacks here)
├── shared/       (shared resources)
└── docs/         (documentation)
```

---

## Phase 2: Developer Onboarding (1 hour)

### Step-by-Step

```bash
# 1. Install CLI (1 minute)
curl -sSL https://deployer.io/install.sh | sh
deployer version

# 2. Login with tenant credentials (2 minutes)
# Platform team provides: tenant name + secret
deployer login
? Tenant: notifications-team
? Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
? Bucket: company-deployer-state
? Region: us-east-1
✓ Logged in as: notifications-team

# (Alternative: Single-tenant mode)
# deployer init
# ? S3 Bucket: company-deployer-state
# ? DynamoDB Table: company-deployer-locks
# ? Region: us-east-1
# ✓ Saved to ~/.deployer/config.yaml

# 3. Clone deployment repo (1 minute)
git clone git@github.com:company/deployment-repo.git
cd deployment-repo

# 4. Create your stack (5 minutes)
mkdir -p stacks/notification-platform
cd stacks/notification-platform
deployer stack init

# 5. Define your service (30 minutes)
# Create YAML files for:
# - Service definition
# - API component
# - Database component
# - Queue component

# 6. Build container image (10 minutes)
cd ~/work/your-service/
docker build -t your-api:v1.0.0 .
docker push ECR_REGISTRY/your-api:v1.0.0

# 7. Deploy! (10 minutes)
cd ~/work/deployment-repo/
deployer apply \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.0
```

### Your First Stack

```yaml
stacks/notification-platform/
├── stack.yaml                    # Stack definition
├── services/
│   └── email-service/
│       ├── service.yaml          # Service definition
│       └── components/
│           ├── api/
│           │   ├── microservice.yaml   # What to deploy
│           │   ├── infra.yaml          # Resources/scaling
│           │   └── configs/            # App configs
│           │       └── app.yaml
│           ├── database/
│           │   └── rds.yaml
│           └── queue/
│               └── sqs.yaml
```

---

## Phase 3: Daily Usage (5 minutes)

### Common Commands

```bash
# Deploy new version
deployer apply --stack notification-platform --environment dev --var VERSION=v1.0.1

# Check status
deployer status --stack notification-platform --environment dev

# View logs
deployer logs --component email-service/api --follow

# View metrics
deployer metrics --component email-service/api --since 1h

# Rollback if issues
deployer rollback --stack notification-platform --environment dev

# Promote to production
deployer apply --stack notification-platform --environment production --var VERSION=v1.0.1
```

### Typical Day

```
Morning:
09:00 - Fix bug in code
09:30 - Build v1.0.2: docker build & push
09:35 - Deploy to dev: deployer apply --var VERSION=v1.0.2
09:45 - Test in dev
10:00 - Deploy to staging: deployer apply --environment staging --var VERSION=v1.0.2

Afternoon:
14:00 - Get approval for prod
14:05 - Deploy to prod: deployer apply --environment production --var VERSION=v1.0.2
14:15 - Monitor: deployer metrics & deployer logs
14:30 - ✓ All good!

If Issues:
14:20 - Error rate high!
14:21 - Rollback: deployer rollback --environment production
14:23 - ✓ Back to v1.0.1
```

---

## Real Example: Notifications Team

### Day 0: Setup

```bash
# Alice (team lead) sets up
$ curl -sSL deployer.io/install.sh | sh
$ deployer init
$ git clone git@github.com:company/deployment-repo.git
$ cd deployment-repo
$ mkdir -p stacks/notification-platform
$ deployer stack init
$ # Creates YAML files for email service
$ git add stacks/notification-platform/
$ git commit -m "Add notification platform"
$ git push
```

### Day 1: First Deployment

```bash
# Alice deploys to dev
$ cd deployment-repo
$ deployer apply --stack notification-platform --environment dev --var VERSION=v1.0.0
✓ Deployment successful! (8m 35s)

# Bob tests
$ curl https://dev-email-api.company.com/health
{"status":"healthy"}
```

### Week 1: Iterating

```bash
# Monday - Bob fixes bug
$ docker push ECR/email-api:v1.0.1
$ deployer apply --stack notification-platform --environment dev --var VERSION=v1.0.1

# Tuesday - Alice adds feature
$ docker push ECR/email-api:v1.1.0
$ deployer apply --stack notification-platform --environment dev --var VERSION=v1.1.0

# Wednesday - Deploy to staging
$ deployer apply --stack notification-platform --environment staging --var VERSION=v1.1.0

# Friday - Production!
$ deployer apply --stack notification-platform --environment production --var VERSION=v1.1.0
```

### Week 2: Adding Cache

```bash
# Alice adds Redis cache
$ cat > stacks/notification-platform/services/email-service/components/cache/elasticache.yaml
# (defines cache)

$ # Update API to use cache
$ vim stacks/notification-platform/services/email-service/components/api/microservice.yaml
# (add REDIS_HOST environment variable)

$ git commit -am "Add Redis cache"
$ deployer apply --stack notification-platform --environment dev --var VERSION=v1.1.0
✓ Cache created and API updated
```

### Month 1: Production

```bash
# Team is productive
$ deployer history --stack notification-platform --environment production

Deployments (last 30 days):
v1.5.0  Jan 30  alice@company  Success  4m 32s
v1.4.0  Jan 28  bob@company    Success  3m 18s
v1.3.2  Jan 25  alice@company  Success  4m 05s
v1.3.1  Jan 24  bob@company    Rolled back
v1.3.0  Jan 22  alice@company  Success  5m 12s
...

Total deployments: 15
Success rate: 93%
Average duration: 4m 20s
```

---

## What You Need

### Prerequisites

- ✅ AWS Account
- ✅ AWS CLI configured
- ✅ Docker installed
- ✅ Git access

### What Platform Team Provides

- ✅ S3 bucket name
- ✅ DynamoDB table name
- ✅ AWS region
- ✅ IAM permissions

### What You Provide

- ✅ Your service code
- ✅ Docker image
- ✅ YAML configurations
- ✅ 1 hour for onboarding

---

## Architecture at a Glance

```
┌────────────────────────────────────────────────────────────┐
│ YOUR LAPTOP / CI                                           │
│                                                            │
│  $ deployer apply --stack notification-platform           │
│                                                            │
│  ┌──────────────────────────────────────────────────┐    │
│  │  deployer CLI                                    │    │
│  │  • Parses YAML files                            │    │
│  │  • Connects to AWS                              │    │
│  │  • Manages state & locks                        │    │
│  │  • Deploys via Pulumi                           │    │
│  │  • Exits when done                              │    │
│  └──────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────────┐
│ AWS (Your Account)                                         │
│                                                            │
│  S3: company-deployer-state/                              │
│  └── stacks/notification-platform/production/state.json   │
│                                                            │
│  DynamoDB: company-deployer-locks                         │
│  └── Item: "stack:notification-platform:env:production"   │
│                                                            │
│  Your Resources:                                           │
│  ├── ECS Service (your API)                               │
│  ├── RDS Database                                          │
│  ├── ElastiCache Redis                                     │
│  └── SQS Queue                                             │
└────────────────────────────────────────────────────────────┘
```

---

## Benefits

### For Developers ✅

- **Simple**: Just YAML, no Terraform/Pulumi coding
- **Fast**: 5-minute deployments
- **Safe**: Automatic rollback on failures
- **Consistent**: Same process for all teams
- **Self-service**: Deploy when you want

### For Platform Team ✅

- **No backend**: Just CLI tool
- **Low cost**: ~$3/month (S3 + DynamoDB)
- **Easy maintenance**: Distribute binary updates
- **Standardized**: All teams use same patterns
- **Auditable**: All changes in Git

### For Organization ✅

- **Faster delivery**: 10x more deployments
- **Lower risk**: Automatic rollback
- **Better reliability**: Consistent deployments
- **Cost control**: Track costs per stack
- **Compliance**: All changes audited

---

## Next Steps

### 1. Read Complete Guide

See [GETTING_STARTED_GUIDE.md](docs/GETTING_STARTED_GUIDE.md) for detailed walkthrough.

### 2. Review Examples

```bash
# Look at example stacks
cd deployment-repo/stacks/
ls -la
# notification-platform/
# payment-platform/
# analytics-platform/
```

### 3. Try It

```bash
# Install and configure
deployer init

# Create your first stack
cd deployment-repo
deployer stack init

# Deploy!
deployer apply --stack your-stack --environment dev
```

---

## Getting Help

- **Documentation**: [INDEX.md](INDEX.md)
- **Complete Guide**: [GETTING_STARTED_GUIDE.md](docs/GETTING_STARTED_GUIDE.md)
- **Architecture**: [CLI_ARCHITECTURE.md](docs/CLI_ARCHITECTURE.md)
- **Slack**: #deployer-help
- **Email**: platform-team@company.com

---

## Summary

1. **Platform team** creates S3 bucket + DynamoDB table (30 min, once)
2. **You** install CLI and configure backend (5 min, once)
3. **You** define your service in YAML (30 min, once)
4. **You** deploy with one command (5 min, ongoing)

**That's it!** 🚀

No backend service. No complex setup. Just a CLI tool and YAML files.

---

**Ready to get started? → [GETTING_STARTED_GUIDE.md](docs/GETTING_STARTED_GUIDE.md)**


