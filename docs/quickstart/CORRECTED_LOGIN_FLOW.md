# Corrected Multi-Tenancy Login Flow ✅

## What Changed

**Before (❌ Wrong):**
- `panka login` asked for bucket, region, tenant name, and secret
- `panka admin login` asked for bucket, region, and password
- **Problem:** Teams shouldn't need to know infrastructure details

**After (✅ Correct):**
- `panka login` only asks for tenant name and secret
- `panka admin login` only asks for password
- **Bucket and region are read from `.panka.yaml`**

---

## Correct Setup Flow

### Step 1: Configure `.panka.yaml` (One-Time)

**Everyone** (admins and tenants) needs this file with backend configuration:

```yaml
# .panka.yaml
backend:
  type: s3
  bucket: company-panka-state    # ← Your S3 bucket
  region: us-east-1               # ← Your AWS region
  prefix: states/
  dynamodb_table: company-panka-locks
```

**Important:** This file should be:
- ✅ Committed to your infrastructure repo (with placeholders)
- ✅ Shared with all teams
- ✅ Placed in the directory where you run `panka` commands

---

### Step 2: Admin Login (Platform Team)

```bash
./bin/panka admin login
```

**Prompts:**
```
👤 Admin Authentication
──────────────────────────────────────────────────

📦 Using backend: s3://company-panka-state (region: us-east-1)

? Admin Password: ••••••••

Validating credentials...
✓ Admin authentication successful
```

**Only asks for:** Admin password  
**Reads from config:** Bucket and region

---

### Step 3: Create Tenants (Platform Team)

```bash
./bin/panka admin tenant init --name notifications-team
```

**Output:**
```
Creating tenant...
├── Generating secure credentials... ✓
└── Tenant created successfully ✓

Tenant ID:     notifications-team
Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG
               ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
               SAVE THIS - IT CANNOT BE RECOVERED
```

---

### Step 4: Share with Team

**Give teams:**
1. ✅ The `.panka.yaml` file (or tell them the bucket/region)
2. ✅ Their tenant ID: `notifications-team`
3. ✅ Their tenant secret: `ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG`

**Example sharing message:**
```
Hi Notifications Team!

Your Panka access is ready:

1. Create .panka.yaml with this content:
   backend:
     bucket: company-panka-state
     region: us-east-1

2. Login:
   panka login
   
   Tenant ID: notifications-team
   Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG

3. Start deploying!
```

---

### Step 5: Tenant Login (Development Team)

```bash
./bin/panka login
```

**Prompts:**
```
🔐 Tenant Authentication
──────────────────────────────────────────────────

📦 Using backend: s3://company-panka-state (region: us-east-1)

? Tenant Name: notifications-team
? Tenant Secret: ntfy_7Kx9pLmQ2wR8vN3jH6tY4bZ1cF5aS0dG

Authenticating...
├── Loading tenants.yaml... ✓
├── Finding tenant... ✓
├── Verifying credentials... ✓
└── Authentication successful ✓

✓ Logged in as: notifications-team
```

**Only asks for:** Tenant name and secret  
**Reads from config:** Bucket and region

---

## Benefits of This Approach

### ✅ Better Security
- Teams don't need AWS console access
- Teams don't need to know infrastructure details
- Credentials are isolated (tenant secrets only)

### ✅ Simpler UX
- One `.panka.yaml` file shared with everyone
- Login only asks for credentials
- No need to remember bucket names

### ✅ Centralized Configuration
- Platform team controls backend configuration
- Easy to update (just update `.panka.yaml` in repo)
- Consistent across all teams

---

## Configuration File Locations

Panka looks for `.panka.yaml` in:
1. Current directory (`./.panka.yaml`)
2. Home directory (`~/.panka.yaml`)

**Best practice:** Place `.panka.yaml` in your infrastructure repo root.

---

## Example `.panka.yaml` for Teams

```yaml
# Panka Configuration
# Shared across all teams

# Backend configuration (provided by platform team)
backend:
  type: s3
  bucket: company-panka-state
  region: us-east-1
  prefix: states/
  dynamodb_table: company-panka-locks

# Logging
log:
  level: info
  format: console

# AWS provider
providers:
  aws:
    region: us-east-1

# Default tags
default_tags:
  managed_by: panka
  environment: development
```

---

## Troubleshooting

### Error: "Failed to load config"

**Cause:** No `.panka.yaml` file found

**Solution:**
```bash
# Create .panka.yaml in current directory
cat > .panka.yaml << EOF
backend:
  bucket: company-panka-state
  region: us-east-1
  dynamodb_table: company-panka-locks
EOF
```

### Error: "Backend bucket not configured"

**Cause:** `.panka.yaml` exists but `backend.bucket` is empty

**Solution:**
Edit `.panka.yaml` and set:
```yaml
backend:
  bucket: your-actual-bucket-name
  region: us-east-1
```

### Still asks for bucket/region

**Cause:** Using old binary

**Solution:**
```bash
# Rebuild
cd /path/to/panka
go build -o bin/panka ./cmd/panka

# Verify
./bin/panka login --help
# Should say: "Read backend config from .panka.yaml"
```

---

## Summary

### Old Flow (Wrong):
```
panka login
? Bucket: company-panka-state    ← Teams shouldn't need this
? Region: us-east-1               ← Teams shouldn't need this
? Tenant: notifications-team
? Secret: ntfy_...
```

### New Flow (Correct):
```
# .panka.yaml (shared file)
backend:
  bucket: company-panka-state
  region: us-east-1

panka login
? Tenant: notifications-team     ← Only credentials
? Secret: ntfy_...                ← Only credentials
```

---

## What Teams Need

**Platform Team (Admin):**
1. `.panka.yaml` with backend config
2. Admin password

**Development Team (Tenant):**
1. `.panka.yaml` with backend config (same file)
2. Tenant ID
3. Tenant secret

**That's it!** ✅

---

## Next Steps

1. ✅ Update `.panka.yaml` with your bucket/region
2. ✅ Platform team: `panka admin login`
3. ✅ Platform team: `panka admin tenant init` for each team
4. ✅ Share `.panka.yaml` + credentials with teams
5. ✅ Teams: `panka login` and start deploying!

**The login flow is now much simpler!** 🎉

