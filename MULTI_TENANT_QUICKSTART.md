# Multi-Tenant Deployer - QuickStart

Complete guide to using deployer in multi-tenant mode.

---

## The Big Picture

```
┌─────────────────────────────────────────────────────────────────┐
│                     ONE DEPLOYER CLI                             │
│                                                                  │
│  Two Modes:                                                      │
│  • Admin Mode - Platform team creates/manages tenants           │
│  • Tenant Mode - Dev teams deploy their stacks                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              S3: company-deployer-state                          │
│                                                                  │
│  tenants.yaml    ← Registry of all tenants                      │
│                                                                  │
│  tenants/                                                        │
│  ├── notifications-team/  ← Tenant 1 (isolated)                │
│  ├── payments-team/        ← Tenant 2 (isolated)                │
│  └── analytics-team/       ← Tenant 3 (isolated)                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## For Platform Team (One-Time)

### 1. Deploy Infrastructure (30 minutes)

```bash
# Deploy shared infrastructure
cd deployer/infrastructure/terraform
terraform apply \
  -var="bucket_name=company-deployer-state" \
  -var="table_name=company-deployer-locks"

# Creates:
# ✓ S3 bucket: company-deployer-state
# ✓ DynamoDB table: company-deployer-locks
# ✓ Admin credentials in Secrets Manager
```

### 2. Login as Admin

```bash
# Install CLI
curl -sSL https://deployer.io/install.sh | sh

# Login as admin
$ deployer admin login

? S3 Bucket: company-deployer-state
? Region: us-east-1
? Admin Password: ••••••••••••••••••••••

✓ Admin authentication successful
Mode: ADMIN
```

### 3. Create Tenants

```bash
$ deployer tenant init

? Tenant Name: notifications-team
? Display Name: Notifications Team
? Contact Email: notifications-team@company.com
? Monthly cost limit (USD): 5000

Creating tenant...
✓ Tenant created successfully

────────────────────────────────────────────────
Tenant ID: notifications-team
Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
                ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                SAVE THIS - CANNOT BE RECOVERED

S3 Path: tenants/notifications-team/v1/
────────────────────────────────────────────────
```

### 4. Share Credentials

Share with the Notifications Team:

```
Tenant: notifications-team
Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
Bucket: company-deployer-state
Region: us-east-1

Getting Started: https://docs.deployer.io/getting-started
```

**Do this for each team:**
- Notifications Team: `notifications-team`
- Payments Team: `payments-team`
- Analytics Team: `analytics-team`

---

## For Development Teams (Each Team)

### 1. Install CLI

```bash
curl -sSL https://deployer.io/install.sh | sh
deployer version
```

### 2. Login with Tenant Credentials

```bash
$ deployer login

? Tenant Name: notifications-team
? Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
? S3 Bucket: company-deployer-state
? Region: us-east-1

Authenticating...
✓ Logged in as: notifications-team

Session saved to ~/.deployer/session
```

### 3. Use Deployer Normally

```bash
# Everything scoped to your tenant automatically

# Create stack
cd deployment-repo
deployer stack init

# Deploy
deployer apply --stack notification-platform --environment dev --var VERSION=v1.0.0

# Check status
deployer status --stack notification-platform

# View logs
deployer logs --component email-service/api

# All state goes to:
# s3://company-deployer-state/tenants/notifications-team/v1/stacks/...
```

---

## How It Works

### Admin Commands

```bash
# Admin login
deployer admin login

# Create tenant
deployer tenant init

# List all tenants
deployer tenant list

# Show tenant details
deployer tenant show notifications-team

# Rotate credentials
deployer tenant rotate notifications-team

# Suspend/activate
deployer tenant suspend notifications-team
deployer tenant activate notifications-team

# Delete tenant
deployer tenant delete notifications-team

# Logout
deployer admin logout
```

### Tenant Commands

```bash
# Tenant login
deployer login

# View your tenant details
deployer tenant details

# View usage
deployer tenant usage

# Normal deployer commands
deployer stack init
deployer apply
deployer status
deployer logs
# ... all other commands

# Logout
deployer logout
```

---

## State Isolation

Each tenant gets its own isolated namespace:

```
S3 Structure:

company-deployer-state/
├── tenants.yaml                                    ← Global registry
│
├── tenants/notifications-team/                    ← Tenant 1
│   ├── tenant.yaml
│   └── v1/
│       └── stacks/
│           └── notification-platform/
│               ├── production/state.json
│               └── staging/state.json
│
├── tenants/payments-team/                          ← Tenant 2
│   ├── tenant.yaml
│   └── v1/
│       └── stacks/
│           └── payment-platform/
│               └── production/state.json
│
└── tenants/analytics-team/                         ← Tenant 3
    ├── tenant.yaml
    └── v1/
        └── stacks/...
```

**Tenants cannot access each other's state.**

---

## Lock Isolation

Each tenant's locks are namespaced:

```
DynamoDB Lock Keys:

tenant:notifications-team:stack:notification-platform:env:production
tenant:notifications-team:stack:notification-platform:env:staging
tenant:payments-team:stack:payment-platform:env:production
tenant:analytics-team:stack:analytics-platform:env:production
```

**Tenants cannot interfere with each other's deployments.**

---

## Credential Management

### Format

```
Prefix_32-random-characters

Examples:
ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG  (notifications-team)
pymt_3Hx8kLnM1vQ7tN2jG5sX3aY0bE4cR9fT  (payments-team)
anly_9Zx6mPqL3wS8vM4jK7tY2bN1dF5aH0cG  (analytics-team)
```

### Security

- **Storage**: Bcrypt hash in `tenants.yaml`
- **Never stored**: Plain-text secret
- **Rotation**: Admin can rotate anytime
- **Sharing**: Via secure channel (1Password, Secrets Manager)

### Rotation

```bash
# Admin rotates credentials
$ deployer tenant rotate notifications-team

✓ Credentials Rotated

New Secret: ntfy_2Kx7pMnR4wT9vL3jH8sY5bZ2cF6aS1dG

# Team members see:
$ deployer apply
✗ Authentication failed: Invalid credentials
  Credentials may have been rotated. Contact your admin.

# Team re-authenticates:
$ deployer login
? Tenant: notifications-team
? Secret: ntfy_2Kx7pMnR4wT9vL3jH8sY5bZ2cF6aS1dG
✓ Logged in
```

---

## Monitoring (Admin)

### View All Tenants

```bash
$ deployer tenant list

Tenants
────────────────────────────────────────────────────────────────
ID                  NAME                  STATUS    STACKS  COST
────────────────────────────────────────────────────────────────
notifications-team  Notifications Team    active    3       $342
payments-team       Payments Team         active    5       $1,245
analytics-team      Analytics Team        active    2       $876
────────────────────────────────────────────────────────────────
Total: 3 tenants, 10 stacks, $2,463/month
```

### View Tenant Details

```bash
$ deployer tenant show notifications-team

Tenant: notifications-team
────────────────────────────────────────────────
Name: Notifications Team
Status: Active
Created: 2024-01-15

Stacks: 3
  • notification-platform (production)
  • sms-platform (staging)
  • push-platform (development)

Cost: $342 / $5,000 per month (6.8%)
Deployments: 47 this month
Success Rate: 96%
────────────────────────────────────────────────
```

### Real-Time Monitor

```bash
$ deployer admin monitor

Active Deployments: 2
───────────────────────────────────────────────
notifications-team  notification-platform  prod  Deploying  2m 34s
payments-team       payment-platform       stg   Deploying  1m 12s
```

---

## Cost Tracking

Each tenant has:
- **Cost Limit**: e.g., $5,000/month
- **Cost Alerts**: Email when approaching limit
- **Cost Breakdown**: By service/component

```bash
$ deployer tenant show notifications-team

Cost Breakdown:
  Compute (ECS): $145
  Database (RDS): $128
  Storage (S3): $42
  Network: $27
  ──────────────
  Total: $342 / $5,000 (6.8%)

Trend: ↗ +$45 vs last month
```

---

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

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
          role-to-assume: arn:aws:iam::123456789012:role/GithubActionsDeployer
          aws-region: us-east-1
      
      - name: Login to Deployer
        run: |
          deployer login \
            --tenant notifications-team \
            --secret ${{ secrets.DEPLOYER_TENANT_SECRET }} \
            --bucket company-deployer-state \
            --region us-east-1
      
      - name: Deploy
        run: |
          deployer apply \
            --stack notification-platform \
            --environment production \
            --var VERSION=${{ github.sha }} \
            --auto-approve
```

**Store tenant secret in GitHub Secrets:**
- Go to repo Settings → Secrets → Actions
- Add `DEPLOYER_TENANT_SECRET`: `ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG`

---

## Benefits

### For Platform Team

✅ **Centralized Management**
- One place to manage all teams
- Create tenants in minutes
- Monitor all activity

✅ **Cost Control**
- Track costs per team
- Set limits per tenant
- Get alerts

✅ **Security**
- Isolated state per tenant
- Credential rotation
- Audit trail

✅ **Low Maintenance**
- No backend service
- ~$3/month base cost
- Add tenants without infrastructure changes

### For Development Teams

✅ **Simple**
- Login once with tenant credentials
- Use deployer normally
- No complex configuration

✅ **Isolated**
- Your state is completely separate
- Can't see or affect other teams
- Independent deployments

✅ **Flexible**
- Manage your own stacks
- Deploy anytime
- Self-service

---

## Comparison

### Single-Tenant (Legacy)

```
Each team configures:
- S3 bucket
- DynamoDB table
- AWS credentials

Problems:
- No isolation
- No cost tracking
- Hard to manage at scale
```

### Multi-Tenant (Recommended)

```
Platform team creates:
- One S3 bucket
- One DynamoDB table
- Tenant namespaces

Benefits:
✓ Complete isolation
✓ Cost tracking per team
✓ Centralized management
✓ Same cost as single-tenant
```

---

## FAQs

**Q: Do I need separate AWS accounts per tenant?**
A: No. Tenants are logical isolation within the same AWS account using S3 prefixes and DynamoDB key namespacing.

**Q: Can tenants access each other's data?**
A: No. Credentials are tenant-specific and validated against bcrypt hashes. State is stored in separate S3 prefixes.

**Q: What if I lose tenant credentials?**
A: Admin can rotate: `deployer tenant rotate <tenant-id>`. Generates new credentials.

**Q: Can I have multiple admins?**
A: Yes. Admin password is shared (stored in AWS Secrets Manager). All admins use same password.

**Q: What's the cost?**
A: Same as single-tenant: ~$3/month base (S3 + DynamoDB) + usage.

**Q: Can I migrate from single-tenant to multi-tenant?**
A: Yes. Admin creates tenant, then migrates existing state to tenant prefix.

**Q: How many tenants can I have?**
A: Unlimited. Each tenant is just an S3 prefix.

**Q: Do tenants need separate IAM roles?**
A: Optional. For additional security, you can use IAM policies to restrict S3 access per tenant.

---

## Summary

### Platform Team (One-Time)

1. Deploy infrastructure: `terraform apply`
2. Login as admin: `deployer admin login`
3. Create tenants: `deployer tenant init`
4. Share credentials with teams

### Development Teams (One-Time)

1. Install CLI: `curl -sSL deployer.io/install.sh | sh`
2. Login: `deployer login` (with tenant credentials)
3. Use normally: `deployer apply`, `deployer status`, etc.

### Result

- ✅ Complete isolation per team
- ✅ Centralized management
- ✅ Cost tracking per tenant
- ✅ Simple for everyone
- ✅ Scalable to any number of teams

---

## Next Steps

- **Platform Team**: See [PLATFORM_ADMIN_GUIDE.md](docs/PLATFORM_ADMIN_GUIDE.md)
- **Development Teams**: See [GETTING_STARTED_GUIDE.md](docs/GETTING_STARTED_GUIDE.md)
- **Architecture Details**: See [MULTI_TENANCY.md](docs/MULTI_TENANCY.md)

