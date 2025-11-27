# Multi-Tenant Architecture

Deployer supports a multi-tenant model where a single CLI tool can be used by multiple isolated teams (tenants), each with their own credentials and isolated state storage.

---

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        DEPLOYER CLI                              │
│                                                                  │
│  Two Modes:                                                      │
│  1. Admin Mode  - Create and manage tenants                     │
│  2. Tenant Mode - Deploy stacks within a tenant                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   S3: company-deployer-state                     │
│                                                                  │
│  tenants.yaml                    ← Admin-managed                │
│                                                                  │
│  tenants/                                                        │
│  ├── notifications-team/         ← Tenant 1                     │
│  │   ├── tenant.yaml                                            │
│  │   ├── v1/                     ← Version namespace           │
│  │   │   └── stacks/                                           │
│  │   │       └── notification-platform/                         │
│  │   │           └── production/                                │
│  │   │               └── state.json                             │
│  │   └── v2/                                                    │
│  │                                                               │
│  ├── payments-team/              ← Tenant 2                     │
│  │   ├── tenant.yaml                                            │
│  │   └── v1/                                                    │
│  │       └── stacks/                                            │
│  │                                                               │
│  └── analytics-team/             ← Tenant 3                     │
│      ├── tenant.yaml                                            │
│      └── v1/                                                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              DynamoDB: company-deployer-locks                    │
│                                                                  │
│  Locks namespaced by tenant:                                    │
│  - tenant:notifications-team:stack:notification-platform:env:prod│
│  - tenant:payments-team:stack:payment-platform:env:prod         │
│  - tenant:analytics-team:stack:analytics-platform:env:prod      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Architecture

### Admin Mode

Platform administrators use admin mode to:
- Create new tenants
- Manage tenant lifecycle
- View all tenants
- Rotate tenant credentials

### Tenant Mode

Development teams use tenant mode to:
- Deploy stacks within their tenant
- Manage their own resources
- View their tenant details
- Cannot access other tenants

---

## Platform Team Workflow

### Step 1: Initial Setup

Platform team creates the shared infrastructure:

```bash
# 1. Create AWS resources
cd deployer/infrastructure/terraform
terraform apply \
  -var="bucket_name=company-deployer-state" \
  -var="table_name=company-deployer-locks"

# Output:
# ✓ Created S3 bucket: company-deployer-state
# ✓ Created DynamoDB table: company-deployer-locks
# ✓ Created tenants.yaml in S3

# 2. Set up admin credentials (one-time)
aws secretsmanager create-secret \
  --name /deployer/admin-credentials \
  --secret-string '{"password":"admin-super-secret-password"}'
```

**Initial `tenants.yaml` structure:**

```yaml
version: v1
metadata:
  created: 2024-01-15T10:00:00Z
  updated: 2024-01-15T10:00:00Z
  bucket: company-deployer-state
  region: us-east-1
  
config:
  lockTable: company-deployer-locks
  defaultVersion: v1
  
tenants: []
```

### Step 2: Admin Login

```bash
# Platform admin logs in
$ deployer admin login

Admin Authentication
────────────────────────────────────────────────

? S3 Bucket: company-deployer-state
? Region: us-east-1
? Admin Password: ••••••••••••••••••••••

Validating credentials...
✓ Admin authentication successful

Session saved to ~/.deployer/admin-session
Mode: ADMIN

Available commands:
  deployer tenant init       - Create new tenant
  deployer tenant list       - List all tenants
  deployer tenant show       - Show tenant details
  deployer tenant rotate     - Rotate tenant credentials
  deployer tenant delete     - Delete tenant
  deployer admin logout      - Logout from admin mode
```

**What happens:**
1. CLI prompts for S3 bucket and admin password
2. Validates admin password against AWS Secrets Manager
3. Creates admin session file: `~/.deployer/admin-session`
4. CLI is now in ADMIN mode

**Admin session file** (`~/.deployer/admin-session`):
```yaml
mode: admin
bucket: company-deployer-state
region: us-east-1
authenticated: 2024-01-15T10:30:00Z
expires: 2024-01-15T18:30:00Z  # 8 hours
```

### Step 3: Create Tenants

```bash
# Create tenant for Notifications Team
$ deployer tenant init

Create New Tenant
────────────────────────────────────────────────

? Tenant Name: notifications-team
? Display Name: Notifications Team
? Contact Email: notifications-team@company.com
? AWS Account ID (for IAM policies): 123456789012
? Enable cost tracking: Yes
? Monthly cost limit (USD): 5000
? State version: v1

Creating tenant...
├── Generating tenant ID... ✓
├── Generating credentials... ✓
├── Creating S3 prefix... ✓
├── Updating tenants.yaml... ✓
└── Tenant created successfully ✓

────────────────────────────────────────────────
✓ Tenant Created

Tenant ID: notifications-team
Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
                ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                SAVE THIS - IT CANNOT BE RECOVERED

S3 Path: s3://company-deployer-state/tenants/notifications-team/v1/
Lock Prefix: tenant:notifications-team:

Share with team:
  Tenant: notifications-team
  Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
  Bucket: company-deployer-state
  Region: us-east-1
────────────────────────────────────────────────

⚠ IMPORTANT: Store the tenant secret securely.
   It cannot be retrieved later. If lost, use 'deployer tenant rotate'.
```

**What happens:**
1. CLI prompts for tenant details
2. Generates unique tenant ID (based on name)
3. Generates random tenant secret (32-character token with prefix `ntfy_`)
4. Creates bcrypt hash of secret
5. Creates tenant directory structure in S3
6. Updates `tenants.yaml` with new tenant
7. Returns credentials to admin

**Updated `tenants.yaml`:**

```yaml
version: v1
metadata:
  created: 2024-01-15T10:00:00Z
  updated: 2024-01-15T11:00:00Z
  bucket: company-deployer-state
  region: us-east-1
  
config:
  lockTable: company-deployer-locks
  defaultVersion: v1
  
tenants:
  - id: notifications-team
    displayName: "Notifications Team"
    email: notifications-team@company.com
    created: 2024-01-15T11:00:00Z
    status: active
    
    credentials:
      hash: $2a$10$rX8vN3jH6tY4bZ1cF5aS0.KpLmQ2wR8vN3jH6tY4bZ1cF5aS0dGxY
      algorithm: bcrypt
      rotations: 0
      lastRotated: null
    
    storage:
      prefix: tenants/notifications-team
      version: v1
      path: tenants/notifications-team/v1
    
    locks:
      prefix: tenant:notifications-team
    
    aws:
      accountId: "123456789012"
      region: us-east-1
    
    limits:
      costTracking: true
      monthlyCostLimit: 5000
      maxStacks: 100
      maxServices: 500
    
    metadata:
      team: notifications
      department: engineering
    
  - id: payments-team
    displayName: "Payments Team"
    # ... similar structure
```

**Tenant directory structure in S3:**

```
s3://company-deployer-state/
├── tenants.yaml
└── tenants/
    └── notifications-team/
        ├── tenant.yaml                    ← Tenant-specific config
        └── v1/                            ← Version namespace
            └── stacks/                    ← (empty, ready for stacks)
```

**`tenant.yaml`** (created automatically):

```yaml
tenant:
  id: notifications-team
  displayName: "Notifications Team"
  version: v1
  created: 2024-01-15T11:00:00Z

storage:
  bucket: company-deployer-state
  prefix: tenants/notifications-team/v1
  
locks:
  table: company-deployer-locks
  prefix: tenant:notifications-team

config:
  environments:
    - production
    - staging
    - development
  
  regions:
    - us-east-1
    - us-west-2
  
  policies:
    requireApprovalForProduction: true
    enableDriftDetection: true
    enableCostTracking: true
```

### Step 4: Share Credentials with Team

```bash
# Platform admin shares credentials via secure channel
# (Slack DM, 1Password, AWS Secrets Manager, etc.)

📧 Message to Notifications Team:

Subject: Deployer Access - Notifications Team

Your deployer tenant has been created!

Tenant: notifications-team
Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
Bucket: company-deployer-state
Region: us-east-1

Getting Started:
1. Install CLI: curl -sSL https://deployer.io/install.sh | sh
2. Login: deployer login
3. See guide: https://docs.deployer.io/getting-started

⚠ Keep the secret secure. Do not commit to Git.
```

---

## Development Team Workflow

### Step 1: Install CLI

```bash
# Team member installs CLI
curl -sSL https://deployer.io/install.sh | sh
deployer version
```

### Step 2: Tenant Login

```bash
$ deployer login

Tenant Authentication
────────────────────────────────────────────────

? Tenant Name: notifications-team
? Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
? S3 Bucket: company-deployer-state
? Region: us-east-1

Authenticating...
├── Loading tenants.yaml... ✓
├── Finding tenant... ✓
├── Verifying credentials... ✓
├── Loading tenant configuration... ✓
└── Authentication successful ✓

────────────────────────────────────────────────
✓ Logged in as: notifications-team

Tenant: Notifications Team
Email: notifications-team@company.com
S3 Path: tenants/notifications-team/v1/
Version: v1

Session saved to ~/.deployer/session

Available commands:
  deployer stack init        - Create new stack
  deployer apply            - Deploy stack
  deployer status           - Check stack status
  deployer tenant details   - View tenant details
  deployer logout           - Logout
────────────────────────────────────────────────
```

**What happens:**
1. CLI prompts for tenant name and secret
2. Downloads `tenants.yaml` from S3
3. Finds tenant by name
4. Verifies secret against bcrypt hash
5. Loads tenant configuration from S3
6. Creates tenant session file: `~/.deployer/session`
7. CLI is now in TENANT mode

**Tenant session file** (`~/.deployer/session`):

```yaml
mode: tenant
tenant:
  id: notifications-team
  displayName: "Notifications Team"
  version: v1

backend:
  type: s3
  bucket: company-deployer-state
  region: us-east-1
  prefix: tenants/notifications-team/v1

locks:
  type: dynamodb
  table: company-deployer-locks
  region: us-east-1
  prefix: tenant:notifications-team

aws:
  profile: default
  region: us-east-1
  accountId: "123456789012"

authenticated: 2024-01-15T12:00:00Z
expires: 2024-01-22T12:00:00Z  # 7 days
```

### Step 3: Use Deployer Normally

```bash
# All deployer commands now scoped to this tenant

# View tenant details
$ deployer tenant details

Tenant Details
────────────────────────────────────────────────
Name: Notifications Team (notifications-team)
Email: notifications-team@company.com
Created: 2024-01-15
Status: Active

Storage:
  Bucket: company-deployer-state
  Prefix: tenants/notifications-team/v1/
  Version: v1

Limits:
  Monthly Cost Limit: $5,000
  Max Stacks: 100
  Max Services: 500

Usage (current month):
  Stacks: 3 / 100
  Services: 12 / 500
  Estimated Cost: $342 / $5,000

Stacks:
  • notification-platform (3 services, production)
  • sms-platform (2 services, staging)
  • push-platform (7 services, development)
────────────────────────────────────────────────

# Create and deploy stacks (same as before)
$ cd deployment-repo
$ deployer stack init
$ deployer apply --stack notification-platform --environment dev
```

**All state is isolated:**

```
S3 Structure for notifications-team:
s3://company-deployer-state/tenants/notifications-team/v1/
└── stacks/
    ├── notification-platform/
    │   ├── production/
    │   │   └── state.json
    │   ├── staging/
    │   │   └── state.json
    │   └── development/
    │       └── state.json
    ├── sms-platform/
    └── push-platform/

DynamoDB Locks for notifications-team:
- tenant:notifications-team:stack:notification-platform:env:production
- tenant:notifications-team:stack:notification-platform:env:staging
- tenant:notifications-team:stack:sms-platform:env:staging
```

---

## CLI Command Reference

### Admin Mode Commands

Only available after `deployer admin login`:

```bash
# Tenant Management
deployer tenant init                          # Create new tenant
deployer tenant list                          # List all tenants
deployer tenant show <tenant-id>              # Show tenant details
deployer tenant rotate <tenant-id>            # Rotate tenant credentials
deployer tenant suspend <tenant-id>           # Suspend tenant
deployer tenant activate <tenant-id>          # Activate tenant
deployer tenant delete <tenant-id>            # Delete tenant
deployer tenant stats                         # Show tenant statistics

# Session Management
deployer admin logout                         # Logout from admin mode
deployer admin session                        # Show current session
```

### Tenant Mode Commands

Available after `deployer login`:

```bash
# Tenant Operations
deployer tenant details                       # View tenant details
deployer tenant usage                         # View usage statistics
deployer tenant stacks                        # List all stacks

# Stack Operations (normal deployer commands)
deployer stack init                           # Create stack
deployer apply                                # Deploy
deployer status                               # Check status
deployer logs                                 # View logs
# ... all other deployer commands

# Session Management
deployer logout                               # Logout from tenant
deployer session                              # Show current session
```

### Public Commands

Available without login:

```bash
deployer version                              # Show version
deployer help                                 # Show help
deployer admin login                          # Login as admin
deployer login                                # Login as tenant
```

---

## Command Examples

### Admin: Create Multiple Tenants

```bash
# Login as admin
$ deployer admin login

# Create tenants
$ deployer tenant init
? Tenant Name: notifications-team
✓ Tenant: notifications-team
  Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG

$ deployer tenant init
? Tenant Name: payments-team
✓ Tenant: payments-team
  Secret: pymt_3Hx8kLnM1vQ7tN2jG5sX3aY0bE4cR9fT

$ deployer tenant init
? Tenant Name: analytics-team
✓ Tenant: analytics-team
  Secret: anly_9Zx6mPqL3wS8vM4jK7tY2bN1dF5aH0cG

# List tenants
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

# Show tenant details
$ deployer tenant show notifications-team

Tenant: notifications-team
────────────────────────────────────────────────
Display Name: Notifications Team
Email: notifications-team@company.com
Status: Active
Created: 2024-01-15

Storage:
  Path: tenants/notifications-team/v1/
  Bucket: company-deployer-state

Credentials:
  Last Rotated: Never
  Rotations: 0

Usage:
  Stacks: 3 / 100
  Services: 12 / 500
  Cost: $342 / $5,000

Stacks:
  • notification-platform (production, 3 services)
  • sms-platform (staging, 2 services)
  • push-platform (development, 7 services)
────────────────────────────────────────────────
```

### Admin: Rotate Credentials

```bash
$ deployer tenant rotate notifications-team

Rotate Tenant Credentials
────────────────────────────────────────────────

Tenant: notifications-team (Notifications Team)
Current rotations: 0
Last rotated: Never

⚠ This will invalidate the current tenant secret.
  All team members will need to re-authenticate.

Continue? (yes/no): yes

Rotating credentials...
├── Generating new secret... ✓
├── Updating tenants.yaml... ✓
├── Invalidating old sessions... ✓
└── Credentials rotated successfully ✓

────────────────────────────────────────────────
✓ Credentials Rotated

New Tenant Secret: ntfy_2Kx7pMnR4wT9vL3jH8sY5bZ2cF6aS1dG
                    ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                    SHARE WITH TEAM SECURELY

Rotations: 1
Rotated at: 2024-01-20T14:30:00Z
────────────────────────────────────────────────

# Team members will see:
$ deployer apply
✗ Authentication failed: Invalid credentials
  Credentials may have been rotated. Contact your admin.
```

### Tenant: View Details

```bash
$ deployer login
? Tenant: notifications-team
? Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
✓ Logged in

$ deployer tenant details

Tenant Details
────────────────────────────────────────────────
Name: Notifications Team
ID: notifications-team
Email: notifications-team@company.com
Status: Active

Your Stacks: 3
  • notification-platform (production)
  • sms-platform (staging)
  • push-platform (development)

This Month:
  Deployments: 47
  Cost: $342 / $5,000
  Uptime: 99.8%
────────────────────────────────────────────────

$ deployer tenant usage

Usage Statistics
────────────────────────────────────────────────
Period: January 2024

Resources:
  Stacks: 3 / 100
  Services: 12 / 500
  Components: 45

Compute:
  ECS Tasks: 28
  Fargate vCPU: 12.5
  Memory: 32 GB

Storage:
  RDS: 3 instances (200 GB)
  S3: 8 buckets (1.2 TB)
  ElastiCache: 2 clusters

Cost Breakdown:
  Compute: $145
  Database: $128
  Storage: $42
  Networking: $27
  ──────────────
  Total: $342 / $5,000 (6.8%)

Trend: ↗ +$45 vs last month
────────────────────────────────────────────────
```

---

## Credential Management

### Credential Format

**Tenant Secret Format:**
```
<prefix>_<random-32-chars>

Examples:
ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG  (notifications-team)
pymt_3Hx8kLnM1vQ7tN2jG5sX3aY0bE4cR9fT  (payments-team)
anly_9Zx6mPqL3wS8vM4jK7tY2bN1dF5aH0cG  (analytics-team)
```

**Prefix by team:**
- `ntfy_` - Notifications
- `pymt_` - Payments
- `anly_` - Analytics
- (or use generic `dplr_` for all)

### Storage

**In `tenants.yaml` (S3):**
```yaml
credentials:
  hash: $2a$10$rX8vN3jH6tY4bZ1cF5aS0.KpLmQ2wR8vN3jH6tY4bZ1cF5aS0dGxY
  algorithm: bcrypt
  cost: 10
```

**Never stored:**
- Plain-text secret
- Reversible encryption

### Verification Flow

```
User Input
    │
    │ ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
    │
    ▼
bcrypt.CompareHashAndPassword(hash, secret)
    │
    ├─ Match ✓    → Create session
    └─ No Match ✗ → Reject
```

### Security Best Practices

1. **Admin Credentials**:
   - Store in AWS Secrets Manager
   - Rotate every 90 days
   - Require MFA for production

2. **Tenant Credentials**:
   - Share via secure channel (1Password, Secrets Manager)
   - Rotate on team member departure
   - Never commit to Git
   - Use environment variables in CI/CD

3. **Session Management**:
   - Admin sessions: 8-hour expiry
   - Tenant sessions: 7-day expiry
   - Automatic renewal on command execution
   - Logout on inactivity

---

## Multi-Tenant State Isolation

### S3 Structure

```
s3://company-deployer-state/
│
├── tenants.yaml                               ← Global tenant registry
│
└── tenants/
    ├── notifications-team/
    │   ├── tenant.yaml                        ← Tenant config
    │   ├── v1/                                ← Version 1
    │   │   └── stacks/
    │   │       └── notification-platform/
    │   │           ├── production/
    │   │           │   ├── state.json
    │   │           │   └── history/
    │   │           │       ├── 001.json
    │   │           │       └── 002.json
    │   │           └── staging/
    │   │               └── state.json
    │   └── v2/                                ← Future version migration
    │
    ├── payments-team/
    │   ├── tenant.yaml
    │   └── v1/
    │       └── stacks/
    │
    └── analytics-team/
        ├── tenant.yaml
        └── v1/
            └── stacks/
```

### DynamoDB Lock Keys

```
Format: tenant:<tenant-id>:stack:<stack-name>:env:<environment>

Examples:
tenant:notifications-team:stack:notification-platform:env:production
tenant:notifications-team:stack:notification-platform:env:staging
tenant:payments-team:stack:payment-platform:env:production
tenant:analytics-team:stack:analytics-platform:env:production
```

**Lock attributes:**
```json
{
  "LockID": "tenant:notifications-team:stack:notification-platform:env:production",
  "TenantID": "notifications-team",
  "StackName": "notification-platform",
  "Environment": "production",
  "LockedBy": "alice@company.com",
  "LockedAt": "2024-01-15T14:30:00Z",
  "LockExpiry": 1705329000,
  "Heartbeat": 1705325400,
  "Version": "v1.0.0",
  "TTL": 1705332600
}
```

### IAM Policies (Optional Enhancement)

For additional security, use IAM policies to restrict S3 access:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::company-deployer-state/tenants/notifications-team/*"
      ],
      "Condition": {
        "StringEquals": {
          "aws:PrincipalTag/tenant": "notifications-team"
        }
      }
    }
  ]
}
```

---

## Tenant Lifecycle

### 1. Creation

```bash
deployer tenant init
```

- Generate tenant ID
- Generate credentials
- Create S3 prefix
- Create initial `tenant.yaml`
- Update `tenants.yaml`

### 2. Active Use

- Team logs in with credentials
- Deploys stacks within tenant namespace
- All state isolated to tenant prefix

### 3. Suspension

```bash
deployer tenant suspend notifications-team
```

- Mark tenant as `suspended` in `tenants.yaml`
- Prevent new logins (existing sessions valid until expiry)
- State preserved
- Can be reactivated

### 4. Deletion

```bash
deployer tenant delete notifications-team --confirm
```

- Archive state to separate location
- Remove from `tenants.yaml`
- Invalidate all sessions
- Optional: Delete S3 prefix after grace period

---

## Migration Path

### For Existing Single-Tenant Deployments

If you already have teams using deployer without multi-tenancy:

```bash
# 1. Admin creates tenant for existing team
$ deployer admin login
$ deployer tenant init
? Tenant Name: existing-team
✓ Created: existing-team

# 2. Migrate existing state
$ aws s3 sync \
  s3://company-deployer-state/stacks/ \
  s3://company-deployer-state/tenants/existing-team/v1/stacks/

# 3. Update tenants.yaml to mark migration complete

# 4. Team logs in with new credentials
$ deployer login
? Tenant: existing-team
? Secret: (provided by admin)
✓ Logged in

# 5. Verify state
$ deployer status --stack my-stack
✓ All resources healthy
```

---

## Configuration Examples

### Platform Admin: Setting Up Multi-Tenancy

**Terraform** (`infrastructure/terraform/main.tf`):

```hcl
resource "aws_s3_bucket" "deployer_state" {
  bucket = "company-deployer-state"
  
  versioning {
    enabled = true
  }
  
  tags = {
    Purpose = "Deployer Multi-Tenant State"
  }
}

resource "aws_s3_object" "tenants_yaml" {
  bucket  = aws_s3_bucket.deployer_state.id
  key     = "tenants.yaml"
  content = templatefile("${path.module}/templates/tenants.yaml.tpl", {
    bucket = aws_s3_bucket.deployer_state.id
    region = var.region
    table  = aws_dynamodb_table.deployer_locks.name
  })
}

resource "aws_dynamodb_table" "deployer_locks" {
  name         = "company-deployer-locks"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"
  
  attribute {
    name = "LockID"
    type = "S"
  }
  
  attribute {
    name = "TenantID"
    type = "S"
  }
  
  ttl {
    attribute_name = "TTL"
    enabled        = true
  }
  
  global_secondary_index {
    name            = "TenantIndex"
    hash_key        = "TenantID"
    projection_type = "ALL"
  }
}

resource "aws_secretsmanager_secret" "admin_credentials" {
  name = "/deployer/admin-credentials"
}
```

---

## Benefits of Multi-Tenancy

### For Platform Teams

- ✅ **Centralized Management**: Manage all teams from one place
- ✅ **Cost Tracking**: Track costs per team
- ✅ **Access Control**: Grant/revoke access easily
- ✅ **Compliance**: Audit trail per tenant
- ✅ **Scalability**: Add teams without infrastructure changes

### For Development Teams

- ✅ **Isolation**: Your state is completely separate
- ✅ **Security**: Credentials are tenant-specific
- ✅ **Simplicity**: Same CLI, no extra setup
- ✅ **Flexibility**: Manage your own stacks independently

### For Organization

- ✅ **Cost Efficiency**: Single S3 bucket, shared DynamoDB
- ✅ **Standardization**: All teams use same tool
- ✅ **Visibility**: Platform team sees all tenants
- ✅ **Security**: Credential rotation, audit logs

---

## Summary

Multi-tenancy in deployer provides:

1. **Two Modes**:
   - Admin: Create/manage tenants
   - Tenant: Deploy within tenant namespace

2. **Isolation**:
   - Separate S3 prefixes per tenant
   - Namespaced DynamoDB locks
   - Bcrypt-hashed credentials

3. **Simple Workflow**:
   - Admin: `deployer admin login` → `deployer tenant init`
   - Team: `deployer login` → use normally

4. **Scalability**:
   - Add unlimited tenants
   - No infrastructure changes needed
   - Low cost (~$3/month base + usage)

**Next Steps**:
- Platform team: See [PLATFORM_ADMIN_GUIDE.md](PLATFORM_ADMIN_GUIDE.md)
- Development teams: See [GETTING_STARTED_GUIDE.md](GETTING_STARTED_GUIDE.md)

