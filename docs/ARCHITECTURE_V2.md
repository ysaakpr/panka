# Panka Architecture v2.0

> **This is the authoritative architecture document for Panka.**  
> All previous architecture documents are superseded by this one.

**Last Updated:** December 2024  
**Status:** Active Implementation

---

## Table of Contents

1. [Overview](#overview)
2. [Core Concepts](#core-concepts)
3. [Hierarchy Model](#hierarchy-model)
4. [Folder Structure](#folder-structure)
5. [Data Model](#data-model)
6. [Networking Architecture](#networking-architecture)
7. [User Workflows](#user-workflows)
8. [CLI Commands](#cli-commands)
9. [State Management](#state-management)
10. [Implementation Phases](#implementation-phases)

---

## Overview

Panka is a **multi-tenant infrastructure deployment tool** that manages cloud resources declaratively using YAML configuration with an opinionated folder-based structure.

### Key Principles

1. **Tenant-First**: All networking and isolation is defined at tenant level
2. **Folder-Based**: Stacks are defined as folders, not single files
3. **Inheritance**: Tenant → Stack → Service → Component
4. **Convention over Configuration**: Sensible defaults, override when needed
5. **Security by Default**: Services in same stack can communicate automatically

---

## Core Concepts

### Tenant

A **Tenant** is an isolated environment managed by a platform admin.

- Owns networking resources (VPC, Subnets, NAT, Security Groups)
- Contains multiple stacks
- Has resource limits and policies
- Credentials managed by admin

**Owner:** Platform Admin  
**Stored:** `s3://bucket/tenants/{tenant-id}/tenant.yaml`

### Stack

A **Stack** is a group of related services deployed together.

- Defined as a **folder** containing `stack.yaml` and `services/`
- Inherits networking from tenant
- Can override security group rules
- Contains multiple services

**Owner:** Tenant User  
**Stored:** `{local-folder}/stack.yaml`

### Service

A **Service** is a logical grouping of related components.

- Lives in `services/{service-name}/` folder
- Can have multiple YAML files
- All components in a service share the same networking
- Can communicate with other services in the same stack

**Owner:** Tenant User  
**Stored:** `{stack-folder}/services/{service-name}/service.yaml`

### Component

A **Component** is an individual AWS resource.

- Defined in YAML files within a service folder
- Types: MicroService (ECS), Lambda, RDS, DynamoDB, S3, SQS, SNS, etc.
- Inherits networking from service → stack → tenant

**Owner:** Tenant User  
**Stored:** `{service-folder}/*.yaml`

---

## Hierarchy Model

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              PLATFORM                                   │
│                         (panka admin login)                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                        TENANT A                                  │   │
│  │                   (notifications-team)                           │   │
│  │                                                                  │   │
│  │  Networking: VPC 10.0.0.0/16                                    │   │
│  │  ├── Public Subnets: 10.0.1.0/24, 10.0.2.0/24                  │   │
│  │  ├── Private Subnets: 10.0.10.0/24, 10.0.20.0/24               │   │
│  │  ├── NAT Gateway: Enabled                                       │   │
│  │  └── Security Group: Allow internal traffic                     │   │
│  │                                                                  │   │
│  │  ┌─────────────────┐  ┌─────────────────┐                       │   │
│  │  │   STACK 1       │  │   STACK 2       │                       │   │
│  │  │ (notification-  │  │ (analytics-     │                       │   │
│  │  │  platform)      │  │  pipeline)      │                       │   │
│  │  │                 │  │                 │                       │   │
│  │  │ ┌───────────┐   │  │ ┌───────────┐   │                       │   │
│  │  │ │ api       │   │  │ │ collector │   │                       │   │
│  │  │ │ service   │   │  │ │ service   │   │                       │   │
│  │  │ ├───────────┤   │  │ ├───────────┤   │                       │   │
│  │  │ │ worker    │   │  │ │ processor │   │                       │   │
│  │  │ │ service   │   │  │ │ service   │   │                       │   │
│  │  │ └───────────┘   │  │ └───────────┘   │                       │   │
│  │  └─────────────────┘  └─────────────────┘                       │   │
│  │                                                                  │   │
│  │  All stacks share the same VPC and can communicate              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                        TENANT B                                  │   │
│  │                     (payments-team)                              │   │
│  │                                                                  │   │
│  │  Networking: VPC 10.1.0.0/16 (isolated from Tenant A)           │   │
│  │  ...                                                             │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Folder Structure

### Stack Folder Layout

```
my-stack/                              # Stack root folder
├── stack.yaml                         # Stack definition
└── services/                          # Services folder
    ├── api/                           # Service: api
    │   ├── service.yaml               # Service definition
    │   ├── ecs.yaml                   # ECS/Fargate components
    │   ├── resources.yaml             # RDS, SQS, S3, etc.
    │   └── config/                    # Optional config files
    │       ├── app.env                # Environment variables
    │       └── settings.yaml          # Application config
    │
    ├── worker/                        # Service: worker
    │   ├── service.yaml
    │   ├── lambda.yaml                # Lambda functions
    │   └── queues.yaml                # SQS queues
    │
    └── scheduler/                     # Service: scheduler
        ├── service.yaml
        └── eventbridge.yaml           # EventBridge rules
```

### Naming Conventions

| Item | Convention | Example |
|------|------------|---------|
| Stack folder | lowercase-hyphen | `notification-platform/` |
| Service folder | lowercase-hyphen | `api-gateway/` |
| YAML files | lowercase | `service.yaml`, `ecs.yaml` |
| Config folder | `config/` | `services/api/config/` |

---

## Data Model

### 1. Tenant Configuration

**Location:** `s3://{bucket}/tenants/{tenant-id}/tenant.yaml`  
**Created by:** `panka admin tenant init`

```yaml
apiVersion: admin.panka.io/v1
kind: TenantConfig
metadata:
  name: notifications-team
  createdBy: platform-admin@company.com
  createdAt: "2024-11-28T10:00:00Z"

spec:
  # AWS Account binding
  aws:
    accountId: "123456789012"
    region: us-east-1
    assumeRoleArn: "arn:aws:iam::123456789012:role/panka-notifications-team"

  # Networking - shared by ALL stacks
  networking:
    vpc:
      cidrBlock: "10.0.0.0/16"
      enableDNSHostnames: true
      enableDNSSupport: true

    subnets:
      public:
        - cidrBlock: "10.0.1.0/24"
          availabilityZone: us-east-1a
        - cidrBlock: "10.0.2.0/24"
          availabilityZone: us-east-1b

      private:
        - cidrBlock: "10.0.10.0/24"
          availabilityZone: us-east-1a
        - cidrBlock: "10.0.20.0/24"
          availabilityZone: us-east-1b

    natGateway:
      enabled: true
      type: single  # single | per-az

    internetGateway:
      enabled: true

    defaultSecurityGroup:
      name: "{tenant}-default-sg"
      allowInternalTraffic: true
      egress:
        - protocol: "-1"
          cidrBlocks: ["0.0.0.0/0"]
          description: "Allow all outbound"

  # Resource limits
  limits:
    maxStacks: 10
    maxServicesPerStack: 20
    maxResourcesPerService: 50

  # Default tags for all resources
  defaultTags:
    tenant: notifications-team
    managed-by: panka
    cost-center: platform-engineering

  # Allowed resource types
  allowedResources:
    - MicroService
    - Lambda
    - RDS
    - DynamoDB
    - S3
    - SQS
    - SNS
```

### 2. Stack Definition

**Location:** `{stack-folder}/stack.yaml`  
**Created by:** User or `panka stack init`

```yaml
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: notification-platform
  tenant: notifications-team          # Links to tenant
  description: "Notification platform for sending emails, SMS, and push notifications"
  labels:
    environment: production
    team: notifications

spec:
  # Variables available to all services
  variables:
    ENVIRONMENT: production
    LOG_LEVEL: info
    DOMAIN: notifications.example.com

  # Optional: Override security group (adds to tenant defaults)
  networking:
    securityGroup:
      ingress:
        - port: 443
          protocol: tcp
          cidrBlocks: ["0.0.0.0/0"]
          description: "HTTPS from internet"
        - port: 80
          protocol: tcp
          cidrBlocks: ["0.0.0.0/0"]
          description: "HTTP from internet (redirect to HTTPS)"
```

### 3. Service Definition

**Location:** `{stack-folder}/services/{service-name}/service.yaml`  
**Created by:** User or `panka service add`

```yaml
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: api
  stack: notification-platform
  description: "REST API for notification management"
  labels:
    tier: frontend

spec:
  # Service-level variables
  variables:
    PORT: "8080"
    REPLICAS: "3"
    MEMORY: "512Mi"
    CPU: "256m"

  # Optional: Dependencies on other services
  dependsOn:
    - worker  # API depends on worker service
```

### 4. Component Definitions

**Location:** `{service-folder}/*.yaml`

#### MicroService (ECS/Fargate)

```yaml
# services/api/ecs.yaml
apiVersion: components.panka.io/v1
kind: MicroService
metadata:
  name: api-server
  service: api
  stack: notification-platform
  description: "Main API server"

spec:
  image:
    repository: 123456789012.dkr.ecr.us-east-1.amazonaws.com/api
    tag: "${VERSION:-latest}"
    pullPolicy: Always

  runtime:
    platform: fargate
    cpu: "${CPU}"
    memory: "${MEMORY}"

  replicas: "${REPLICAS}"

  ports:
    - name: http
      port: 8080
      protocol: tcp

  healthCheck:
    path: /health
    port: 8080
    initialDelaySeconds: 30
    periodSeconds: 10

  environment:
    - name: ENV
      value: "${ENVIRONMENT}"
    - name: PORT
      value: "${PORT}"
    - name: LOG_LEVEL
      value: "${LOG_LEVEL}"
    - name: DB_HOST
      valueFrom:
        component: api-db
        output: endpoint
    - name: QUEUE_URL
      valueFrom:
        component: notification-queue
        output: queueUrl

  secrets:
    - name: DB_PASSWORD
      secretRef: "${ENVIRONMENT}/api/db-password"

  # Networking inherited automatically!
  # - VPC: from tenant
  # - Subnets: private subnets from tenant
  # - Security Group: stack's security group (inherits tenant default)

  dependsOn:
    - api-db
    - notification-queue
```

#### RDS Database

```yaml
# services/api/resources.yaml
---
apiVersion: components.panka.io/v1
kind: RDS
metadata:
  name: api-db
  service: api
  stack: notification-platform

spec:
  engine:
    type: postgres
    version: "15"
    parameters:
      max_connections: "200"

  instance:
    class: db.t3.medium
    storage:
      type: gp3
      allocatedGB: 100
      maxAllocatedGB: 500
    multiAZ: true

  database:
    name: notifications
    username: dbadmin
    passwordSecret:
      ref: "${ENVIRONMENT}/api/db-password"
    port: 5432

  backup:
    enabled: true
    retentionDays: 7
    preferredWindow: "03:00-04:00"

  # Networking inherited automatically!
  # Placed in private subnets from tenant
```

#### SQS Queue

```yaml
# services/api/resources.yaml (continued)
---
apiVersion: components.panka.io/v1
kind: SQS
metadata:
  name: notification-queue
  service: api
  stack: notification-platform

spec:
  type: standard
  messageRetentionPeriod: 345600  # 4 days
  visibilityTimeout: 300          # 5 minutes
  receiveWaitTime: 20             # Long polling

  deadLetterQueue:
    enabled: true
    maxReceiveCount: 3
```

---

## Networking Architecture

### Tenant-Level Networking

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         VPC: 10.0.0.0/16                                │
│                    (Created by tenant admin)                            │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                      PUBLIC SUBNETS                                │ │
│  │  ┌─────────────────────┐     ┌─────────────────────┐              │ │
│  │  │  10.0.1.0/24        │     │  10.0.2.0/24        │              │ │
│  │  │  us-east-1a         │     │  us-east-1b         │              │ │
│  │  │                     │     │                     │              │ │
│  │  │  • ALB              │     │  • ALB              │              │ │
│  │  │  • NAT Gateway      │     │                     │              │ │
│  │  └─────────────────────┘     └─────────────────────┘              │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                              │                                          │
│                              │ Internet Gateway                         │
│                              ▼                                          │
│                          INTERNET                                       │
│                              ▲                                          │
│                              │ NAT Gateway                              │
│                              │                                          │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                      PRIVATE SUBNETS                               │ │
│  │  ┌─────────────────────┐     ┌─────────────────────┐              │ │
│  │  │  10.0.10.0/24       │     │  10.0.20.0/24       │              │ │
│  │  │  us-east-1a         │     │  us-east-1b         │              │ │
│  │  │                     │     │                     │              │ │
│  │  │  • ECS Tasks        │     │  • ECS Tasks        │              │ │
│  │  │  • RDS (primary)    │     │  • RDS (standby)    │              │ │
│  │  │  • Lambda           │     │  • Lambda           │              │ │
│  │  └─────────────────────┘     └─────────────────────┘              │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                    SECURITY GROUP (Default)                        │ │
│  │                                                                    │ │
│  │  Ingress:                                                          │ │
│  │    - Self (all traffic from same SG) ← Services communicate       │ │
│  │                                                                    │ │
│  │  Egress:                                                           │ │
│  │    - 0.0.0.0/0 (all outbound)                                     │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Service Communication

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    STACK: notification-platform                         │
│                                                                         │
│  ┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐ │
│  │   api service   │      │  worker service │      │scheduler service│ │
│  │                 │      │                 │      │                 │ │
│  │  ┌───────────┐  │      │  ┌───────────┐  │      │  ┌───────────┐  │ │
│  │  │ api-server│  │─────►│  │ processor │  │◄─────│  │ cron-job  │  │ │
│  │  │   (ECS)   │  │ SQS  │  │  (Lambda) │  │ Event│  │  (Lambda) │  │ │
│  │  └───────────┘  │      │  └───────────┘  │      │  └───────────┘  │ │
│  │       │         │      │       │         │      │                 │ │
│  │       ▼         │      │       ▼         │      │                 │ │
│  │  ┌───────────┐  │      │  ┌───────────┐  │      │                 │ │
│  │  │  api-db   │  │      │  │ sessions  │  │      │                 │ │
│  │  │   (RDS)   │  │      │  │ (DynamoDB)│  │      │                 │ │
│  │  └───────────┘  │      │  └───────────┘  │      │                 │ │
│  └─────────────────┘      └─────────────────┘      └─────────────────┘ │
│                                                                         │
│  All services use same VPC, can communicate via:                       │
│  - Direct TCP (same security group)                                     │
│  - SQS queues                                                           │
│  - SNS topics                                                           │
│  - DynamoDB streams                                                     │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## User Workflows

### Workflow 1: Platform Admin Sets Up Tenant

```bash
# 1. Admin logs in
panka admin login

# 2. Create tenant with networking
panka admin tenant init notifications-team \
  --aws-account 123456789012 \
  --region us-east-1 \
  --vpc-cidr 10.0.0.0/16 \
  --nat-gateway \
  --output credentials.txt

# Output:
# ✓ Tenant created: notifications-team
# ✓ VPC configured: 10.0.0.0/16
# ✓ Public subnets: 10.0.1.0/24, 10.0.2.0/24
# ✓ Private subnets: 10.0.10.0/24, 10.0.20.0/24
# ✓ NAT Gateway: enabled
# ✓ Security Group: notifications-team-default-sg
#
# Credentials saved to: credentials.txt

# 3. Share credentials with tenant team
cat credentials.txt
# Tenant ID: notifications-team
# Secret: <generated-secret>
```

### Workflow 2: Tenant User Creates Stack

```bash
# 1. Login as tenant
panka login
# Enter tenant ID: notifications-team
# Enter secret: ****

# 2. Create stack folder
panka stack init notification-platform --services api,worker,scheduler

# Creates:
# notification-platform/
# ├── stack.yaml
# └── services/
#     ├── api/
#     │   └── service.yaml
#     ├── worker/
#     │   └── service.yaml
#     └── scheduler/
#         └── service.yaml

# 3. Edit stack.yaml and services
cd notification-platform
vim stack.yaml
vim services/api/service.yaml
vim services/api/ecs.yaml
vim services/api/resources.yaml

# 4. Validate
panka validate .

# 5. Plan
panka plan .

# 6. Deploy
panka apply .
```

### Workflow 3: Add Service to Existing Stack

```bash
cd notification-platform

# Add new service
panka service add analytics

# Creates:
# services/analytics/
# └── service.yaml

# Edit service files
vim services/analytics/service.yaml
vim services/analytics/kinesis.yaml

# Deploy updated stack
panka apply .
```

---

## CLI Commands

### Admin Commands

```bash
# Login as admin
panka admin login

# Tenant management
panka admin tenant init <name> [flags]
panka admin tenant list
panka admin tenant show <name>
panka admin tenant update <name> --config <file>
panka admin tenant delete <name>
panka admin tenant rotate <name>
panka admin tenant suspend <name>
panka admin tenant activate <name>

# Flags for tenant init:
#   --aws-account      AWS account ID
#   --region           AWS region
#   --vpc-cidr         VPC CIDR block (default: 10.0.0.0/16)
#   --nat-gateway      Enable NAT gateway
#   --config           Config file for advanced settings
#   --output           Output credentials file
```

### User Commands

```bash
# Login as tenant
panka login
panka logout

# Stack management (folder-based)
panka stack init <name> [--services svc1,svc2]
panka stack list
panka stack info <folder>
panka validate <folder>
panka plan <folder>
panka apply <folder>
panka destroy <folder>

# Service management
panka service add <name> [--stack <folder>]
panka service remove <name> [--stack <folder>]
panka service list [--stack <folder>]

# State management
panka state list
panka state show <resource>
panka state remove <resource>

# Utilities
panka graph <folder> [--format ascii|dot|mermaid]
panka version
```

---

## State Management

### S3 State Structure

```
s3://panka-state-bucket/
├── tenants.yaml                           # Admin: tenant registry
└── tenants/
    └── notifications-team/
        ├── tenant.yaml                    # Tenant config (networking)
        └── v1/
            └── stacks/
                └── notification-platform/
                    └── production/
                        └── state.json     # Deployed resources
```

### State File Format

```json
{
  "version": "1.0.0",
  "metadata": {
    "stack": "notification-platform",
    "tenant": "notifications-team",
    "environment": "production",
    "deployedBy": "user@example.com",
    "deployedAt": "2024-11-28T12:00:00Z"
  },
  "networking": {
    "vpcId": "vpc-12345678",
    "subnetIds": {
      "public": ["subnet-pub1", "subnet-pub2"],
      "private": ["subnet-priv1", "subnet-priv2"]
    },
    "securityGroupId": "sg-12345678"
  },
  "resources": {
    "api-server": {
      "type": "AWS::ECS::Service",
      "status": "ready",
      "physicalId": "arn:aws:ecs:...",
      "attributes": {
        "clusterArn": "...",
        "serviceArn": "..."
      }
    },
    "api-db": {
      "type": "AWS::RDS::DBInstance",
      "status": "ready",
      "physicalId": "notification-platform-api-db",
      "attributes": {
        "endpoint": "...",
        "port": 5432
      }
    }
  },
  "outputs": {
    "api-server.endpoint": "https://...",
    "api-db.endpoint": "notification-platform-api-db.xxx.us-east-1.rds.amazonaws.com"
  }
}
```

---

## Implementation Phases

### Phase 1: Tenant Networking (Current)

- [ ] Update `pkg/tenant/types.go` with networking config
- [ ] Update `panka admin tenant init` command
- [ ] Store tenant config in S3
- [ ] Add networking validation

### Phase 2: Folder Parser

- [ ] Create `pkg/parser/folder_parser.go`
- [ ] Parse stack folder structure
- [ ] Parse service subfolders
- [ ] Merge tenant networking with stack overrides

### Phase 3: Schema Updates

- [ ] Update Stack schema for folder-based definition
- [ ] Add networking inheritance logic
- [ ] Update component schemas with networking refs

### Phase 4: CLI Updates

- [ ] Update `panka validate` for folders
- [ ] Update `panka plan` for folders
- [ ] Update `panka apply` for folders
- [ ] Add `panka stack init` command
- [ ] Add `panka service add` command

### Phase 5: AWS Networking Providers

- [ ] VPC provider
- [ ] Subnet provider
- [ ] Internet Gateway provider
- [ ] NAT Gateway provider
- [ ] Route Table provider
- [ ] Security Group provider

### Phase 6: Integration Testing

- [ ] End-to-end tenant setup
- [ ] Stack deployment with networking
- [ ] Service communication testing
- [ ] Multi-stack in same tenant

---

## Appendix: Complete Example

### Tenant Config

```yaml
# Created by admin, stored in S3
apiVersion: admin.panka.io/v1
kind: TenantConfig
metadata:
  name: notifications-team
spec:
  aws:
    accountId: "123456789012"
    region: us-east-1
  networking:
    vpc:
      cidrBlock: "10.0.0.0/16"
    subnets:
      public:
        - cidrBlock: "10.0.1.0/24"
          availabilityZone: us-east-1a
        - cidrBlock: "10.0.2.0/24"
          availabilityZone: us-east-1b
      private:
        - cidrBlock: "10.0.10.0/24"
          availabilityZone: us-east-1a
        - cidrBlock: "10.0.20.0/24"
          availabilityZone: us-east-1b
    natGateway:
      enabled: true
    defaultSecurityGroup:
      allowInternalTraffic: true
  limits:
    maxStacks: 10
  defaultTags:
    tenant: notifications-team
```

### Stack Folder

```
notification-platform/
├── stack.yaml
└── services/
    ├── api/
    │   ├── service.yaml
    │   ├── ecs.yaml
    │   └── resources.yaml
    └── worker/
        ├── service.yaml
        └── lambda.yaml
```

### Deployment Output

```
$ panka apply ./notification-platform

📦 Deploying Stack: notification-platform
   Tenant: notifications-team
   VPC: 10.0.0.0/16 (from tenant)

🔍 Analyzing changes...

Services: 2
  ├── api (3 components)
  └── worker (2 components)

Resources to create: 5
  ├── api-server (MicroService)
  ├── api-db (RDS)
  ├── notification-queue (SQS)
  ├── processor (Lambda)
  └── sessions-table (DynamoDB)

🚀 Deploying...
  ├── Creating api-db... ✓
  ├── Creating notification-queue... ✓
  ├── Creating sessions-table... ✓
  ├── Creating api-server... ✓
  └── Creating processor... ✓

✓ Stack deployed successfully!

Outputs:
  api-server.endpoint = https://api.notifications.example.com
  api-db.endpoint = notification-platform-api-db.xxx.rds.amazonaws.com
```

---

## Document History

| Version | Date | Changes |
|---------|------|---------|
| 2.0 | Dec 2024 | New architecture: Tenant → Stack → Service hierarchy |
| 1.0 | Nov 2024 | Initial single-file architecture |

---

**This document is the source of truth for Panka's architecture.**

