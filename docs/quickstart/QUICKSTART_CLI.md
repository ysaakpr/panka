# 🚀 Panka CLI Quick Start Guide

## Running the CLI

The Panka CLI binary is located at `bin/panka` after building.

### Build (if needed)
```bash
cd /Users/vyshakhp/work/panka
go build -o bin/panka ./cmd/panka
```

---

## Basic Commands

### 1. Get Help
```bash
./bin/panka --help
./bin/panka <command> --help
```

### 2. Show Version
```bash
./bin/panka version
```

**Output:**
```
Panka Version: 0.1.0-dev
Git Commit:    unknown
Build Date:    unknown
```

---

## Working with Infrastructure

### 3. Initialize a New Project
```bash
./bin/panka init
```

This creates:
- `.panka.yaml` - Configuration file
- `infrastructure.yaml` - Example infrastructure definition

**What it does:**
- Creates configuration with backend settings (S3, DynamoDB)
- Creates example YAML with S3, DynamoDB, SQS, SNS resources
- Shows next steps

### 4. Validate Your Configuration
```bash
./bin/panka validate infrastructure.yaml

# Or use the provided example:
./bin/panka validate examples/simple-stack.yaml
```

**What it checks:**
- ✅ YAML syntax
- ✅ Schema compliance
- ✅ Resource references
- ✅ Required fields

### 5. Visualize Dependencies
```bash
# ASCII output (default)
./bin/panka graph examples/simple-stack.yaml

# DOT format (for Graphviz)
./bin/panka graph examples/simple-stack.yaml --output dot --file graph.dot

# Mermaid format
./bin/panka graph examples/simple-stack.yaml --output mermaid --file graph.mmd
```

**Shows:**
- Dependency relationships
- Graph statistics (nodes, edges, cycles)
- Deployment order

### 6. Generate Deployment Plan
```bash
./bin/panka plan examples/simple-stack.yaml

# Detailed output
./bin/panka plan examples/simple-stack.yaml --detailed
```

**Shows:**
- What resources will be created
- Deployment stages (parallel execution)
- Dependency ordering
- Estimated duration

---

## State Management

### 7. List Resources in State
```bash
./bin/panka state list
```

**Shows:** All tracked resources with status

### 8. Show Resource Details
```bash
./bin/panka state show <resource-id>

# Example:
./bin/panka state show demo-stack.backend-api.uploads-bucket
```

**Shows:** Complete resource information in JSON format

### 9. Remove Resource from State
```bash
./bin/panka state rm <resource-id>

# Example (use with caution):
./bin/panka state rm demo-stack.backend-api.uploads-bucket
```

⚠️ **Warning:** This only removes from state, doesn't destroy the resource!

---

## Destruction

### 10. Plan Resource Destruction
```bash
# Dry-run (recommended first)
./bin/panka destroy examples/simple-stack.yaml --dry-run

# With confirmation prompt
./bin/panka destroy examples/simple-stack.yaml

# Skip confirmation (dangerous!)
./bin/panka destroy examples/simple-stack.yaml --auto-approve
```

**Shows:**
- Reverse dependency order
- What will be destroyed
- Requires confirmation (type stack name)

---

## Advanced Usage

### Global Flags

Available for all commands:

```bash
# Custom config file
./bin/panka --config /path/to/config.yaml <command>

# Change log level
./bin/panka --log-level debug <command>

# JSON logging
./bin/panka --log-format json <command>

# Verbose output
./bin/panka -v <command>

# Tenant mode
./bin/panka --tenant-mode --tenant-id my-tenant <command>
```

### Examples

**Debug mode:**
```bash
./bin/panka --log-level debug validate examples/simple-stack.yaml
```

**Multiple files:**
```bash
./bin/panka validate file1.yaml file2.yaml file3.yaml
```

**Save graph to file:**
```bash
./bin/panka graph infrastructure.yaml --output dot --file deployment.dot
# Then use Graphviz: dot -Tpng deployment.dot -o deployment.png
```

---

## Complete Workflow

### Typical Development Cycle:

```bash
# 1. Initialize (first time only)
./bin/panka init

# 2. Edit infrastructure.yaml
# (add your resources)

# 3. Validate
./bin/panka validate infrastructure.yaml

# 4. Check dependencies
./bin/panka graph infrastructure.yaml

# 5. Generate plan
./bin/panka plan infrastructure.yaml

# 6. Review the plan
# - Check stages
# - Verify dependencies
# - Review parallel execution

# 7. (Deploy - when apply is implemented)
# ./bin/panka apply infrastructure.yaml

# 8. Verify state
./bin/panka state list
./bin/panka state show <resource-id>

# 9. Later: destroy (with confirmation)
./bin/panka destroy infrastructure.yaml --dry-run
./bin/panka destroy infrastructure.yaml
```

---

## File Structure

Your project should look like:

```
your-project/
├── .panka.yaml              # Panka configuration
├── infrastructure.yaml       # Your resources
└── .gitignore               # Don't commit sensitive data
```

---

## Example Infrastructure File

Create `infrastructure.yaml`:

```yaml
---
apiVersion: panka.dev/v1
kind: Stack
metadata:
  name: my-app
spec:
  region: us-east-1
  variables:
    environment: production

---
apiVersion: panka.dev/v1
kind: S3
metadata:
  name: uploads
  stack: my-app
  service: backend
spec:
  bucket:
    acl: private
  versioning:
    enabled: true
  encryption:
    enabled: true
    algorithm: AES256

---
apiVersion: panka.dev/v1
kind: DynamoDB
metadata:
  name: sessions
  stack: my-app
  service: backend
spec:
  billing_mode: PAY_PER_REQUEST
  hash_key:
    name: sessionId
    type: S
  ttl:
    enabled: true
    attribute_name: expiresAt
```

Then run:
```bash
./bin/panka validate infrastructure.yaml
./bin/panka plan infrastructure.yaml
```

---

## Troubleshooting

### Command not found?
```bash
# Make sure you're in the project directory
cd /Users/vyshakhp/work/panka

# Use full path or relative path
./bin/panka --help
```

### Build errors?
```bash
# Rebuild
go build -o bin/panka ./cmd/panka

# Check for errors
go build -v -o bin/panka ./cmd/panka
```

### Want to install globally?
```bash
# Copy to your PATH
sudo cp bin/panka /usr/local/bin/

# Or add to PATH
export PATH=$PATH:/Users/vyshakhp/work/panka/bin

# Then use directly:
panka --help
```

---

## What Works Now

✅ **Fully Functional:**
- Configuration initialization
- YAML validation
- Dependency visualization
- Deployment planning
- State inspection
- Destruction planning

⚠️ **Not Yet Implemented:**
- Actual resource deployment (`apply` command)
- Real state backend (S3)
- Lock management (DynamoDB)

---

## Next Steps

1. **Try the commands** with `examples/simple-stack.yaml`
2. **Create your own** `infrastructure.yaml`
3. **Validate** your configuration
4. **Visualize** dependencies
5. **Generate plans** to see deployment order

---

## Tips

💡 **Always validate first:**
```bash
./bin/panka validate infrastructure.yaml
```

💡 **Use dry-run for destructive operations:**
```bash
./bin/panka destroy infrastructure.yaml --dry-run
```

💡 **Check dependencies before planning:**
```bash
./bin/panka graph infrastructure.yaml
```

💡 **Save important outputs:**
```bash
./bin/panka plan infrastructure.yaml > deployment-plan.txt
./bin/panka graph infrastructure.yaml --output mermaid --file graph.mmd
```

---

## Getting Help

```bash
# General help
./bin/panka --help

# Command-specific help
./bin/panka validate --help
./bin/panka plan --help
./bin/panka destroy --help
./bin/panka state --help
```

---

**🎉 You're ready to use Panka!**

Start with `./bin/panka validate examples/simple-stack.yaml` to see it in action!

