# AWS Credentials Setup for Panka

## Your Situation

You have two AWS credential options:
1. **Default profile** - `streamverse-ci` user (limited permissions, may be expired)
2. **SSO profile** - `AdministratorAccess-499063035928` (your personal admin account)

## ✅ Recommended Solution

Use your SSO profile with AdministratorAccess.

---

## Step-by-Step Setup

### 1. Login to AWS SSO

```bash
aws sso login --profile AdministratorAccess-499063035928
```

**Complete the authentication in your browser when it opens.**

### 2. Set AWS Profile for This Session

```bash
export AWS_PROFILE=AdministratorAccess-499063035928
```

### 3. Verify Your Identity

```bash
aws sts get-caller-identity
```

You should see YOUR personal AWS account, NOT `streamverse-ci`.

### 4. Create AWS Resources

**Create S3 Bucket:**
```bash
aws s3 mb s3://d11dataplatform-panka-state --region us-east-1

# Enable versioning (recommended)
aws s3api put-bucket-versioning \
  --bucket d11dataplatform-panka-state \
  --versioning-configuration Status=Enabled \
  --region us-east-1
```

**Create DynamoDB Table:**
```bash
aws dynamodb create-table \
  --table-name panka-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1 \
  --table-class STANDARD
```

### 5. Test Access

```bash
# Test S3
aws s3 ls s3://d11dataplatform-panka-state/

# Test DynamoDB
aws dynamodb describe-table --table-name panka-locks --region us-east-1
```

### 6. Use Panka

```bash
# Make sure AWS_PROFILE is still set
export AWS_PROFILE=AdministratorAccess-499063035928

# Admin login
./bin/panka admin login

# Create tenant
./bin/panka admin tenant init --name my-first-team
```

---

## 🔄 For Future Sessions

Every time you open a new terminal, you'll need to:

```bash
# 1. Check if SSO session is still valid
aws sts get-caller-identity --profile AdministratorAccess-499063035928

# 2. If expired, login again
aws sso login --profile AdministratorAccess-499063035928

# 3. Set the profile
export AWS_PROFILE=AdministratorAccess-499063035928

# 4. Use Panka
./bin/panka admin tenant list
```

---

## 💡 Make It Permanent (Optional)

Add to your `~/.zshrc` or `~/.bashrc`:

```bash
# Default AWS profile for Panka
export AWS_PROFILE=AdministratorAccess-499063035928
```

Then reload: `source ~/.zshrc`

---

## 🐛 Troubleshooting

### Error: "ExpiredToken"

**Cause:** SSO session expired (typically after 12 hours)

**Fix:**
```bash
aws sso login --profile AdministratorAccess-499063035928
export AWS_PROFILE=AdministratorAccess-499063035928
```

### Error: "AccessDenied" 

**Cause:** Using wrong profile (default/streamverse-ci)

**Fix:**
```bash
# Check current identity
aws sts get-caller-identity

# Should show YOUR account, not streamverse-ci
# If wrong, set profile:
export AWS_PROFILE=AdministratorAccess-499063035928
```

### Error: "NoSuchBucket"

**Cause:** S3 bucket doesn't exist yet

**Fix:**
```bash
aws s3 mb s3://d11dataplatform-panka-state --region us-east-1
```

### Error: "ResourceNotFoundException" (DynamoDB)

**Cause:** DynamoDB table doesn't exist yet

**Fix:**
```bash
aws dynamodb create-table \
  --table-name panka-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1
```

---

## 📋 Quick Reference

```bash
# Login
aws sso login --profile AdministratorAccess-499063035928

# Set profile
export AWS_PROFILE=AdministratorAccess-499063035928

# Verify
aws sts get-caller-identity

# Use Panka
./bin/panka admin login
./bin/panka admin tenant init --name team-name
```

---

## Summary

The "ExpiredToken" error was because:
1. Your default profile is `streamverse-ci` (limited permissions)
2. Panka was using default credentials, not your SSO profile
3. You need to explicitly set `AWS_PROFILE` environment variable

**Solution:** Always set `export AWS_PROFILE=AdministratorAccess-499063035928` before using Panka!

