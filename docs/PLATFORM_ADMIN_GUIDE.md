# Platform Admin Guide

Complete guide for platform administrators managing panka multi-tenant infrastructure.

---

## Overview

As a platform administrator, you will:
1. Set up the shared panka infrastructure (one-time)
2. Create tenants for development teams
3. Manage tenant lifecycle
4. Monitor usage and costs
5. Handle credential rotation

---

## Initial Setup (One-Time)

### Prerequisites

- AWS Account with admin access
- AWS CLI configured
- Terraform installed
- Admin password decided (secure, 32+ characters)

### Step 1: Deploy Infrastructure

```bash
# Clone panka repository
git clone https://github.com/company/panka.git
cd panka/infrastructure/terraform

# Review and customize variables
cat > terraform.tfvars << EOF
bucket_name = "company-panka-state"
table_name  = "company-panka-locks"
region      = "us-east-1"
aws_account_id = "123456789012"

admin_password = "your-super-secure-admin-password-here"

# Optional: Cost alerts
cost_alert_email = "platform-team@company.com"
monthly_cost_threshold = 10000

# Optional: Backup settings
enable_s3_replication = true
backup_region = "us-west-2"
EOF

# Initialize and apply
terraform init
terraform plan
terraform apply

# Output will show:
# ✓ S3 bucket created: company-panka-state
# ✓ DynamoDB table created: company-panka-locks
# ✓ Admin credentials stored in Secrets Manager
# ✓ IAM roles created
# ✓ CloudWatch alarms configured
```

**What gets created:**

```
AWS Resources:
├── S3 Bucket: company-panka-state
│   ├── Versioning: enabled
│   ├── Encryption: AES-256
│   ├── Lifecycle: archive old versions after 90 days
│   └── Object: tenants.yaml (initial)
│
├── DynamoDB Table: company-panka-locks
│   ├── Billing: PAY_PER_REQUEST
│   ├── TTL: enabled on TTL attribute
│   └── GSI: TenantIndex (for querying by tenant)
│
├── Secrets Manager:
│   └── /panka/admin-credentials
│
├── IAM Roles:
│   ├── PankaAdminRole
│   └── PankaTenantRole (template)
│
└── CloudWatch:
    ├── Alarms: High cost, high error rate
    └── Dashboards: Usage metrics
```

### Step 2: Verify Setup

```bash
# Check S3 bucket
aws s3 ls s3://company-panka-state/
# Output: tenants.yaml

# Check DynamoDB table
aws dynamodb describe-table --table-name company-panka-locks
# Output: Table details with TenantIndex GSI

# Check admin credentials
aws secretsmanager describe-secret --secret-id /panka/admin-credentials
# Output: Secret metadata (not the value)
```

### Step 3: Install Panka CLI

```bash
# Install CLI for admin use
curl -sSL https://panka.io/install.sh | sh

# Verify installation
panka version
# Output: panka version 1.0.0

# Move to system path
sudo mv panka /usr/local/bin/
```

### Step 4: First Admin Login

```bash
$ panka admin login

Admin Authentication
────────────────────────────────────────────────

This is your first login. Admin credentials are stored in:
AWS Secrets Manager: /panka/admin-credentials

? S3 Bucket: company-panka-state
? AWS Region: us-east-1
? Admin Password: ••••••••••••••••••••••••••••••

Authenticating...
├── Connecting to AWS... ✓
├── Loading admin credentials from Secrets Manager... ✓
├── Verifying password... ✓
├── Loading tenants.yaml from S3... ✓
└── Admin authentication successful ✓

────────────────────────────────────────────────
✓ Logged in as Administrator

Bucket: company-panka-state
Region: us-east-1
Tenants: 0

Session saved to ~/.panka/admin-session
Session expires in 8 hours

Admin Commands:
  panka tenant init      - Create new tenant
  panka tenant list      - List all tenants
  panka tenant show      - Show tenant details
  panka tenant stats     - Usage statistics
  panka admin logout     - Logout
────────────────────────────────────────────────
```

---

## Creating Tenants

### Create First Tenant

```bash
$ panka tenant init

Create New Tenant
────────────────────────────────────────────────

? Tenant Name (lowercase, alphanumeric, hyphens): notifications-team
? Display Name: Notifications Team
? Contact Email: notifications-team@company.com
? Department: Engineering
? AWS Account ID: 123456789012
? State Version: v1

Cost Controls:
? Enable cost tracking: Yes
? Monthly cost limit (USD, 0 for unlimited): 5000
? Enable cost alerts: Yes
? Alert email: notifications-team@company.com

Limits:
? Max stacks: 100
? Max services per stack: 50

Policies:
? Require approval for production deployments: Yes
? Enable drift detection: Yes
? Enable audit logging: Yes

Confirm creation? (yes/no): yes

Creating tenant...
├── Validating tenant name... ✓
├── Checking for conflicts... ✓
├── Generating tenant ID... ✓
├── Generating secure credentials... ✓
│   Format: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
│   Hashing with bcrypt (cost 10)... ✓
├── Creating S3 directory structure... ✓
│   Created: tenants/notifications-team/
│   Created: tenants/notifications-team/v1/
│   Created: tenants/notifications-team/v1/stacks/
├── Creating tenant.yaml... ✓
├── Updating tenants.yaml... ✓
├── Setting up IAM policies (optional)... ✓
├── Creating CloudWatch dashboard... ✓
└── Tenant created successfully ✓

────────────────────────────────────────────────
✓ Tenant Created Successfully

Tenant ID: notifications-team
Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
                ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                ⚠ SAVE THIS - CANNOT BE RECOVERED

S3 Path: s3://company-panka-state/tenants/notifications-team/v1/
Lock Prefix: tenant:notifications-team:

Credentials to share with team:
────────────────────────────────────────────────
Tenant: notifications-team
Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
Bucket: company-panka-state
Region: us-east-1

Getting Started: https://docs.panka.io/getting-started
────────────────────────────────────────────────

⚠ IMPORTANT:
1. Save the tenant secret securely (1Password, Vault, etc.)
2. Share credentials via secure channel (NOT email/Slack)
3. The secret cannot be retrieved later
4. Use 'panka tenant rotate' if credentials are lost
────────────────────────────────────────────────

Next: Share credentials with the Notifications Team
```

### Batch Create Tenants

Create a YAML file with multiple tenants:

**`tenants-to-create.yaml`:**

```yaml
tenants:
  - name: notifications-team
    displayName: Notifications Team
    email: notifications-team@company.com
    department: Engineering
    costLimit: 5000
    
  - name: payments-team
    displayName: Payments Team
    email: payments-team@company.com
    department: Engineering
    costLimit: 8000
    
  - name: analytics-team
    displayName: Analytics Team
    email: analytics-team@company.com
    department: Data
    costLimit: 12000
```

```bash
$ panka tenant init --batch tenants-to-create.yaml

Batch Tenant Creation
────────────────────────────────────────────────

Loading tenants from: tenants-to-create.yaml
Found: 3 tenants

Creating tenants...

[1/3] Creating notifications-team...
✓ Created
  Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG

[2/3] Creating payments-team...
✓ Created
  Secret: pymt_3Hx8kLnM1vQ7tN2jG5sX3aY0bE4cR9fT

[3/3] Creating analytics-team...
✓ Created
  Secret: anly_9Zx6mPqL3wS8vM4jK7tY2bN1dF5aH0cG

────────────────────────────────────────────────
✓ Created 3 tenants

Credentials saved to: tenant-credentials.txt
⚠ Store securely and distribute to teams
────────────────────────────────────────────────
```

**`tenant-credentials.txt`:**

```
Panka Tenant Credentials
Generated: 2024-01-15 10:30:00
────────────────────────────────────────────────

Tenant: notifications-team
Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
Contact: notifications-team@company.com

Tenant: payments-team
Secret: pymt_3Hx8kLnM1vQ7tN2jG5sX3aY0bE4cR9fT
Contact: payments-team@company.com

Tenant: analytics-team
Secret: anly_9Zx6mPqL3wS8vM4jK7tY2bN1dF5aH0cG
Contact: analytics-team@company.com

────────────────────────────────────────────────
Backend: company-panka-state (us-east-1)
Getting Started: https://docs.panka.io/getting-started
```

---

## Managing Tenants

### List All Tenants

```bash
$ panka tenant list

Tenants
────────────────────────────────────────────────────────────────────────────
ID                  NAME                    STATUS    STACKS  COST/MONTH  LIMIT
────────────────────────────────────────────────────────────────────────────
notifications-team  Notifications Team      active    3       $342        $5,000
payments-team       Payments Team           active    5       $1,245      $8,000
analytics-team      Analytics Team          active    2       $876        $12,000
marketing-team      Marketing Team          suspended 1       $45         $3,000
────────────────────────────────────────────────────────────────────────────
Total: 4 tenants (3 active, 1 suspended)
Total Stacks: 11
Total Cost: $2,508/month
────────────────────────────────────────────────────────────────────────────

Filters:
  --status active|suspended|all    (default: active)
  --sort name|cost|stacks          (default: name)
  --department <name>
```

### Show Tenant Details

```bash
$ panka tenant show notifications-team

Tenant: notifications-team
────────────────────────────────────────────────────────────────
Display Name: Notifications Team
Email: notifications-team@company.com
Department: Engineering
Status: Active
Created: 2024-01-15 10:30:00

Contact:
  Email: notifications-team@company.com
  Slack: #notifications-team
  Owner: Alice Smith (alice@company.com)

Storage:
  Bucket: company-panka-state
  Prefix: tenants/notifications-team/v1/
  Version: v1
  Size: 1.2 GB

Credentials:
  Last Rotated: 2024-01-10 (5 days ago)
  Rotations: 1
  Active Sessions: 3
  Last Login: 2024-01-15 08:45:00 (alice@company.com)

Limits:
  Cost: $342 / $5,000 per month (6.8%)
  Stacks: 3 / 100 (3%)
  Services: 12 / 500 (2.4%)

Policies:
  ✓ Production approval required
  ✓ Drift detection enabled
  ✓ Audit logging enabled
  ✓ Cost alerts enabled

Usage (Current Month):
  Deployments: 47
  Failed Deployments: 2 (4.3%)
  Average Duration: 4m 23s
  Total Lock Time: 3h 42m

Cost Breakdown:
  Compute (ECS): $145
  Database (RDS): $128
  Storage (S3): $42
  Network: $27
  ───────────────
  Total: $342

Stacks:
  1. notification-platform (production)
     - 3 services, 12 components
     - Last deployed: 2h ago (v1.3.0)
     - Cost: $245/month
     
  2. sms-platform (staging)
     - 2 services, 8 components
     - Last deployed: 1d ago (v1.1.5)
     - Cost: $67/month
     
  3. push-platform (development)
     - 1 service, 4 components
     - Last deployed: 3d ago (v0.9.0)
     - Cost: $30/month

Recent Activity:
  2024-01-15 08:45 - alice@company.com deployed notification-platform v1.3.0
  2024-01-14 15:20 - bob@company.com deployed sms-platform v1.1.5
  2024-01-13 10:10 - carol@company.com updated push-platform config
────────────────────────────────────────────────────────────────
```

### View Tenant Statistics

```bash
$ panka tenant stats

Tenant Statistics
────────────────────────────────────────────────────────────────
Period: Last 30 days

Overview:
  Total Tenants: 4
  Active Tenants: 3
  Suspended Tenants: 1
  
  Total Stacks: 11
  Total Services: 45
  Total Components: 178

Deployments:
  Total: 342
  Successful: 329 (96.2%)
  Failed: 13 (3.8%)
  Average Duration: 4m 12s

Resource Usage:
  ECS Tasks: 87
  RDS Instances: 12
  ElastiCache Clusters: 8
  S3 Buckets: 34
  Lambda Functions: 23

Cost:
  Total: $2,508/month
  Average per Tenant: $627/month
  Trend: ↗ +12% vs last month

Top Tenants by Cost:
  1. analytics-team: $876 (34.9%)
  2. payments-team: $1,245 (49.6%)
  3. notifications-team: $342 (13.6%)

Top Tenants by Deployments:
  1. notifications-team: 47
  2. payments-team: 38
  3. analytics-team: 15

Storage:
  Total: 15.7 GB
  State Files: 2.3 GB
  History: 13.4 GB

Locks:
  Current Active: 2
  Total This Month: 342
  Average Hold Time: 6m 32s
  Max Hold Time: 45m 12s (payments-team)
────────────────────────────────────────────────────────────────

Export options:
  --format json|yaml|csv
  --output <file>
```

---

## Credential Management

### Rotate Tenant Credentials

```bash
$ panka tenant rotate notifications-team

Rotate Tenant Credentials
────────────────────────────────────────────────

Tenant: notifications-team (Notifications Team)
Current Credentials:
  Last Rotated: 2024-01-10 (5 days ago)
  Rotations: 1
  Active Sessions: 3

⚠ Warning:
  - This will invalidate the current tenant secret
  - All team members (3 active sessions) will be logged out
  - They will need to re-authenticate with the new secret
  - Active deployments will continue but new ones will fail

Reason for rotation (optional): Scheduled rotation

Continue? (yes/no): yes

Rotating credentials...
├── Generating new secret... ✓
├── Hashing with bcrypt... ✓
├── Updating tenants.yaml... ✓
├── Invalidating active sessions (3)... ✓
├── Notifying team via email... ✓
└── Credentials rotated successfully ✓

────────────────────────────────────────────────
✓ Credentials Rotated

New Tenant Secret: ntfy_2Kx7pMnR4wT9vL3jH8sY5bZ2cF6aS1dG
                    ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Previous Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
                 (Invalidated)

Rotations: 2
Rotated: 2024-01-15 14:30:00
Rotated By: admin@company.com
Reason: Scheduled rotation

────────────────────────────────────────────────

Next Steps:
1. Save new secret securely
2. Share with team via secure channel
3. Team re-authenticates: panka login
   
Email sent to: notifications-team@company.com
────────────────────────────────────────────────
```

### Automated Rotation Policy

Set up automatic credential rotation:

**`.panka/admin-config.yaml`:**

```yaml
admin:
  credentials:
    rotationPolicy:
      enabled: true
      interval: 90  # days
      notifyBefore: 7  # days
      
tenants:
  credentials:
    rotationPolicy:
      enabled: true
      interval: 180  # days
      notifyBefore: 14  # days
      autoRotate: false  # require manual approval
```

```bash
$ panka admin config set-rotation-policy

Credential Rotation Policy
────────────────────────────────────────────────

Admin Credentials:
  Auto-rotation: Every 90 days
  Notify: 7 days before
  Next rotation: 2024-04-15

Tenant Credentials:
  Auto-rotation: Every 180 days (with approval)
  Notify: 14 days before
  
Tenants pending rotation (within 14 days):
  • notifications-team (7 days)
  • marketing-team (12 days)

✓ Policy configured
```

---

## Tenant Lifecycle Management

### Suspend Tenant

```bash
$ panka tenant suspend notifications-team

Suspend Tenant
────────────────────────────────────────────────

Tenant: notifications-team (Notifications Team)
Status: Active
Stacks: 3
Active Deployments: 0

Reason for suspension: Team reorganization

⚠ Effect:
  - New logins will be blocked
  - Active sessions remain valid until expiry
  - Deployments from active sessions will continue
  - State is preserved
  - Can be reactivated later

Continue? (yes/no): yes

Suspending tenant...
├── Marking as suspended in tenants.yaml... ✓
├── Updating CloudWatch alarms... ✓
├── Notifying team... ✓
└── Tenant suspended ✓

────────────────────────────────────────────────
✓ Tenant Suspended

New login attempts will receive:
  "Tenant notifications-team is suspended. Contact admin."

To reactivate:
  panka tenant activate notifications-team
────────────────────────────────────────────────
```

### Activate Tenant

```bash
$ panka tenant activate notifications-team

Activate Tenant
────────────────────────────────────────────────

Tenant: notifications-team (Notifications Team)
Status: Suspended
Suspended: 2024-01-10 (5 days ago)

Reactivating tenant...
├── Marking as active in tenants.yaml... ✓
├── Updating CloudWatch alarms... ✓
├── Notifying team... ✓
└── Tenant activated ✓

────────────────────────────────────────────────
✓ Tenant Activated

Team can now log in and deploy normally.
────────────────────────────────────────────────
```

### Delete Tenant

```bash
$ panka tenant delete notifications-team

Delete Tenant
────────────────────────────────────────────────

⚠ DANGER: This will permanently delete the tenant

Tenant: notifications-team (Notifications Team)
Stacks: 3
State Size: 1.2 GB

⚠ This action:
  - Removes tenant from tenants.yaml
  - Invalidates all credentials
  - Archives state to backup location
  - Optionally deletes state after grace period

Backup Location:
  s3://company-panka-state/archive/notifications-team-2024-01-15/

Grace Period: 30 days (configurable)

Type 'notifications-team' to confirm: notifications-team

Deleting tenant...
├── Archiving state to backup location... ✓
│   Copied 1.2 GB to archive/
├── Removing from tenants.yaml... ✓
├── Invalidating all sessions... ✓
├── Scheduling cleanup (after 30 days)... ✓
├── Updating CloudWatch dashboards... ✓
└── Tenant deleted ✓

────────────────────────────────────────────────
✓ Tenant Deleted

Backup Location:
  s3://company-panka-state/archive/notifications-team-2024-01-15/

State will be permanently deleted after: 2024-02-14

To cancel deletion:
  panka tenant restore notifications-team
────────────────────────────────────────────────
```

---

## Monitoring and Alerts

### View Real-Time Activity

```bash
$ panka admin monitor

Real-Time Activity Monitor
────────────────────────────────────────────────
Refreshing every 5 seconds... (Ctrl+C to exit)

Active Deployments: 2
───────────────────────────────────────────────
Tenant              Stack               Env         Status      Duration
───────────────────────────────────────────────
notifications-team  notification-platform  prod    Deploying   2m 34s
payments-team       payment-platform       staging Deploying   1m 12s

Active Locks: 2
───────────────────────────────────────────────
Lock                                                Held By             Duration
───────────────────────────────────────────────────────────────────────
tenant:notifications-team:stack:notification-...    alice@company.com   2m 34s
tenant:payments-team:stack:payment-platform:...     bob@company.com     1m 12s

Recent Completions (Last 10 minutes):
───────────────────────────────────────────────
14:25  analytics-team      analytics-platform  prod    ✓ Success  4m 23s
14:22  notifications-team  sms-platform        dev     ✓ Success  3m 45s
14:18  payments-team       payment-platform    dev     ✗ Failed   2m 10s

System Health:
  S3 Operations: 342/hr
  DynamoDB Operations: 1,234/hr
  Active Sessions: 12
  Error Rate: 2.1%

Cost (Today):
  $83.45 (projected: $2,504/month)
────────────────────────────────────────────────
```

### Configure Alerts

```bash
$ panka admin alerts configure

Alert Configuration
────────────────────────────────────────────────

Cost Alerts:
  [x] Daily cost exceeds $100
  [x] Monthly projection exceeds $3,000
  [x] Tenant exceeds individual limit
  
  Recipients: platform-team@company.com

Error Alerts:
  [x] Deployment failure rate > 5%
  [x] Lock held > 1 hour
  [x] DynamoDB throttling detected
  
  Recipients: platform-team@company.com, oncall@company.com

Security Alerts:
  [x] Failed authentication attempts > 5
  [x] Admin login from new location
  [x] Tenant credential rotation overdue
  
  Recipients: security@company.com

Notification Channels:
  [x] Email
  [x] Slack (#panka-alerts)
  [ ] PagerDuty
  [ ] SNS

✓ Alerts configured
````

---

## Best Practices

### Security

1. **Admin Credentials**:
   ```bash
   # Rotate every 90 days
   panka admin rotate-password
   
   # Use AWS Secrets Manager
   # Never commit to Git
   # Require MFA for production
   ```

2. **Tenant Credentials**:
   ```bash
   # Rotate on team member departure
   panka tenant rotate <tenant-id>
   
   # Share via secure channels (1Password, Vault)
   # Monitor for unusual activity
   ```

3. **IAM Policies**:
   - Least privilege access
   - Separate roles per tenant (optional)
   - Regular audit of permissions

### Operations

1. **Regular Maintenance**:
   ```bash
   # Weekly: Review tenant usage
   panka tenant stats --period week
   
   # Monthly: Review costs
   panka admin costs --period month
   
   # Quarterly: Credential rotation
   panka admin rotate-all
   ```

2. **Backup**:
   ```bash
   # Automated: S3 versioning enabled
   # Automated: Cross-region replication
   # Manual: Export tenant configs
   panka admin export --all --output tenants-backup.yaml
   ```

3. **Monitoring**:
   - Set up CloudWatch dashboards
   - Configure cost alerts
   - Monitor deployment success rate
   - Track lock contention

---

## Troubleshooting

### Tenant Cannot Login

```bash
# Check tenant status
panka tenant show <tenant-id>

# Common issues:
# 1. Tenant suspended
panka tenant activate <tenant-id>

# 2. Wrong credentials
panka tenant rotate <tenant-id>

# 3. Session expired (team member issue, not admin)
# Team member runs: panka login
```

### High Costs

```bash
# Identify expensive tenants
panka tenant list --sort cost

# Review tenant details
panka tenant show <expensive-tenant>

# Check for:
# - Over-provisioned resources
# - Development stacks in production
# - Unused resources
# - Missing auto-scaling

# Contact tenant
# Or adjust cost limit:
panka tenant update <tenant-id> --cost-limit 3000
```

### Lock Contention

```bash
# View active locks
panka admin locks

# Force release stale lock (caution!)
panka admin unlock tenant:<tenant-id>:stack:<stack>:env:<env> --force

# Reason for stale locks:
# - CLI crashed
# - Network interruption
# - Manual termination

# Prevention:
# - TTL set to 1 hour (automatic cleanup)
# - Monitor lock hold times
```

---

## Summary

As a platform admin, you:

1. **Setup** (one-time):
   - Deploy infrastructure with Terraform
   - Configure admin credentials
   - Login with `panka admin login`

2. **Create Tenants**:
   - `panka tenant init`
   - Share credentials securely
   - Monitor usage

3. **Manage Lifecycle**:
   - List: `panka tenant list`
   - Details: `panka tenant show <tenant>`
   - Rotate: `panka tenant rotate <tenant>`
   - Suspend/Activate/Delete

4. **Monitor**:
   - Real-time: `panka admin monitor`
   - Stats: `panka tenant stats`
   - Alerts: Configure in CloudWatch

**Next Steps**:
- Share credentials with teams
- Point teams to [GETTING_STARTED_GUIDE.md](GETTING_STARTED_GUIDE.md)
- Set up monitoring dashboards
- Configure cost alerts

