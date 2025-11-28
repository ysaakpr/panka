# Panka S3 State Store Structure

## Overview

Panka uses S3 to store:
1. **Tenant registry** (`tenants.yaml`) - List of all tenants and their configuration
2. **Deployment state** - Current state of deployed resources for each stack
3. **State history** - Versioned history of state changes (via S3 versioning)

---

## S3 Bucket Structure

### Multi-Tenant Mode (Recommended)

```
s3://d11dataplatform-panka-state/
│
├── tenants.yaml                           ← Global tenant registry
│
└── tenants/                               ← All tenant data
    │
    ├── notifications-team/                ← Tenant 1
    │   ├── tenant.yaml                    ← Tenant configuration
    │   └── v1/                            ← State version namespace
    │       └── stacks/                    ← All stacks for this tenant
    │           ├── notification-platform/ ← Stack 1
    │           │   ├── production/
    │           │   │   └── state.json     ← Production environment state
    │           │   ├── staging/
    │           │   │   └── state.json     ← Staging environment state
    │           │   └── development/
    │           │       └── state.json     ← Development environment state
    │           │
    │           └── sms-platform/          ← Stack 2
    │               ├── production/
    │               │   └── state.json
    │               └── staging/
    │                   └── state.json
    │
    ├── payments-team/                     ← Tenant 2
    │   ├── tenant.yaml
    │   └── v1/
    │       └── stacks/
    │           └── payment-platform/
    │               ├── production/
    │               │   └── state.json
    │               └── staging/
    │                   └── state.json
    │
    └── analytics-team/                    ← Tenant 3
        ├── tenant.yaml
        └── v1/
            └── stacks/
                └── analytics-platform/
                    └── production/
                        └── state.json
```

### Single-Tenant Mode (Simpler, No Multi-Tenancy)

```
s3://d11dataplatform-panka-state/
│
└── stacks/                                ← All stacks (no tenant isolation)
    ├── notification-platform/
    │   ├── production/
    │   │   └── state.json
    │   ├── staging/
    │   │   └── state.json
    │   └── development/
    │       └── state.json
    │
    └── payment-platform/
        └── production/
            └── state.json
```

---

## File Formats and Content

### 1. `tenants.yaml` (Root Level)

**Location:** `s3://bucket-name/tenants.yaml`

**Purpose:** Global registry of all tenants

**Content:**
```yaml
version: v1
metadata:
  created: 2024-11-27T10:00:00Z
  updated: 2024-11-27T12:00:00Z
  bucket: d11dataplatform-panka-state
  region: us-east-1

config:
  lockTable: panka-locks
  defaultVersion: v1

tenants:
  - id: notifications-team
    displayName: "Notifications Team"
    email: notifications@company.com
    status: active
    created: 2024-11-27T11:00:00Z
    updated: 2024-11-27T11:00:00Z
    
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

**Who accesses it:**
- ✅ Platform admins (during `panka admin tenant init/list/show`)
- ✅ Tenants (during `panka login` to verify credentials)

---

### 2. `tenant.yaml` (Per Tenant)

**Location:** `s3://bucket-name/tenants/<tenant-id>/tenant.yaml`

**Example:** `s3://bucket-name/tenants/notifications-team/tenant.yaml`

**Purpose:** Tenant-specific configuration and metadata

**Content:**
```yaml
tenant:
  id: notifications-team
  displayName: "Notifications Team"
  version: v1
  created: 2024-11-27T11:00:00Z

storage:
  bucket: d11dataplatform-panka-state
  prefix: tenants/notifications-team/v1

locks:
  table: panka-locks
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

**Who accesses it:**
- ✅ Tenant members (after login)
- ✅ Platform admins (for auditing)

---

### 3. `state.json` (Deployment State)

**Location:** `s3://bucket-name/tenants/<tenant-id>/v1/stacks/<stack-name>/<environment>/state.json`

**Example:** `s3://bucket-name/tenants/notifications-team/v1/stacks/notification-platform/production/state.json`

**Purpose:** Current state of deployed infrastructure

**Content:**
```json
{
  "version": "1.0.0",
  "format_version": 1,
  "metadata": {
    "created_at": "2024-11-27T10:00:00Z",
    "updated_at": "2024-11-27T12:30:00Z",
    "created_by": "alice@company.com",
    "stack_name": "notification-platform",
    "environment": "production",
    "tenant_id": "notifications-team"
  },
  "configuration": {
    "source_file": "infrastructure.yaml",
    "checksum": "sha256:abc123...",
    "variables": {
      "region": "us-east-1",
      "environment": "production"
    }
  },
  "resources": [
    {
      "id": "notification-queue",
      "type": "AWS::SQS::Queue",
      "name": "notification-platform-prod-queue",
      "provider": "aws",
      "status": "available",
      "created_at": "2024-11-27T10:05:00Z",
      "updated_at": "2024-11-27T10:05:00Z",
      "properties": {
        "QueueUrl": "https://sqs.us-east-1.amazonaws.com/123456789012/notification-platform-prod-queue",
        "QueueArn": "arn:aws:sqs:us-east-1:123456789012:notification-platform-prod-queue",
        "DelaySeconds": 0,
        "MessageRetentionPeriod": 345600,
        "VisibilityTimeout": 30
      },
      "dependencies": [],
      "tags": {
        "Environment": "production",
        "ManagedBy": "panka",
        "Stack": "notification-platform",
        "Tenant": "notifications-team"
      }
    },
    {
      "id": "notification-topic",
      "type": "AWS::SNS::Topic",
      "name": "notification-platform-prod-topic",
      "provider": "aws",
      "status": "available",
      "created_at": "2024-11-27T10:06:00Z",
      "updated_at": "2024-11-27T10:06:00Z",
      "properties": {
        "TopicArn": "arn:aws:sns:us-east-1:123456789012:notification-platform-prod-topic",
        "DisplayName": "Notification Platform Production Topic"
      },
      "dependencies": ["notification-queue"],
      "tags": {
        "Environment": "production",
        "ManagedBy": "panka",
        "Stack": "notification-platform"
      }
    },
    {
      "id": "notification-db",
      "type": "AWS::DynamoDB::Table",
      "name": "notification-platform-prod-events",
      "provider": "aws",
      "status": "available",
      "created_at": "2024-11-27T10:07:00Z",
      "updated_at": "2024-11-27T10:07:00Z",
      "properties": {
        "TableName": "notification-platform-prod-events",
        "TableArn": "arn:aws:dynamodb:us-east-1:123456789012:table/notification-platform-prod-events",
        "KeySchema": [
          {
            "AttributeName": "event_id",
            "KeyType": "HASH"
          }
        ],
        "AttributeDefinitions": [
          {
            "AttributeName": "event_id",
            "AttributeType": "S"
          }
        ],
        "BillingMode": "PAY_PER_REQUEST"
      },
      "dependencies": [],
      "tags": {
        "Environment": "production",
        "ManagedBy": "panka"
      }
    }
  ],
  "outputs": {
    "queue_url": "https://sqs.us-east-1.amazonaws.com/123456789012/notification-platform-prod-queue",
    "topic_arn": "arn:aws:sns:us-east-1:123456789012:notification-platform-prod-topic",
    "table_name": "notification-platform-prod-events"
  },
  "deployment": {
    "plan_hash": "sha256:def456...",
    "started_at": "2024-11-27T10:05:00Z",
    "completed_at": "2024-11-27T10:08:00Z",
    "duration_seconds": 180,
    "status": "success"
  }
}
```

**Who accesses it:**
- ✅ Panka CLI during `plan`, `apply`, `destroy`, `state list/show`
- ✅ Read during deployments to determine current state
- ✅ Written after successful deployments

---

## S3 Key Patterns

### Tenant Registry
```
Key: tenants.yaml
Purpose: Global tenant list
Access: Admins (write), All tenants (read during login)
```

### Tenant Configuration
```
Key: tenants/<tenant-id>/tenant.yaml
Purpose: Tenant-specific config
Access: Tenant members, Admins
```

### Deployment State
```
Key: tenants/<tenant-id>/v1/stacks/<stack-name>/<environment>/state.json
Purpose: Current infrastructure state
Access: Tenant members only
```

### Single-Tenant Mode (No Isolation)
```
Key: stacks/<stack-name>/<environment>/state.json
Purpose: Current infrastructure state
Access: Anyone with AWS credentials
```

---

## S3 Versioning (History)

When S3 versioning is enabled (recommended), every change to `state.json` creates a new version:

```
s3://bucket/tenants/team-a/v1/stacks/my-stack/production/state.json

Versions:
├── v1: 2024-11-27T10:00:00Z (Initial deployment - 3 resources)
├── v2: 2024-11-27T12:00:00Z (Added SNS topic - 4 resources)
├── v3: 2024-11-27T15:00:00Z (Updated DynamoDB table - 4 resources)
└── v4: 2024-11-27T18:00:00Z (Current - 5 resources)
```

**Benefits:**
- ✅ Rollback to previous state if needed
- ✅ Audit trail of all changes
- ✅ Disaster recovery
- ✅ Compliance (who changed what when)

**View history:**
```bash
aws s3api list-object-versions \
  --bucket d11dataplatform-panka-state \
  --prefix tenants/notifications-team/v1/stacks/notification-platform/production/state.json
```

---

## How Panka Uses S3

### During `panka login`
1. Reads `tenants.yaml` from S3
2. Finds tenant by ID
3. Verifies credentials (bcrypt)
4. Reads `tenants/<tenant-id>/tenant.yaml`
5. Creates local session file

### During `panka plan`
1. Parses your `infrastructure.yaml`
2. Reads current state from S3: `tenants/<tenant-id>/v1/stacks/<stack>/production/state.json`
3. Compares desired vs current state
4. Generates deployment plan
5. Does NOT modify S3

### During `panka apply` (Future Phase)
1. Acquires lock in DynamoDB
2. Re-reads current state (in case it changed)
3. Executes deployment plan
4. Updates AWS resources
5. Writes new state to S3
6. Releases lock

### During `panka state list`
1. Reads state from S3
2. Displays all resources
3. Read-only operation

### During `panka destroy`
1. Acquires lock
2. Reads current state
3. Destroys resources in reverse dependency order
4. Either deletes state file OR marks all resources as destroyed
5. Releases lock

---

## State File Size & Performance

### Typical Sizes
- **Small stack** (5-10 resources): ~10-20 KB
- **Medium stack** (50 resources): ~50-100 KB
- **Large stack** (200 resources): ~200-500 KB

### Performance
- **Read operation**: ~100-300ms (depends on file size and S3 region)
- **Write operation**: ~200-500ms
- **S3 API costs**: $0.005 per 1,000 GET requests, $0.005 per 1,000 PUT requests

### Optimization Tips
1. ✅ Use S3 in the same region as your deployments
2. ✅ Enable S3 versioning for history
3. ✅ Use lifecycle policies to archive old versions (e.g., move to Glacier after 90 days)
4. ✅ Don't store large files in state (use references instead)

---

## Security & Access Control

### IAM Permissions Required

**For Platform Admins:**
```json
{
  "Effect": "Allow",
  "Action": [
    "s3:GetObject",
    "s3:PutObject",
    "s3:DeleteObject",
    "s3:ListBucket"
  ],
  "Resource": [
    "arn:aws:s3:::d11dataplatform-panka-state",
    "arn:aws:s3:::d11dataplatform-panka-state/*"
  ]
}
```

**For Tenant Members (Enhanced with IAM):**
```json
{
  "Effect": "Allow",
  "Action": [
    "s3:GetObject",
    "s3:PutObject",
    "s3:DeleteObject",
    "s3:ListBucket"
  ],
  "Resource": [
    "arn:aws:s3:::d11dataplatform-panka-state/tenants.yaml",
    "arn:aws:s3:::d11dataplatform-panka-state/tenants/notifications-team/*"
  ]
}
```

### Encryption
- ✅ Enable S3 server-side encryption (SSE-S3 or SSE-KMS)
- ✅ Enable encryption in transit (HTTPS only)
- ✅ Consider KMS for sensitive data

---

## Example: Complete S3 State for a Stack

Let's say you deploy this stack:

**infrastructure.yaml:**
```yaml
stack:
  name: notification-platform
  environment: production

components:
  - kind: SQS
    name: event-queue
    properties:
      visibilityTimeout: 30
      messageRetentionPeriod: 345600
  
  - kind: SNS
    name: alert-topic
    properties:
      displayName: "Alert Topic"
  
  - kind: DynamoDB
    name: events-table
    properties:
      billingMode: PAY_PER_REQUEST
```

**Resulting S3 structure:**
```
s3://d11dataplatform-panka-state/
└── tenants/
    └── notifications-team/
        ├── tenant.yaml              ← Created during tenant init
        └── v1/
            └── stacks/
                └── notification-platform/
                    └── production/
                        └── state.json   ← Created during first apply
                        
                        Version History (if S3 versioning enabled):
                        ├── v1 (2024-11-27 10:00) - Initial deployment
                        ├── v2 (2024-11-27 12:00) - Updated queue timeout
                        └── v3 (2024-11-27 15:00) - Current
```

---

## Commands to Inspect State

### View Tenant Registry
```bash
aws s3 cp s3://d11dataplatform-panka-state/tenants.yaml - | cat
```

### View Tenant Configuration
```bash
aws s3 cp s3://d11dataplatform-panka-state/tenants/notifications-team/tenant.yaml - | cat
```

### View Deployment State
```bash
aws s3 cp s3://d11dataplatform-panka-state/tenants/notifications-team/v1/stacks/notification-platform/production/state.json - | jq .
```

### List All States for a Tenant
```bash
aws s3 ls s3://d11dataplatform-panka-state/tenants/notifications-team/v1/stacks/ --recursive
```

### View State History (with versioning enabled)
```bash
aws s3api list-object-versions \
  --bucket d11dataplatform-panka-state \
  --prefix tenants/notifications-team/v1/stacks/notification-platform/production/state.json \
  | jq '.Versions[] | {VersionId, LastModified, Size}'
```

### Download Previous Version
```bash
aws s3api get-object \
  --bucket d11dataplatform-panka-state \
  --key tenants/notifications-team/v1/stacks/notification-platform/production/state.json \
  --version-id "version-id-here" \
  /tmp/previous-state.json
```

---

## Summary

### Key Patterns

| File | Location | Purpose | Who Writes | Who Reads |
|------|----------|---------|------------|-----------|
| `tenants.yaml` | Root | Global registry | Admin | All |
| `tenant.yaml` | `tenants/<id>/` | Tenant config | Admin | Tenant |
| `state.json` | `tenants/<id>/v1/stacks/<name>/<env>/` | Deployment state | Panka apply | Panka commands |

### Tenant Isolation

**Each tenant gets:**
- ✅ Separate S3 prefix: `tenants/<tenant-id>/`
- ✅ Own state files
- ✅ Cannot read/write other tenants' data (with IAM policies)
- ✅ Version history per tenant

### State Lifecycle

```
1. Tenant created
   └─> tenants.yaml updated
   └─> tenants/<id>/tenant.yaml created

2. First deployment
   └─> state.json created

3. Subsequent deployments
   └─> state.json updated (new S3 version)

4. Stack destroyed
   └─> state.json deleted OR marked as destroyed
```

---

**Ready to explore?** Check your bucket structure:
```bash
export AWS_PROFILE=AdministratorAccess-499063035928
aws s3 ls s3://d11dataplatform-panka-state/ --recursive
```

