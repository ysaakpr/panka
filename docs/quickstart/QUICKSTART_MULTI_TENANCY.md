# Panka Multi-Tenancy Quick Start

## 🚀 5-Minute Setup

### Prerequisites

- AWS account with S3 and DynamoDB access
- Panka CLI installed (`./bin/panka`)
- AWS credentials configured

---

## For Platform Administrators

### Step 1: Create AWS Resources (One-Time)

```bash
# Create S3 bucket
aws s3 mb s3://company-panka-state --region us-east-1

# Create DynamoDB table
aws dynamodb create-table \
  --table-name company-panka-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST
```

### Step 2: Admin Login

```bash
./bin/panka admin login
```

**Interactive prompts:**
```
? S3 Bucket: company-panka-state
? AWS Region [us-east-1]: us-east-1
? Admin Password: ••••••••

✓ Admin authentication successful
```

### Step 3: Create First Tenant

```bash
./bin/panka admin tenant init
```

**Interactive prompts:**
```
? Tenant Name: notifications-team
? Display Name: Notifications Team
? Contact Email: notifications@company.com

Creating tenant...
✓ Tenant created

Tenant ID:     notifications-team
Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
               ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
               SAVE THIS - IT CANNOT BE RECOVERED
```

### Step 4: Share Credentials with Team

Send via secure channel (1Password, encrypted email, etc.):

```
Panka Access for Notifications Team

Tenant: notifications-team
Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
Bucket: company-panka-state
Region: us-east-1

Getting Started:
1. Run: ./bin/panka login
2. Enter the credentials above
3. Start deploying!
```

---

## For Development Teams

### Step 1: Login

```bash
./bin/panka login
```

**Interactive prompts:**
```
? S3 Bucket: company-panka-state
? AWS Region [us-east-1]: us-east-1
? Tenant Name: notifications-team
? Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG

✓ Logged in as: notifications-team
```

### Step 2: Use Panka Normally

```bash
# Validate your infrastructure
./bin/panka validate infrastructure.yaml

# Generate deployment plan
./bin/panka plan infrastructure.yaml

# Visualize dependencies
./bin/panka graph infrastructure.yaml

# Check state
./bin/panka state list
```

**All operations are automatically scoped to your tenant!**

Your state is saved to:
```
s3://company-panka-state/tenants/notifications-team/v1/stacks/...
```

Your locks use keys like:
```
tenant:notifications-team:stack:my-app:env:production
```

---

## Common Admin Tasks

### List All Tenants

```bash
./bin/panka admin tenant list
```

Output:
```
ID                  NAME                  STATUS    CREATED
notifications-team  Notifications Team    ✓ active  2024-01-15
payments-team       Payments Team         ✓ active  2024-01-16
analytics-team      Analytics Team        ⏸ suspended 2024-01-17
```

### Show Tenant Details

```bash
./bin/panka admin tenant show notifications-team
```

### Rotate Credentials

```bash
./bin/panka admin tenant rotate notifications-team
```

**⚠️ Warning:** This invalidates the old secret. Share the new secret with the team.

### Suspend a Tenant

```bash
./bin/panka admin tenant suspend analytics-team
```

### Activate a Tenant

```bash
./bin/panka admin tenant activate analytics-team
```

---

## Session Management

### Check Your Session

**Admin:**
```bash
./bin/panka admin session
```

**Tenant:**
```bash
# Session info shown during login
# Check session file: ~/.panka/session
cat ~/.panka/session
```

### Logout

```bash
./bin/panka logout
```

---

## Non-Interactive Mode

### Admin Login with Flags

```bash
./bin/panka admin login \
  --bucket company-panka-state \
  --region us-east-1
# Still prompts for password (for security)
```

### Create Tenant with Flags

```bash
./bin/panka admin tenant init \
  --name payments-team \
  --display-name "Payments Team" \
  --email payments@company.com \
  --cost-limit 10000 \
  --max-stacks 200
```

### Tenant Login with Flags

```bash
./bin/panka login \
  --bucket company-panka-state \
  --region us-east-1
# Still prompts for tenant name and secret
```

---

## Troubleshooting

### "No valid session found"

```bash
# Login again
./bin/panka login  # for tenants
./bin/panka admin login  # for admins
```

### "Authentication failed: Invalid credentials"

- Check that you're using the correct tenant secret
- Credentials may have been rotated - ask your admin
- Ensure you're not copying extra spaces

### "Tenant not found"

- Check tenant name spelling (case-sensitive)
- Verify with admin: `panka admin tenant list`

### "Session expired"

- Admin sessions expire after 8 hours
- Tenant sessions expire after 7 days
- Just login again

---

## Security Best Practices

### For Admins

1. **Protect Admin Credentials**
   - Store admin password in password manager
   - Don't share admin access widely
   - Rotate regularly

2. **Tenant Secret Management**
   - Never commit secrets to Git
   - Share via secure channels only (1Password, encrypted email)
   - Rotate on team member departure

3. **Regular Audits**
   - Review tenant list regularly: `panka admin tenant list`
   - Suspend unused tenants
   - Check for suspicious activity

### For Teams

1. **Secure Storage**
   - Store tenant secret in team password manager
   - Don't commit to Git or CI configs
   - Use environment variables if needed

2. **Session Files**
   - Located at `~/.panka/session`
   - Permissions: 0600 (user-only access)
   - Deleted on logout

---

## Quick Reference

### Admin Commands

```bash
panka admin login                      # Login as admin
panka admin logout                     # Logout
panka admin session                    # Show session
panka admin tenant init                # Create tenant
panka admin tenant list                # List tenants
panka admin tenant show <id>           # Show details
panka admin tenant rotate <id>         # Rotate credentials
panka admin tenant suspend <id>        # Suspend
panka admin tenant activate <id>       # Activate
```

### Tenant Commands

```bash
panka login                            # Login as tenant
panka logout                           # Logout
panka validate <file>                  # Validate config
panka plan <file>                      # Generate plan
panka graph <file>                     # Show graph
panka state list                       # List resources
```

---

## What You Get with Multi-Tenancy

✅ **Complete Isolation**
- Each team has separate S3 prefix
- Each team has namespaced locks
- No cross-team access possible

✅ **Secure Authentication**
- Bcrypt-hashed credentials
- Session-based access
- Automatic expiry

✅ **Easy Management**
- Create tenants in seconds
- Rotate credentials anytime
- Suspend/activate as needed

✅ **Transparent to Teams**
- Teams use normal Panka commands
- Isolation is automatic
- No extra configuration needed

---

## Next Steps

1. **Setup AWS Resources** (if not done)
2. **Login as Admin**
3. **Create Tenants for Each Team**
4. **Share Credentials**
5. **Teams Login and Start Deploying!**

For detailed documentation, see:
- `MULTI_TENANCY_IMPLEMENTATION.md` - Complete technical details
- `docs/MULTI_TENANCY.md` - Architecture overview
- `docs/PLATFORM_ADMIN_GUIDE.md` - Admin guide

---

**Happy Multi-Tenanting! 🎉**

