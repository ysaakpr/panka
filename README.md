# Deployer

A Golang-based deployment management system for managing application deployments on AWS using ECS/Fargate/EKS with Pulumi as the backend orchestrator.

## Overview

Deployer enables teams to deploy and manage their applications on AWS with a simple, declarative YAML-based configuration. It handles all the complexity of infrastructure provisioning, state management, and deployment orchestration.

### Key Features

- **Declarative Configuration**: Define your entire stack in YAML files
- **Stack-Based Deployments**: Deploy entire environments or individual services/components
- **Distributed Locking**: Prevent conflicting deployments with DynamoDB-backed locking
- **State Management**: Track deployment state with S3-backed versioned storage
- **Dependency Management**: Automatic resolution and ordering of component dependencies
- **Multiple Environments**: Easy environment promotion (dev → staging → production)
- **Drift Detection**: Detect and remediate configuration drift
- **Automatic Rollback**: Rollback on failures or metric thresholds
- **Policy Enforcement**: Security, cost, and compliance policies with OPA
- **Comprehensive Observability**: Built-in logging, metrics, and alerting

## Quick Start

### Prerequisites

- AWS Account with appropriate permissions
- AWS CLI configured
- Docker (for building container images)
- Git

### Installation

```bash
# Install deployer CLI
curl -sSL https://deployer.io/install.sh | sh

# Verify installation
deployer --version
```

### Initial Setup

```bash
# Clone your deployment repository
git clone git@github.com:company/deployment-repo.git
cd deployment-repo

# Validate your stack configuration
deployer validate --stack user-platform --environment development

# Preview deployment
deployer plan --stack user-platform --environment development --var VERSION=v1.0.0

# Deploy
deployer apply --stack user-platform --environment development --var VERSION=v1.0.0
```

## Core Concepts

### Stack
A **stack** is the unit of deployment representing a complete environment (production, staging, development) containing multiple services.

```
Stack: user-platform
├── Services
│   ├── user-service
│   ├── auth-service
│   └── notification-service
└── Environments
    ├── production
    ├── staging
    └── development
```

### Service
A **service** is a logical grouping of related components (e.g., an API, its database, cache, and workers).

```
Service: user-service
├── Components
│   ├── api (MicroService)
│   ├── worker (Worker)
│   ├── database (RDS)
│   ├── cache (ElastiCacheRedis)
│   └── queue (SQS)
```

### Component
A **component** is a single deployable unit:
- **Compute**: MicroService, Worker, CronJob, Lambda, EC2Instance
- **Database**: RDS, DynamoDB, DocumentDB
- **Cache**: ElastiCacheRedis, ElastiCacheMemcached, AWSMemoryDB
- **Storage**: S3, EFS, EBS
- **Messaging**: SQS, SNS, Kafka, MSK, EventBridge
- **Networking**: ALB, NLB, CloudFront, APIGateway

## Repository Structure

```
deployment-repo/
├── stacks/
│   └── user-platform/
│       ├── stack.yaml                    # Stack definition
│       ├── infra/                        # Infrastructure rules
│       │   ├── defaults.yaml
│       │   ├── networking.yaml
│       │   ├── security.yaml
│       │   └── observability.yaml
│       │
│       ├── services/
│       │   └── user-service/
│       │       ├── service.yaml          # Service definition
│       │       ├── infra/                # Service-level infra
│       │       └── components/
│       │           ├── api/
│       │           │   ├── microservice.yaml  # Component definition
│       │           │   ├── infra.yaml         # Infra config
│       │           │   └── configs/           # App configs
│       │           └── database/
│       │               └── rds.yaml
│       │
│       └── environments/                 # Environment overrides
│           ├── production/
│           ├── staging/
│           └── development/
│
├── templates/                            # Reusable templates
├── policies/                             # OPA policies
└── docs/                                 # Documentation
```

## Example: Deploy a New Service

### 1. Define Your Service

Create `stacks/user-platform/services/notification-service/service.yaml`:

```yaml
apiVersion: core.deployer.io/v1
kind: Service

metadata:
  name: notification-service
  stack: user-platform
  description: "Email and SMS notification service"
  
  labels:
    team: notifications

spec:
  infrastructure:
    defaults: ./infra/defaults.yaml
```

### 2. Define Components

Create `stacks/user-platform/services/notification-service/components/api/microservice.yaml`:

```yaml
apiVersion: components.deployer.io/v1
kind: MicroService

metadata:
  name: api
  service: notification-service
  stack: user-platform

spec:
  image:
    repository: 123456789012.dkr.ecr.us-east-1.amazonaws.com/notification-api
    tag: "${VERSION}"
  
  runtime:
    platform: fargate
  
  ports:
    - name: http
      port: 8080
  
  environment:
    - name: DATABASE_HOST
      valueFrom:
        component: notification-service/database
        output: endpoint
  
  secrets:
    - name: DB_PASSWORD
      secretRef: /stacks/user-platform/notification-service/db-password
      envVar: DATABASE_PASSWORD
  
  healthCheck:
    readiness:
      http:
        path: /health/ready
        port: 8080
  
  dependsOn:
    - notification-service/database
```

Create `stacks/user-platform/services/notification-service/components/api/infra.yaml`:

```yaml
apiVersion: infra.deployer.io/v1
kind: ComponentInfra

metadata:
  name: api
  service: notification-service
  stack: user-platform

spec:
  resources:
    cpu: 256
    memory: 512
  
  scaling:
    replicas: 2
    autoscaling:
      enabled: true
      minReplicas: 2
      maxReplicas: 10
  
  networking:
    loadBalancer:
      enabled: true
```

Create `stacks/user-platform/services/notification-service/components/database/rds.yaml`:

```yaml
apiVersion: components.deployer.io/v1
kind: RDS

metadata:
  name: database
  service: notification-service
  stack: user-platform

spec:
  engine:
    type: postgres
    version: "15.4"
  
  instance:
    class: db.t3.medium
    storage:
      type: gp3
      allocatedGB: 50
  
  database:
    name: notificationdb
    username: dbadmin
    passwordSecret:
      ref: /stacks/user-platform/notification-service/db-password
```

### 3. Deploy

```bash
# Validate
deployer validate --stack user-platform --service notification-service

# Plan (dry-run)
deployer plan \
  --stack user-platform \
  --service notification-service \
  --environment development \
  --var VERSION=v1.0.0

# Deploy
deployer apply \
  --stack user-platform \
  --service notification-service \
  --environment development \
  --var VERSION=v1.0.0
```

### 4. Monitor

```bash
# Check status
deployer status --service notification-service --environment development

# View logs
deployer logs --component notification-service/api --environment development --follow

# View metrics
deployer metrics --component notification-service/api --environment development
```

## CLI Commands

```bash
# Deployment
deployer apply        # Deploy stack/service/component
deployer plan         # Show execution plan (dry-run)
deployer destroy      # Destroy stack/service/component

# Validation
deployer validate     # Validate configuration
deployer graph        # Visualize dependency graph

# Status & Information
deployer status       # Show deployment status
deployer list         # List all resources
deployer show         # Show resource details
deployer history      # Show deployment history

# Logs & Metrics
deployer logs         # View logs
deployer metrics      # View metrics

# Drift Management
deployer drift detect    # Detect configuration drift
deployer drift remediate # Fix drift

# Rollback
deployer rollback     # Rollback to previous version

# State Management
deployer state show   # Show current state
deployer state locks  # Show active locks
deployer unlock       # Unlock stuck deployment
```

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      DEPLOYER CLI                            │
│                                                              │
│  Discovery → Reconciler → Executor                          │
│     ↓            ↓            ↓                             │
│  Parser      State Mgr    Pulumi Backend                    │
└─────────────────────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────┐
│                    AWS Services                              │
│                                                              │
│  • S3 (State)           • ECS/Fargate    • RDS             │
│  • DynamoDB (Locks)     • ElastiCache    • S3              │
│  • Secrets Manager      • SQS            • ALB             │
└─────────────────────────────────────────────────────────────┘
```

### State Management

- **State Storage**: S3 with versioning
- **Locking**: DynamoDB with atomic conditional writes
- **TTL**: Automatic cleanup of expired locks
- **Heartbeats**: Keep-alive mechanism for long deployments

### Execution Flow

```
1. Discovery Phase
   ├── Parse YAML files
   ├── Apply environment overlays
   ├── Resolve variables
   └── Validate schemas

2. Dependency Resolution
   ├── Build dependency graph
   ├── Detect cycles
   ├── Topological sort
   └── Generate deployment waves

3. State Reconciliation
   ├── Acquire distributed lock
   ├── Load current state
   ├── Compute diff
   ├── Generate execution plan
   └── Get approval

4. Execution
   ├── Execute waves in order
   ├── Deploy resources in parallel within wave
   ├── Run health checks
   ├── Update state
   └── Release lock

5. Verification
   ├── Run smoke tests
   ├── Monitor metrics
   └── Auto-rollback on failure
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) - System architecture and design
- [Implementation Plan](docs/IMPLEMENTATION_PLAN.md) - Development roadmap
- [User Workflows](docs/USER_WORKFLOWS.md) - Guide for application teams
- [State & Locking](docs/STATE_AND_LOCKING.md) - State management details

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

[MIT License](LICENSE)

## Support

- Documentation: https://docs.company.com/deployer
- Slack: #deployer-help
- Email: platform-team@company.com
- Issues: https://github.com/company/deployer/issues

## Roadmap

### Phase 1: MVP (Weeks 1-8)
- [x] Project setup
- [ ] State management (S3)
- [ ] Distributed locking (DynamoDB)
- [ ] YAML parser
- [ ] Basic component support (ECS, RDS, S3)
- [ ] Pulumi integration
- [ ] CLI basics

### Phase 2: Core Features (Weeks 9-12)
- [ ] All component types
- [ ] Drift detection
- [ ] Policy validation
- [ ] Rollback system
- [ ] Comprehensive testing

### Phase 3: Advanced Features (Weeks 13-16)
- [ ] Multi-region support
- [ ] Advanced autoscaling
- [ ] Cost optimization
- [ ] Performance tuning

### Phase 4: GA (Weeks 17-18)
- [ ] Documentation
- [ ] Production hardening
- [ ] Security audit
- [ ] Performance benchmarks

## Quick Links

- [Getting Started](docs/USER_WORKFLOWS.md#quick-start)
- [Common Workflows](docs/USER_WORKFLOWS.md#common-workflows)
- [Troubleshooting](docs/USER_WORKFLOWS.md#troubleshooting)
- [Best Practices](docs/USER_WORKFLOWS.md#best-practices)

---

**Built with ❤️ by the Platform Team**

