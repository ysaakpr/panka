# How Development Teams Use Panka

A visual guide showing exactly how teams use the panka CLI tool.

---

## The Complete Picture

```
┌───────────────────────────────────────────────────────────────────────┐
│                                                                         │
│                         NOTIFICATIONS TEAM                              │
│                                                                         │
│  Team Members: Alice (Lead), Bob (Backend), Carol (DevOps)            │
│  Service: email-service                                                │
│  Goal: Deploy email notification service to production                │
│                                                                         │
└───────────────────────────────────────────────────────────────────────┘
```

---

## Timeline: From Zero to Production

### Day 0: Platform Team Setup (Before Teams Start)

**Platform Team Creates Shared Infrastructure:**

```bash
# Create S3 bucket and DynamoDB table
terraform apply

# Share configuration
📧 Email to all teams:
   S3 Bucket: company-panka-state
   DynamoDB Table: company-panka-locks
   Region: us-east-1
```

---

### Day 1, 9:00 AM - Alice: Install and Configure

```bash
# Alice installs CLI
alice@laptop:~$ curl -sSL https://panka.io/install.sh | sh
Downloading panka v1.0.0...
✓ Installed to /usr/local/bin/panka

alice@laptop:~$ panka version
panka version 1.0.0

# Alice configures backend
alice@laptop:~$ panka init

Welcome to Panka!

? AWS Region: us-east-1
? S3 Bucket for state: company-panka-state
? DynamoDB Table for locks: company-panka-locks
? AWS Profile (press Enter for default): 

✓ Configuration saved to /home/alice/.panka/config.yaml

# Verify configuration
alice@laptop:~$ cat ~/.panka/config.yaml
version: v1
backend:
  type: s3
  region: us-east-1
  bucket: company-panka-state
locks:
  type: dynamodb
  region: us-east-1
  table: company-panka-locks
aws:
  profile: default
  region: us-east-1
```

---

### Day 1, 10:00 AM - Alice: Create Stack

```bash
# Clone deployment repository
alice@laptop:~$ git clone git@github.com:company/deployment-repo.git
alice@laptop:~$ cd deployment-repo

# Create stack for notification platform
alice@laptop:~/deployment-repo$ mkdir -p stacks/notification-platform
alice@laptop:~/deployment-repo$ cd stacks/notification-platform

# Initialize stack
alice@laptop:~/deployment-repo/stacks/notification-platform$ panka stack init

Creating new stack...
? Stack name: notification-platform
? Description: Email and SMS notification services
? Team: notifications

✓ Created stack.yaml
✓ Created infra/ directory
✓ Created services/ directory
✓ Created environments/ directory

# Edit stack.yaml
alice@laptop:~/deployment-repo/stacks/notification-platform$ vim stack.yaml
```

**stack.yaml:**
```yaml
apiVersion: core.panka.io/v1
kind: Stack

metadata:
  name: notification-platform
  description: "Email and SMS notification services"
  
  labels:
    team: notifications
  
  annotations:
    owner: "notifications-team@company.com"
    slack: "#notifications-team"

spec:
  provider:
    name: aws
    region: us-east-1
```

---

### Day 1, 11:00 AM - Alice: Define Email Service

```bash
# Create service structure
alice@laptop:~/deployment-repo/stacks/notification-platform$ \
  mkdir -p services/email-service/components/{api,database,queue}

# Create service definition
alice@laptop:...$ cat > services/email-service/service.yaml << 'EOF'
apiVersion: core.panka.io/v1
kind: Service

metadata:
  name: email-service
  stack: notification-platform
  description: "Email notification service"

spec:
  infrastructure:
    defaults: ./infra/defaults.yaml
EOF

# Create API component
alice@laptop:...$ cat > services/email-service/components/api/microservice.yaml << 'EOF'
apiVersion: components.panka.io/v1
kind: MicroService

metadata:
  name: api
  service: email-service
  stack: notification-platform

spec:
  image:
    repository: 123456789012.dkr.ecr.us-east-1.amazonaws.com/email-api
    tag: "${VERSION}"
  
  runtime:
    platform: fargate
  
  ports:
    - name: http
      port: 8080
  
  environment:
    - name: DATABASE_HOST
      valueFrom:
        component: email-service/database
        output: endpoint
    
    - name: QUEUE_URL
      valueFrom:
        component: email-service/queue
        output: url
  
  secrets:
    - name: DB_PASSWORD
      secretRef: /stacks/notification-platform/email-service/db-password
      envVar: DATABASE_PASSWORD
  
  configs:
    mountPath: /config
    files:
      - app.yaml
  
  healthCheck:
    readiness:
      http:
        path: /health/ready
        port: 8080
  
  dependsOn:
    - email-service/database
    - email-service/queue
EOF

# Create infrastructure config
alice@laptop:...$ cat > services/email-service/components/api/infra.yaml << 'EOF'
apiVersion: infra.panka.io/v1
kind: ComponentInfra

metadata:
  name: api
  service: email-service
  stack: notification-platform

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
EOF

# Create app config
alice@laptop:...$ mkdir -p services/email-service/components/api/configs
alice@laptop:...$ cat > services/email-service/components/api/configs/app.yaml << 'EOF'
app:
  name: email-api

server:
  port: 8080
  timeout: 30s

email:
  provider: smtp
  from: noreply@company.com
  maxRetries: 3
EOF

# Create database component
alice@laptop:...$ cat > services/email-service/components/database/rds.yaml << 'EOF'
apiVersion: components.panka.io/v1
kind: RDS

metadata:
  name: database
  service: email-service
  stack: notification-platform

spec:
  engine:
    type: postgres
    version: "15.4"
  
  instance:
    class: db.t3.small
    storage:
      type: gp3
      allocatedGB: 20
  
  database:
    name: emaildb
    username: dbadmin
    passwordSecret:
      ref: /stacks/notification-platform/email-service/db-password
EOF

# Create queue component
alice@laptop:...$ cat > services/email-service/components/queue/sqs.yaml << 'EOF'
apiVersion: components.panka.io/v1
kind: SQS

metadata:
  name: queue
  service: email-service
  stack: notification-platform

spec:
  type: standard
  messageRetentionPeriod: 345600
  visibilityTimeout: 300
  
  deadLetterQueue:
    enabled: true
    maxReceiveCount: 3
EOF
```

---

### Day 1, 2:00 PM - Alice: Validate Configuration

```bash
alice@laptop:~/deployment-repo$ panka validate --stack notification-platform

Validating stack: notification-platform
├── Parsing stack.yaml... ✓
├── Parsing services... ✓
│   └── email-service ✓
├── Parsing components... ✓
│   ├── email-service/api ✓
│   ├── email-service/database ✓
│   └── email-service/queue ✓
├── Building dependency graph... ✓
│   Wave 1: database, queue
│   Wave 2: api
├── Validating references... ✓
├── Checking for cycles... ✓
└── Running policy checks... ✓

✓ Stack configuration is valid!
```

---

### Day 1, 3:00 PM - Bob: Build Application

```bash
# Bob builds the email service application
bob@laptop:~/work/email-service$ docker build -t email-api:v1.0.0 .
[+] Building 45.2s
 => [1/5] FROM node:18-alpine
 => [2/5] WORKDIR /app
 => [3/5] COPY package*.json ./
 => [4/5] RUN npm install
 => [5/5] COPY . .
 => exporting to image

bob@laptop:~/work/email-service$ docker tag email-api:v1.0.0 \
  123456789012.dkr.ecr.us-east-1.amazonaws.com/email-api:v1.0.0

# Login to ECR
bob@laptop:~/work/email-service$ aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  123456789012.dkr.ecr.us-east-1.amazonaws.com

bob@laptop:~/work/email-service$ docker push \
  123456789012.dkr.ecr.us-east-1.amazonaws.com/email-api:v1.0.0

The push refers to repository [123456789012.dkr.ecr.us-east-1.amazonaws.com/email-api]
v1.0.0: digest: sha256:abc123... size: 2415
```

---

### Day 1, 4:00 PM - Alice: Create Secrets

```bash
# Create database password secret
alice@laptop:~$ aws secretsmanager create-secret \
  --name /stacks/notification-platform/email-service/db-password \
  --secret-string '{"password":"super-secure-db-password-12345"}' \
  --region us-east-1

{
  "ARN": "arn:aws:secretsmanager:us-east-1:123456789012:secret:/stacks/notification-platform/email-service/db-password-abc123",
  "Name": "/stacks/notification-platform/email-service/db-password"
}

# Create SMTP password secret
alice@laptop:~$ aws secretsmanager create-secret \
  --name /stacks/notification-platform/email-service/smtp-password \
  --secret-string '{"password":"smtp-password-67890"}' \
  --region us-east-1
```

---

### Day 1, 4:30 PM - Alice: First Deployment to Dev

```bash
alice@laptop:~/deployment-repo$ panka plan \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.0

Panka v1.0.0
────────────────────────────────────────────────

Stack: notification-platform
Environment: development
Version: v1.0.0

Loading configuration...
├── Stack: notification-platform ✓
├── Services: 1 found ✓
├── Components: 3 found ✓
└── Building dependency graph... ✓

Checking current state...
├── Connecting to S3: company-panka-state ✓
├── Loading state... (not found - first deployment)
└── This is a new deployment ✓

Generating plan...

┌─────────────────────────────────────────────────────────┐
│ Deployment Plan: notification-platform (development)    │
├─────────────────────────────────────────────────────────┤
│                                                          │
│ Wave 1 (2 resources, parallel):                         │
│   + email-service/database (RDS)         CREATE         │
│     - Engine: postgres 15.4                             │
│     - Instance: db.t3.small                             │
│     - Storage: 20 GB                                    │
│                                                          │
│   + email-service/queue (SQS)            CREATE         │
│     - Type: standard                                    │
│     - DLQ: enabled                                      │
│                                                          │
│ Wave 2 (after wave 1 completes):                        │
│   + email-service/api (MicroService)     CREATE         │
│     - Image: email-api:v1.0.0                          │
│     - CPU: 256, Memory: 512                            │
│     - Replicas: 2                                       │
│     - Load Balancer: enabled                            │
│                                                          │
├─────────────────────────────────────────────────────────┤
│ Summary:                                                 │
│   + 3 to create                                          │
│   ✓ 0 to update                                          │
│   - 0 to delete                                          │
│                                                          │
│ Estimated duration: ~8 minutes                           │
│ Estimated cost: $45/month                                │
│                                                          │
│ ✓ No dangerous operations                               │
└─────────────────────────────────────────────────────────┘

# Now actually deploy
alice@laptop:~/deployment-repo$ panka apply \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.0

Panka v1.0.0
────────────────────────────────────────────────

Acquiring lock...
├── Lock key: stack:notification-platform:env:development
├── Attempting to acquire... ✓
└── Lock acquired (ID: 550e8400-e29b-41d4...)

Loading state...
└── State not found (first deployment)

Executing deployment...

Wave 1: Dependencies (2 resources, parallel)
├── [1/2] Creating email-service/database (RDS)
│   ├── Creating DB subnet group... ✓ (15s)
│   ├── Creating parameter group... ✓ (8s)
│   ├── Creating RDS instance... ⟳ (this may take 5-10 minutes)
│   │   ├── Provisioning... 30% [=========>                    ]
│   │   ├── Provisioning... 60% [==================>           ]
│   │   ├── Provisioning... 90% [===========================>  ]
│   │   └── Available ✓ (5m 23s)
│   └── ✓ Database created (5m 46s)
│       Endpoint: email-db-dev.abc123.us-east-1.rds.amazonaws.com
│
└── [2/2] Creating email-service/queue (SQS)
    ├── Creating queue... ✓ (3s)
    ├── Creating DLQ... ✓ (2s)
    └── ✓ Queue created (8s)
        URL: https://sqs.us-east-1.amazonaws.com/123456789012/email-queue-dev

Wave 2: Application (1 resource)
└── [1/1] Creating email-service/api (MicroService)
    ├── Creating task definition... ✓ (5s)
    ├── Creating target group... ✓ (10s)
    ├── Creating load balancer... ✓ (1m 45s)
    ├── Creating ECS service... ✓ (30s)
    ├── Waiting for tasks to start... ⟳
    │   ├── Task 1: PROVISIONING → PENDING → RUNNING ✓
    │   └── Task 2: PROVISIONING → PENDING → RUNNING ✓
    ├── Registering with load balancer... ✓ (30s)
    ├── Running health checks... ⟳
    │   ├── Attempt 1: healthy ✓
    │   └── Attempt 2: healthy ✓
    └── ✓ Service created (2m 50s)
        URL: http://dev-email-api.company.internal:8080

Finalizing...
├── Saving state to S3... ✓
├── Releasing lock... ✓
└── Deployment complete! ✓

────────────────────────────────────────────────
✓ Deployment successful!

Duration: 8m 44s
Version: v1.0.0
Deployed by: alice@company.com

Resources created:
  • email-service/database (RDS)
  • email-service/queue (SQS)  
  • email-service/api (MicroService)

Outputs:
  api_url: https://dev-email-api.company.com
  database_endpoint: email-db-dev.abc123.us-east-1.rds.amazonaws.com

Next steps:
  • Test: curl https://dev-email-api.company.com/health
  • Logs: panka logs --component email-service/api --follow
  • Status: panka status --stack notification-platform
────────────────────────────────────────────────
```

---

### Day 1, 5:00 PM - Bob: Verify and Test

```bash
# Bob tests the deployed service
bob@laptop:~$ panka status \
  --stack notification-platform \
  --environment development

┌──────────────────────────────────────────────────────────┐
│ Stack: notification-platform (development)               │
├──────────────────────────────────────────────────────────┤
│ Service: email-service                                   │
│   ✓ api        MicroService    2/2 running    Healthy   │
│   ✓ database   RDS             available      Healthy   │
│   ✓ queue      SQS             active         Healthy   │
│                                                          │
│ Last deployed: 15 minutes ago                            │
│ Version: v1.0.0                                          │
│ Deployed by: alice@company.com                           │
└──────────────────────────────────────────────────────────┘

# Check logs
bob@laptop:~$ panka logs \
  --component email-service/api \
  --environment development \
  --follow

2024-01-15 17:05:23 INFO Starting email-api v1.0.0
2024-01-15 17:05:24 INFO Connected to database
2024-01-15 17:05:24 INFO Connected to queue
2024-01-15 17:05:24 INFO Server listening on :8080

# Test API
bob@laptop:~$ curl https://dev-email-api.company.com/health
{"status":"healthy","database":"connected","queue":"connected"}

bob@laptop:~$ curl -X POST https://dev-email-api.company.com/send \
  -H "Content-Type: application/json" \
  -d '{"to":"bob@company.com","subject":"Test","body":"Hello"}'
{"status":"queued","id":"msg-abc123"}
```

---

### Day 2, 10:00 AM - Alice: Deploy to Staging

```bash
# Commit configuration to Git
alice@laptop:~/deployment-repo$ git add stacks/notification-platform/
alice@laptop:~/deployment-repo$ git commit -m "Add notification platform stack"
alice@laptop:~/deployment-repo$ git push origin main

# Deploy to staging
alice@laptop:~/deployment-repo$ panka apply \
  --stack notification-platform \
  --environment staging \
  --var VERSION=v1.0.0

# Similar output...
# ✓ Deployment successful! (8m 12s)
```

---

### Day 3, 2:00 PM - Alice: Deploy to Production

```bash
alice@laptop:~/deployment-repo$ panka apply \
  --stack notification-platform \
  --environment production \
  --var VERSION=v1.0.0

⚠ Production Deployment Approval Required

Stack: notification-platform
Environment: production
Version: v1.0.0

This will create:
  • email-service/database (RDS) - db.t3.medium, 50GB
  • email-service/queue (SQS)
  • email-service/api (MicroService) - 3 replicas

Estimated cost: $245/month

❗ This is a production deployment. Please review carefully.

Approve this deployment? (yes/no): yes

# Deployment proceeds...
# ✓ Deployment successful! (10m 05s)
```

---

### Week 2: Bob Updates the Service

```bash
# Bob fixed a bug, built v1.0.1
bob@laptop:~/deployment-repo$ panka apply \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.0.1

Panka detects:
  ✓ Only image tag changed: v1.0.0 → v1.0.1
  
Rolling update:
  ├── Starting new task with v1.0.1... ✓
  ├── Health check passing... ✓
  ├── Draining old task... ✓
  └── Update complete ✓

✓ Deployment successful! (3m 15s)
```

---

### Week 3: Carol Adds Cache

```bash
# Carol adds Redis cache
carol@laptop:~/deployment-repo$ cat > \
  stacks/notification-platform/services/email-service/components/cache/elasticache.yaml << 'EOF'
apiVersion: components.panka.io/v1
kind: ElastiCacheRedis

metadata:
  name: cache
  service: email-service
  stack: notification-platform

spec:
  engine:
    version: "7.0"
  cluster:
    mode: replication-group
    nodeType: cache.t3.micro
    numNodes: 2
EOF

# Update API to use cache
carol@laptop:~/deployment-repo$ vim \
  stacks/notification-platform/services/email-service/components/api/microservice.yaml

# Add:
environment:
  - name: REDIS_HOST
    valueFrom:
      component: email-service/cache
      output: endpoint

dependsOn:
  - email-service/database
  - email-service/queue
  - email-service/cache  # New

# Deploy
carol@laptop:~/deployment-repo$ panka apply \
  --stack notification-platform \
  --environment development \
  --var VERSION=v1.1.0  # New version with cache support

Wave 1: New component
  + Creating email-service/cache (ElastiCache)... ✓ (7m 30s)

Wave 2: Update existing
  ✓ Updating email-service/api... ✓ (3m 00s)
    - Added REDIS_HOST environment variable
    - Updated image: v1.0.1 → v1.1.0

✓ Deployment successful! (10m 30s)
```

---

### Month 2: Team is Productive

```bash
# Check deployment history
alice@laptop:~$ panka history \
  --stack notification-platform \
  --environment production \
  --limit 10

┌──────────────────────────────────────────────────────────┐
│ Deployment History: notification-platform (production)   │
├──────────────────────────────────────────────────────────┤
│ Version  Date       By          Duration   Status        │
├──────────────────────────────────────────────────────────┤
│ v1.3.0   Feb 15    carol       4m 32s     ✓ Success     │
│ v1.2.5   Feb 12    bob         3m 18s     ✓ Success     │
│ v1.2.4   Feb 10    alice       4m 05s     ✓ Success     │
│ v1.2.3   Feb 08    bob         2m 55s     ✓ Success     │
│ v1.2.2   Feb 07    alice       3m 12s     ⚠ Rolled back │
│ v1.2.1   Feb 05    carol       4m 20s     ✓ Success     │
│ v1.2.0   Feb 01    bob         5m 10s     ✓ Success     │
│ v1.1.0   Jan 28    alice       8m 15s     ✓ Success     │
│ v1.0.1   Jan 20    bob         3m 05s     ✓ Success     │
│ v1.0.0   Jan 15    alice      10m 05s     ✓ Success     │
└──────────────────────────────────────────────────────────┘

Total deployments: 25
Success rate: 96%
Average duration: 4m 12s
```

---

## Key Points

### What Teams Do

1. **Install CLI once** (1 minute)
   ```bash
   curl -sSL panka.io/install.sh | sh
   ```

2. **Configure once** (2 minutes)
   ```bash
   panka init
   ```

3. **Define service in YAML** (30 minutes first time)
   ```yaml
   # Create microservice.yaml, rds.yaml, etc.
   ```

4. **Deploy with one command** (5 minutes)
   ```bash
   panka apply --stack my-stack --environment dev --var VERSION=v1.0.0
   ```

### What Panka Does Automatically

When you run `panka apply`:

```
1. ✓ Reads YAML from disk
2. ✓ Validates configuration
3. ✓ Builds dependency graph
4. ✓ Acquires lock in DynamoDB
5. ✓ Loads current state from S3
6. ✓ Computes what changed
7. ✓ Shows you the plan
8. ✓ Asks for approval (if production)
9. ✓ Executes via Pulumi
10. ✓ Creates/updates AWS resources
11. ✓ Runs health checks
12. ✓ Saves new state to S3
13. ✓ Releases lock
14. ✓ Exits
```

### What You Get

- ✅ Zero-downtime deployments
- ✅ Automatic rollback on failures
- ✅ State tracking (what's deployed)
- ✅ Deployment history
- ✅ Drift detection
- ✅ Cost estimates
- ✅ Multi-environment support
- ✅ Team collaboration (with locking)

---

## FAQs

### Q: Do I need to install anything besides the CLI?

**A**: No. Just the `panka` binary. It talks directly to AWS.

### Q: Where does the CLI run?

**A**: Anywhere:
- Your laptop
- CI/CD runners (GitHub Actions, GitLab CI, Jenkins)
- Bastion hosts
- Anywhere with AWS access

### Q: Who manages the S3 bucket and DynamoDB table?

**A**: Platform team creates them once. All teams share them. Different stacks use different prefixes/keys.

### Q: Can multiple people deploy at the same time?

**A**: Yes, to different stacks. Same stack deployment is serialized by DynamoDB locks.

### Q: What if the CLI crashes during deployment?

**A**: 
- Lock expires after 1 hour (TTL)
- State is saved incrementally
- Pulumi handles partial failures
- You can resume or rollback

### Q: Do I need to learn Pulumi?

**A**: No. You just write YAML. Panka translates to Pulumi internally.

### Q: How do I rollback?

**A**: 
```bash
panka rollback --stack my-stack --environment production
```
Or automatic based on error rate/health checks.

### Q: What about secrets?

**A**: Stored in AWS Secrets Manager, referenced in YAML:
```yaml
secrets:
  - name: DB_PASSWORD
    secretRef: /path/to/secret
    envVar: DATABASE_PASSWORD
```

---

## Complete Example Repository

```
deployment-repo/
├── README.md
│
└── stacks/
    │
    ├── notification-platform/                    # Your stack
    │   ├── stack.yaml
    │   │
    │   ├── services/
    │   │   └── email-service/
    │   │       ├── service.yaml
    │   │       └── components/
    │   │           ├── api/
    │   │           │   ├── microservice.yaml     # ← You define this
    │   │           │   ├── infra.yaml            # ← You define this
    │   │           │   └── configs/
    │   │           │       └── app.yaml          # ← Your app config
    │   │           │
    │   │           ├── database/
    │   │           │   └── rds.yaml              # ← You define this
    │   │           │
    │   │           └── queue/
    │   │               └── sqs.yaml              # ← You define this
    │   │
    │   └── environments/
    │       ├── production/
    │       ├── staging/
    │       └── development/
    │
    └── payment-platform/                         # Another team's stack
        └── ...
```

**Commands you run:**

```bash
# Deploy to dev
panka apply --stack notification-platform --environment development --var VERSION=v1.0.0

# Deploy to production
panka apply --stack notification-platform --environment production --var VERSION=v1.0.0

# Check status
panka status --stack notification-platform --environment production

# View logs
panka logs --component email-service/api --follow
```

---

## Summary

### As a Development Team:

1. **One-Time Setup** (10 minutes)
   - Install CLI: `curl -sSL panka.io/install.sh | sh`
   - Configure: `panka init` (provide S3 bucket & DynamoDB table from platform team)
   - Done!

2. **Define Your Service** (30 minutes)
   - Create YAML files in `stacks/your-stack/`
   - Define components (API, database, cache, etc.)
   - Commit to Git

3. **Deploy** (5 minutes)
   - Build Docker image
   - Push to ECR
   - Run: `panka apply --stack your-stack --environment dev --var VERSION=v1.0.0`
   - Done!

4. **Daily Updates** (5 minutes)
   - Make code changes
   - Build new version
   - Run: `panka apply --var VERSION=v1.0.1`
   - Monitor with `panka status` and `panka logs`

**That's the complete workflow!** 🎉

No backend service. No complex setup. Just:
- A CLI tool
- YAML files
- S3 bucket (provided by platform team)
- DynamoDB table (provided by platform team)

---

**Ready to start? → [GETTING_STARTED_GUIDE.md](docs/GETTING_STARTED_GUIDE.md)**


