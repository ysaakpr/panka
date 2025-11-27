# Panka - Phase 2 Complete 🎉

## What is Panka?

Panka is a **modern, multi-tenant AWS deployment orchestration tool** that uses declarative YAML configurations to manage complex cloud infrastructures. Think of it as a simplified, opinionated alternative to Terraform specifically designed for multi-tenant AWS environments.

## Development Status

### ✅ Phase 1: Foundation (COMPLETE)
- Go project setup with modern tooling
- Structured logging with zap
- Configuration management
- S3 state backend
- DynamoDB lock manager
- CI/CD pipeline with GitHub Actions
- Comprehensive test suite

### ✅ Phase 2: YAML Parser & Validator (COMPLETE)
- **Schema definitions** for 10+ AWS resource types
- **Multi-document YAML parser** with variable interpolation
- **Comprehensive validator** with circular dependency detection
- **29 passing tests** for parser and validator
- **Example configurations** demonstrating all features

**Total Tests: 70+ tests across all packages**

### 🚧 Coming Next: Phase 3 - Resource Discovery & Graph Building

## Quick Example

```yaml
# Define your stack
---
apiVersion: core.panka.io/v1
kind: Stack
metadata:
  name: my-app
spec:
  provider:
    name: aws
    region: us-east-1
  variables:
    VERSION: "1.0.0"

---
# Define a service
apiVersion: core.panka.io/v1
kind: Service
metadata:
  name: backend
  stack: my-app

---
# Define a microservice
apiVersion: components.panka.io/v1
kind: MicroService
metadata:
  name: api
  service: backend
  stack: my-app
spec:
  image:
    repository: myrepo/api
    tag: ${VERSION}
  runtime:
    platform: fargate
  ports:
    - name: http
      port: 8080
  environment:
    - name: DB_HOST
      valueFrom:
        component: database
        output: endpoint
  dependsOn:
    - database

---
# Define a database
apiVersion: components.panka.io/v1
kind: RDS
metadata:
  name: database
  service: backend
  stack: my-app
spec:
  engine:
    type: postgres
    version: "14.7"
  instance:
    class: db.t3.medium
    storage:
      type: gp3
      allocatedGB: 100
  database:
    name: appdb
    username: admin
    passwordSecret:
      ref: prod/db/password
```

## Supported Resource Types

### Core Resources
- **Stack**: Top-level deployment unit
- **Service**: Logical grouping of components

### Compute Components
- **MicroService**: Containerized services (Fargate, EC2, Lambda)
- Worker, CronJob, Lambda (schema defined, implementation coming)

### Database Components
- **RDS**: PostgreSQL, MySQL, MariaDB, Aurora
- **DynamoDB**: NoSQL tables with GSI, TTL, encryption

### Storage Components
- **S3**: Object storage with lifecycle, versioning, CORS
- EFS, EBS (schema defined, implementation coming)

### Messaging Components
- **SQS**: Standard and FIFO queues with DLQ
- **SNS**: Topics and subscriptions
- Kafka, MSK, EventBridge (schema defined, implementation coming)

### Networking Components
- ALB, NLB, CloudFront, API Gateway (schema defined, implementation coming)

## Key Features

### 1. Variable Interpolation
```yaml
# Stack-level variables
variables:
  VERSION: "1.0.0"
  REGION: "us-east-1"

# Service-level variables
spec:
  variables:
    IMAGE_REPO: "myregistry/backend"

# Usage anywhere
image:
  repository: ${backend.IMAGE_REPO}
  tag: ${VERSION}
```

### 2. Component Cross-References
```yaml
environment:
  - name: DATABASE_URL
    valueFrom:
      component: main-db
      output: endpoint
```

### 3. Dependency Management
```yaml
dependsOn:
  - database
  - cache
  - queue
```

Automatically detects circular dependencies and enforces deployment order!

### 4. Multi-Tenant Support
```yaml
# Stack gets deployed per tenant
# State: s3://bucket/tenants/{tenant-id}/stacks/{stack-name}/state.json
# Locks: DynamoDB table with tenant-id as partition key
```

## Parser API

### Parse YAML Configuration
```go
import "github.com/yourusername/panka/pkg/parser"

parser := parser.NewParser()
result, err := parser.ParseFile("stack.yaml")
if err != nil {
    log.Fatal(err)
}

// Access parsed resources
fmt.Printf("Stack: %s\n", result.Stack.Metadata.Name)
fmt.Printf("Services: %d\n", len(result.Services))
fmt.Printf("Components: %d\n", len(result.Components))
```

### Validate Configuration
```go
validator := parser.NewValidator()
if err := validator.Validate(result); err != nil {
    log.Fatal(err)
}
```

### Variable Interpolation
```go
// Set variables programmatically
parser.SetVariable("ENVIRONMENT", "production")
parser.SetVariable("VERSION", "2.0.0")

// Set component outputs (for cross-references)
parser.SetComponentOutput("database", "endpoint", "db.example.com:5432")
```

## Development

### Prerequisites
- Go 1.21+
- Docker (for LocalStack integration tests)
- AWS CLI (for deployment)

### Quick Start
```bash
# Clone the repository
git clone https://github.com/yourusername/panka
cd panka

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run with example
./bin/panka examples/simple-stack.yaml
```

### Available Make Targets
```bash
make build              # Build the binary
make test               # Run all tests
make test-unit          # Run unit tests only
make test-integration   # Run integration tests (requires LocalStack)
make lint               # Run golangci-lint
make fmt                # Format code
make clean              # Clean build artifacts
make dev                # Development mode (build + run)
make docker-up          # Start LocalStack
make docker-down        # Stop LocalStack
```

## Project Structure
```
panka/
├── cmd/panka/              # CLI entry point
├── internal/
│   └── logger/             # Structured logging
├── pkg/
│   ├── config/             # Configuration management
│   ├── lock/               # DynamoDB lock manager
│   ├── parser/             # YAML parser & validator
│   │   └── schema/         # Resource schemas
│   └── state/              # S3 state backend
├── examples/               # Example configurations
├── docs/                   # Documentation
└── test/                   # Integration tests
```

## Testing

**70+ tests** across all packages with comprehensive coverage:

- ✅ Logger: 8 tests
- ✅ Config: 14 tests  
- ✅ State: 25 tests
- ✅ Lock: 7 tests
- ✅ **Parser: 29 tests** (NEW in Phase 2)

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run specific package
go test ./pkg/parser/... -v
```

## AI-Assisted Development

This project is being developed with **AI assistance** (Claude), achieving **2-3x development speed**:

- ✅ Phase 1: 4 hours (vs 8-12 hours manually)
- ✅ Phase 2: 2 hours (vs 6-8 hours manually)

See [`docs/AI_AGENT_DEVELOPMENT_GUIDE.md`](docs/AI_AGENT_DEVELOPMENT_GUIDE.md) for our AI development methodology.

## Contributing

We welcome contributions! Please see:
- [`CONTRIBUTING.md`](CONTRIBUTING.md) - Contribution guidelines
- [`docs/IMPLEMENTATION_PLAN.md`](docs/IMPLEMENTATION_PLAN.md) - Development roadmap
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) - System architecture

## License

MIT License - see [`LICENSE`](LICENSE) for details

## Roadmap

### Phase 3: Resource Discovery & Graph Building
- Dependency graph construction
- Topological sorting
- Deployment plan generation

### Phase 4: AWS Provider Implementation
- ECS/Fargate provisioning
- RDS instance creation
- S3 bucket management
- DynamoDB table creation
- SQS/SNS setup

### Phase 5: Deployment Engine
- Resource lifecycle management
- State tracking
- Rollback support
- Progress reporting

### Phase 6: Multi-Tenancy
- Tenant isolation
- Resource tagging
- State segregation
- Access control

### Phase 7: CLI & UX
- Interactive mode
- Plan/apply workflow
- Colorized output
- Progress indicators

### Phase 8: Integration & Testing
- End-to-end tests
- Performance testing
- Documentation
- Production readiness

---

**Current Status**: Phase 2 Complete ✅ | **Next**: Phase 3 Development 🚀

