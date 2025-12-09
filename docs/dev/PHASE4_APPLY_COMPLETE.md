# Phase 4: Apply Command & Networking Integration - Complete

**Date**: December 9, 2024  
**Status**: ✅ Complete

---

## Summary

Implemented the `panka apply` command and integrated networking provisioning with tenant management:

1. **Tenant Networking Provisioning**: `--create-networking` flag for `panka admin tenant init`
2. **Apply Command**: Full deployment pipeline with state management
3. **State Backend Integration**: S3-based state storage per tenant/stack

---

## New Features

### 1. Tenant Networking Provisioning

The `panka admin tenant init` command can now provision AWS networking:

```bash
# Create tenant WITH networking (provisions VPC, subnets, NAT, etc.)
panka admin tenant init my-team \
  --vpc-cidr 10.0.0.0/16 \
  --region us-east-1 \
  --nat-gateway \
  --create-networking

# Dry-run to preview
panka admin tenant init my-team \
  --vpc-cidr 10.0.0.0/16 \
  --create-networking \
  --dry-run
```

**New Flags:**
- `--create-networking` - Actually create AWS resources
- `--dry-run` - Preview what would be created

**What Gets Created:**
- VPC with DNS support
- Public and private subnets (2 AZs by default)
- Internet Gateway
- NAT Gateway (optional, with auto EIP)
- Route Tables (public → IGW, private → NAT)
- Default Security Group (allow internal traffic)

**Resource IDs Stored:**
After provisioning, the tenant config in S3 includes:
```yaml
networking:
  resourceIds:
    vpcId: vpc-xxxxxxx
    internetGatewayId: igw-xxxxxxx
    publicSubnetIds: [subnet-xxx, subnet-yyy]
    privateSubnetIds: [subnet-aaa, subnet-bbb]
    natGatewayIds: [nat-xxxxxxx]
    securityGroupId: sg-xxxxxxx
```

### 2. Apply Command

New command to deploy infrastructure:

```bash
# Deploy stack
panka apply ./my-stack

# Preview changes (dry-run)
panka apply ./my-stack --dry-run

# Skip confirmation
panka apply ./my-stack --auto-approve

# Target specific resource
panka apply ./my-stack --target api-server
```

**Pipeline Steps:**
1. ✓ Check authentication (tenant session)
2. ✓ Parse stack folder
3. ✓ Load tenant configuration (networking)
4. ✓ Validate configuration
5. ✓ Build dependency graph
6. ✓ Generate deployment plan
7. ✓ Initialize AWS provider
8. ✓ Load/create state
9. ✓ Apply changes (create resources)
10. ✓ Save state to S3

**State Storage:**
- Path: `s3://{bucket}/tenants/{tenant-id}/v1/stacks/{stack}/{env}/state.json`
- Versioned state with metadata
- Resource status tracking

---

## Code Changes

### Updated Files

| File | Changes |
|------|---------|
| `internal/cli/tenant_admin.go` | Added `--create-networking`, `--dry-run` flags; `provisionTenantNetworking()` function |
| `pkg/tenant/s3_backend.go` | Added `LoadTenantConfig()` method |

### New Files

| File | Description |
|------|-------------|
| `internal/cli/apply.go` | Complete apply command implementation |

---

## Usage Flow

### Admin Creates Tenant with Networking

```bash
# 1. Admin logs in
panka admin login

# 2. Admin creates tenant with AWS networking
panka admin tenant init payments-team \
  --vpc-cidr 10.1.0.0/16 \
  --region us-west-2 \
  --nat-gateway \
  --nat-type per-az \
  --create-networking \
  --output credentials.txt

# Output:
# ✓ Tenant Created
# 
# 📋 Tenant Details:
#   Tenant ID:      payments-team
#   S3 Path:        s3://bucket/tenants/payments-team/v1
#
# 🔗 AWS Resource IDs:
#   VPC:              vpc-0abc123...
#   Internet Gateway: igw-0def456...
#   Public Subnets:   [subnet-111..., subnet-222...]
#   Private Subnets:  [subnet-333..., subnet-444...]
#   NAT Gateways:     [nat-555..., nat-666...]
#   Security Group:   sg-0ghi789...
```

### Developer Deploys Stack

```bash
# 1. Developer logs in
panka login
# Enter: payments-team
# Enter: <secret>

# 2. Create stack folder
mkdir my-app && cd my-app
# ... create stack.yaml and services/...

# 3. Validate
panka validate .

# 4. Preview
panka apply . --dry-run

# 5. Deploy
panka apply . --auto-approve

# Output:
# 🚀 Panka Apply
# ──────────────────────────────────────────────────────────
# Stack Path: /path/to/my-app
#
# ⏳ Checking authentication... ✓
#    Tenant: payments-team
# ⏳ Parsing stack configuration... ✓
#    Stack: my-app
#    Services: 2
#    Components: 5
# ⏳ Loading tenant configuration... ✓
#    VPC: vpc-0abc123...
#    Security Group: sg-0ghi789...
# ⏳ Validating configuration... ✓
# ⏳ Building dependency graph... ✓
# ⏳ Generating deployment plan... ✓
#
# 📋 Deployment Plan
# ──────────────────────────────────────────────────────────
# Stage 1 (parallel - 3 resources)
#   + Create [SQS] notification-queue
#   + Create [S3] uploads-bucket
#   + Create [RDS] api-db
#
# Stage 2 (1 resource)
#   + Create [MicroService] api-server
#
# 🔧 Applying Changes
# ──────────────────────────────────────────────────────────
# 📦 Stage 1: 3 resource(s)
#    Creating [SQS] notification-queue... ✓
#    Creating [S3] uploads-bucket... ✓
#    Creating [RDS] api-db... ✓
#
# 📦 Stage 2: 1 resource(s)
#    Creating [MicroService] api-server... ✓
#
# ⏳ Saving state... ✓
#
# 📊 Apply Summary
# ──────────────────────────────────────────────────────────
# Stack:      my-app
# Duration:   2m 34s
# Created:    4
#
# ✨ Apply complete!
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    ADMIN WORKFLOW                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  panka admin tenant init my-team --create-networking        │
│           │                                                  │
│           ▼                                                  │
│  ┌────────────────────┐                                     │
│  │  Create Tenant     │                                     │
│  │  in S3 Registry    │                                     │
│  └─────────┬──────────┘                                     │
│            │                                                 │
│            ▼ (if --create-networking)                       │
│  ┌────────────────────┐                                     │
│  │  TenantNetworking  │                                     │
│  │  Orchestrator      │                                     │
│  └─────────┬──────────┘                                     │
│            │                                                 │
│            ▼                                                 │
│  ┌────────────────────────────────────────────────────┐    │
│  │  AWS Resources Created:                             │    │
│  │  • VPC, Subnets, IGW, NAT, SG, Route Tables        │    │
│  └────────────────────────────────────────────────────┘    │
│            │                                                 │
│            ▼                                                 │
│  ┌────────────────────┐                                     │
│  │  Store Resource IDs │                                    │
│  │  in Tenant Config   │                                    │
│  └─────────────────────┘                                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                   DEVELOPER WORKFLOW                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  panka apply ./my-stack                                     │
│           │                                                  │
│           ▼                                                  │
│  ┌────────────────────┐                                     │
│  │  Load Session      │──────────► Tenant ID                │
│  └─────────┬──────────┘                                     │
│            │                                                 │
│            ▼                                                 │
│  ┌────────────────────┐                                     │
│  │  Parse Stack       │──────────► Components               │
│  │  (FolderParser)    │                                     │
│  └─────────┬──────────┘                                     │
│            │                                                 │
│            ▼                                                 │
│  ┌────────────────────┐                                     │
│  │  Load Tenant       │──────────► VPC, SG IDs              │
│  │  Configuration     │                                     │
│  └─────────┬──────────┘                                     │
│            │                                                 │
│            ▼                                                 │
│  ┌────────────────────┐                                     │
│  │  Build Graph +     │──────────► Deployment Plan          │
│  │  Generate Plan     │                                     │
│  └─────────┬──────────┘                                     │
│            │                                                 │
│            ▼                                                 │
│  ┌────────────────────┐                                     │
│  │  Create Resources  │──────────► AWS SDK Calls            │
│  │  (per stage)       │                                     │
│  └─────────┬──────────┘                                     │
│            │                                                 │
│            ▼                                                 │
│  ┌────────────────────┐                                     │
│  │  Save State to S3  │                                     │
│  │  (per tenant/stack)│                                     │
│  └────────────────────┘                                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## State Structure

State is stored in S3 with the following structure:

```
s3://bucket/
└── tenants/
    └── {tenant-id}/
        ├── tenant.yaml           # Tenant metadata + networking IDs
        └── v1/
            └── stacks/
                └── {stack-name}/
                    └── {environment}/
                        └── state.json    # Deployment state
```

**state.json Format:**
```json
{
  "version": "1.0",
  "metadata": {
    "stack": "my-app",
    "environment": "default",
    "tenant": "payments-team",
    "deployed_by": "panka-cli",
    "created_at": "2024-12-09T10:00:00Z",
    "updated_at": "2024-12-09T10:05:00Z"
  },
  "resources": {
    "api-db": {
      "id": "arn:aws:rds:...",
      "type": "RDS",
      "name": "api-db",
      "provider": "aws",
      "status": "ready",
      "attributes": {
        "endpoint": "...",
        "port": "5432"
      }
    },
    "notification-queue": {
      "id": "arn:aws:sqs:...",
      "type": "SQS",
      ...
    }
  },
  "outputs": {}
}
```

---

## Next Steps

1. **Destroy Command**: Implement `panka destroy ./my-stack`
2. **Update Detection**: Compare current vs. desired state
3. **Drift Detection**: Detect and report configuration drift
4. **Rollback**: Implement automatic rollback on failure

