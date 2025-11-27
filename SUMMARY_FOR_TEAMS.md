# How Development Teams Start Using Deployer - Summary

This document summarizes exactly how a development team would start using the deployer CLI tool.

---

## The Complete Journey (3 Phases)

### Phase 1: Platform Team Setup (One-Time, 30 minutes)

**Who**: Platform/DevOps team
**When**: Once for the entire organization

```bash
# 1. Create AWS infrastructure
terraform apply

# This creates:
# - S3 bucket: company-deployer-state
# - DynamoDB table: company-deployer-locks

# 2. Share configuration with teams
Email all teams:
  "S3 Bucket: company-deployer-state"
  "DynamoDB Table: company-deployer-locks"
  "Region: us-east-1"
```

**Result**: Shared backend for all teams

---

### Phase 2: Development Team Onboarding (One-Time per team, 1 hour)

**Who**: Each development team (one time)
**When**: When team wants to start deploying

#### Step 1: Install CLI (1 minute)

```bash
curl -sSL https://deployer.io/install.sh | sh
deployer version
```

#### Step 2: Configure (2 minutes)

```bash
deployer init

? S3 Bucket: company-deployer-state
? DynamoDB Table: company-deployer-locks
? Region: us-east-1
✓ Saved to ~/.deployer/config.yaml
```

#### Step 3: Clone Deployment Repo (1 minute)

```bash
git clone git@github.com:company/deployment-repo.git
cd deployment-repo
```

#### Step 4: Create Stack (5 minutes)

```bash
mkdir -p stacks/notification-platform
cd stacks/notification-platform
deployer stack init
```

#### Step 5: Define Service (30 minutes)

Create YAML files:

```
stacks/notification-platform/
├── stack.yaml
└── services/
    └── email-service/
        ├── service.yaml
        └── components/
            ├── api/
            │   ├── microservice.yaml   # What to deploy
            │   ├── infra.yaml          # Resources/scaling
            │   └── configs/
            │       └── app.yaml        # App config
            ├── database/
            │   └── rds.yaml
            └── queue/
                └── sqs.yaml
```

#### Step 6: Build Container (10 minutes)

```bash
cd ~/work/email-service/
docker build -t email-api:v1.0.0 .
docker push ECR_REGISTRY/email-api:v1.0.0
```

#### Step 7: Deploy! (10 minutes)

```bash
cd ~/work/deployment-repo/

deployer apply \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.0

# Output:
# Acquiring lock... ✓
# Creating database... ✓ (5m 23s)
# Creating queue... ✓ (12s)
# Creating API service... ✓ (2m 45s)
# ✓ Deployment successful! (8m 35s)
```

**Result**: Service deployed to AWS

---

### Phase 3: Daily Usage (5 minutes per deployment)

**Who**: Developers
**When**: Ongoing

#### Deploy New Version

```bash
# 1. Build new version
docker build -t email-api:v1.0.1 .
docker push ECR_REGISTRY/email-api:v1.0.1

# 2. Deploy
deployer apply \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.1

# Rolling update, zero downtime
# ✓ Deployment successful! (3m 15s)
```

#### Check Status

```bash
deployer status --stack notification-platform

# Output:
# ✓ api        MicroService    2/2 running    Healthy
# ✓ database   RDS             available      Healthy
# ✓ queue      SQS             active         Healthy
```

#### View Logs

```bash
deployer logs --component email-service/api --follow

# 2024-01-15 17:05:23 INFO Starting email-api v1.0.1
# 2024-01-15 17:05:24 INFO Connected to database
# 2024-01-15 17:05:24 INFO Server listening on :8080
```

#### Promote to Production

```bash
# After testing in dev and staging
deployer apply \
  --stack notification-platform \
  --environment production \
  --var VERSION=v1.0.1

# ⚠ Production deployment - approval required
# Approve? (yes/no): yes
# ✓ Deployment successful! (10m 05s)
```

---

## What You Need

### Prerequisites (You Already Have)
- ✅ AWS Account
- ✅ AWS CLI configured
- ✅ Docker installed
- ✅ Git

### From Platform Team (One Email)
- ✅ S3 bucket name: `company-deployer-state`
- ✅ DynamoDB table name: `company-deployer-locks`
- ✅ AWS region: `us-east-1`

### What You Create
- ✅ YAML files for your service
- ✅ Docker images in ECR
- ✅ Secrets in AWS Secrets Manager

---

## The Architecture (Simple!)

```
YOUR LAPTOP
    │
    │ $ deployer apply --stack my-stack
    │
    ▼
deployer CLI (runs locally)
    │
    │ 1. Reads YAML files from disk
    │ 2. Connects to AWS
    │ 3. Acquires lock in DynamoDB
    │ 4. Loads state from S3
    │ 5. Deploys via Pulumi
    │ 6. Saves state to S3
    │ 7. Releases lock
    │ 8. Exits
    │
    ▼
AWS (Your Account)
    │
    ├── S3: company-deployer-state/
    │   └── stacks/my-stack/dev/state.json
    │
    ├── DynamoDB: company-deployer-locks
    │   └── Lock: "stack:my-stack:env:dev"
    │
    └── Your Resources:
        ├── ECS Service (your API)
        ├── RDS Database
        └── SQS Queue
```

**Key Point**: deployer is just a CLI tool. No backend service to maintain!

---

## Team Collaboration

### Multiple Team Members

**Alice** deploys:
```bash
alice@laptop:~$ deployer apply --stack notification-platform
Acquiring lock... ✓
Deploying...
```

**Bob** tries to deploy at same time:
```bash
bob@laptop:~$ deployer apply --stack notification-platform
⚠ Stack is locked
  Locked by: alice@company.com
  Since: 2 minutes ago
  
Waiting for lock... (Ctrl+C to cancel)
```

**After Alice finishes:**
```bash
# Bob's deployment proceeds automatically
✓ Lock acquired
Deploying...
```

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
- name: Deploy
  run: |
    deployer apply \
      --stack notification-platform \
      --environment production \
      --var VERSION=${{ github.sha }} \
      --auto-approve
```

---

## Benefits

### For Developers
- ✅ Simple YAML configuration
- ✅ 5-minute deployments
- ✅ Automatic rollback on failures
- ✅ Self-service (deploy anytime)
- ✅ Consistent across all teams

### For Platform Team
- ✅ No backend service to maintain
- ✅ Low cost (~$3/month for S3 + DynamoDB)
- ✅ Distribute CLI binary updates easily
- ✅ All changes tracked in Git
- ✅ Standardized deployment patterns

### For Organization
- ✅ 10x more deployments
- ✅ Lower risk (automatic rollback)
- ✅ Better reliability
- ✅ Cost tracking per stack
- ✅ Full audit trail

---

## Day-in-the-Life Example

**Morning** - Bug Fix:
```bash
09:00 - Fix bug in code
09:30 - docker build & push v1.0.2
09:35 - deployer apply --var VERSION=v1.0.2
09:45 - Test in dev ✓
10:00 - deployer apply --environment staging --var VERSION=v1.0.2
```

**Afternoon** - Production:
```bash
14:00 - Get approval
14:05 - deployer apply --environment production --var VERSION=v1.0.2
14:15 - Monitor with deployer logs
14:30 - All good! ✓
```

**If Issues**:
```bash
14:20 - Error rate high! 🚨
14:21 - deployer rollback --environment production
14:23 - Back to v1.0.1 ✓
```

---

## Typical Timeline

### Week 1: Onboarding
- Day 1: Install CLI, configure, create stack
- Day 2: Deploy to dev
- Day 3: Deploy to staging
- Day 4: Deploy to production
- Day 5: Team training

### Week 2: Iterating
- Mon: Deploy v1.0.1 (bug fix)
- Tue: Deploy v1.1.0 (new feature)
- Wed: Add cache component
- Thu: Scale up for load test
- Fri: Production deployment

### Month 2: Productive
- 15 deployments to production
- 96% success rate
- 4-minute average deployment time
- Team is self-sufficient

---

## Common Commands

```bash
# Deploy new version
deployer apply --stack my-stack --environment dev --var VERSION=v1.0.1

# Check status
deployer status --stack my-stack --environment dev

# View logs
deployer logs --component my-service/api --follow

# View deployment history
deployer history --stack my-stack --environment production

# Rollback
deployer rollback --stack my-stack --environment production

# Show current configuration
deployer show --stack my-stack

# Validate YAML
deployer validate --stack my-stack

# Check for drift
deployer drift --stack my-stack --environment production
```

---

## Getting Help

### Documentation
1. **[QUICKSTART.md](QUICKSTART.md)** - 5-minute overview
2. **[HOW_TEAMS_USE_DEPLOYER.md](HOW_TEAMS_USE_DEPLOYER.md)** - Visual walkthrough
3. **[GETTING_STARTED_GUIDE.md](docs/GETTING_STARTED_GUIDE.md)** - Detailed guide

### Support
- **Slack**: #deployer-help
- **Email**: platform-team@company.com
- **Office Hours**: Wednesdays 3-4 PM

---

## FAQs

**Q: Do I need to learn Pulumi?**
A: No. You just write YAML. Deployer handles Pulumi internally.

**Q: Where does deployer run?**
A: Anywhere with AWS access:
- Your laptop
- CI/CD runners
- Bastion hosts

**Q: What if the CLI crashes during deployment?**
A: 
- Lock expires after 1 hour (TTL)
- State is saved incrementally
- You can resume or rollback

**Q: Can I deploy multiple stacks at once?**
A: Yes. Each stack has its own lock.

**Q: How do I share configuration between services?**
A: Use the `shared/` directory in deployment-repo for templates.

---

## Summary

### One-Time Setup (10 minutes)
1. Install: `curl -sSL deployer.io/install.sh | sh`
2. Configure: `deployer init`
3. Done!

### Define Service (30 minutes)
1. Create YAML files
2. Define components
3. Commit to Git

### Daily Deployments (5 minutes)
1. Build Docker image
2. `deployer apply --var VERSION=v1.0.1`
3. Monitor

**That's it!** No complex setup. No backend to maintain. Just YAML and a CLI tool.

---

**Ready to start?** → [QUICKSTART.md](QUICKSTART.md)


