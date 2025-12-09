# Panka Documentation Index

Welcome to Panka - a multi-tenant infrastructure deployment tool for managing cloud resources declaratively.

---

## 📐 Architecture (Start Here!)

> **IMPORTANT:** The new architecture document supersedes all previous architecture docs.

| Document | Description | Status |
|----------|-------------|--------|
| **[Architecture v2.0](docs/ARCHITECTURE_V2.md)** | **Authoritative architecture document** | ✅ **Current** |

**Key concepts in v2.0:**
- **Tenant** → VPC, Subnets, Security Groups (shared networking)
- **Stack** → Folder containing `stack.yaml` + `services/`
- **Service** → Folder containing `service.yaml` + component YAMLs
- **Component** → Individual AWS resource (ECS, RDS, SQS, etc.)

**Example Stack Structure:**
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

See [examples/notification-platform/](examples/notification-platform/) for a complete example.

---

## 🚀 Quick Start

New to Panka? Start with these guides:

| Guide | Description | Audience |
|-------|-------------|----------|
| [README.md](README.md) | Project overview and features | Everyone |
| [Quickstart Guide](docs/quickstart/QUICKSTART.md) | Get started in 5 minutes | Developers |
| [CLI Quickstart](docs/quickstart/QUICKSTART_CLI.md) | Using the Panka CLI | Developers |
| [Multi-Tenancy Quickstart](docs/quickstart/QUICKSTART_MULTI_TENANCY.md) | Multi-tenant setup guide | Platform Teams |

---

## 📚 User Documentation

### Getting Started

| Document | Purpose |
|----------|---------|
| [Getting Started Guide](docs/GETTING_STARTED_GUIDE.md) | Complete getting started walkthrough |
| [Setup AWS Credentials](docs/quickstart/SETUP_AWS_CREDENTIALS.md) | Configure AWS access for Panka |
| [Corrected Login Flow](docs/quickstart/CORRECTED_LOGIN_FLOW.md) | Understanding the login workflow |

### Core Concepts

| Document | Purpose | Status |
|----------|---------|--------|
| **[Architecture v2.0](docs/ARCHITECTURE_V2.md)** | **Tenant → Stack → Service hierarchy** | ✅ **Current** |
| [CLI Architecture](docs/CLI_ARCHITECTURE.md) | Command-line interface design | Current |
| [State and Locking](docs/STATE_AND_LOCKING.md) | How state management works | Current |
| [Multi-Tenancy](docs/MULTI_TENANCY.md) | Multi-tenant architecture | Current |
| [Architecture Overview](docs/ARCHITECTURE.md) | System architecture (v1) | ⚠️ Superseded |

### Reference Documentation

| Document | Purpose |
|----------|---------|
| [Complete Overview](docs/reference/COMPLETE_OVERVIEW.md) | Comprehensive system overview |
| [S3 State Structure](docs/reference/S3_STATE_STRUCTURE.md) | S3 bucket organization explained |
| [How Teams Use Panka](docs/reference/HOW_TEAMS_USE_PANKA.md) | Real-world team workflows |
| [Summary for Teams](docs/reference/SUMMARY_FOR_TEAMS.md) | Quick reference for teams |

### Workflows & Guides

| Document | Purpose | Audience |
|----------|---------|----------|
| [User Workflows](docs/USER_WORKFLOWS.md) | Common user workflows | Developers |
| [Platform Admin Guide](docs/PLATFORM_ADMIN_GUIDE.md) | Managing tenants and platform | Platform Teams |
| [Multi-Tenant Quickstart](docs/quickstart/MULTI_TENANT_QUICKSTART.md) | Setting up multi-tenancy | Platform Teams |

---

## 👥 By Role

### For Developers (Using Panka)

**First Time Setup:**
1. [Quickstart Guide](docs/quickstart/QUICKSTART.md) - Get started
2. [CLI Quickstart](docs/quickstart/QUICKSTART_CLI.md) - Learn the commands
3. [Login Flow](docs/quickstart/CORRECTED_LOGIN_FLOW.md) - Understanding authentication
4. [User Workflows](docs/USER_WORKFLOWS.md) - Common tasks

**Reference:**
- [S3 State Structure](docs/reference/S3_STATE_STRUCTURE.md) - Where your state is stored
- [State and Locking](docs/STATE_AND_LOCKING.md) - How it works

### For Platform Administrators

**Initial Setup:**
1. [Multi-Tenancy Quickstart](docs/quickstart/QUICKSTART_MULTI_TENANCY.md) - Setup multi-tenancy
2. [Setup AWS Credentials](docs/quickstart/SETUP_AWS_CREDENTIALS.md) - Configure AWS
3. [Platform Admin Guide](docs/PLATFORM_ADMIN_GUIDE.md) - Complete admin guide

**Managing Tenants:**
- [Multi-Tenancy Architecture](docs/MULTI_TENANCY.md) - How isolation works
- [Platform Admin Guide](docs/PLATFORM_ADMIN_GUIDE.md) - Tenant management
- [S3 State Structure](docs/reference/S3_STATE_STRUCTURE.md) - Understanding storage

**Reference:**
- [Complete Overview](docs/reference/COMPLETE_OVERVIEW.md) - Full system overview
- [How Teams Use Panka](docs/reference/HOW_TEAMS_USE_PANKA.md) - Team patterns

### For Contributors

**Contributing:**
- [CONTRIBUTING.md](docs/CONTRIBUTING.md) - How to contribute
- [Architecture](docs/ARCHITECTURE.md) - System architecture
- [CLI Architecture](docs/CLI_ARCHITECTURE.md) - CLI design

**Development:**
- [Development Changelog](docs/dev/) - Implementation history
- [AI Development Guide](docs/dev/AI_AGENT_DEVELOPMENT_GUIDE.md) - AI-assisted development

---

## 🏗️ Architecture & Design

| Document | Purpose |
|----------|---------|
| [Architecture Overview](docs/ARCHITECTURE.md) | Complete system architecture |
| [CLI Architecture](docs/CLI_ARCHITECTURE.md) | Command-line interface design |
| [Multi-Tenancy Architecture](docs/MULTI_TENANCY.md) | Tenant isolation design |
| [State and Locking](docs/STATE_AND_LOCKING.md) | State management architecture |

---

## 🔧 Development & Contribution

### Contributing

- [Contributing Guide](docs/CONTRIBUTING.md) - How to contribute to Panka
- [AI Development Guide](docs/dev/AI_AGENT_DEVELOPMENT_GUIDE.md) - Using AI for development

### Development Progress

All implementation history and changelogs are in [`docs/dev/`](docs/dev/):

**Phase Summaries:**
- [Phase 1: Foundation](docs/dev/PHASE1_COMPLETE.md) - Logging, config, state backend
- [Phase 2: Parser & Validator](docs/dev/PHASE2_COMPLETE.md) - YAML parsing
- [Phase 3: Graph & Planning](docs/dev/PHASE3_COMPLETE.md) - Dependency resolution
- [Phase 4: AWS Providers](docs/dev/PHASE4_COMPLETE_SUMMARY.md) - AWS resource providers
- [Phase 5: CLI Commands](docs/dev/PHASE5_COMPLETE.md) - Command-line interface
- [Phase 6: Multi-Tenancy](docs/dev/PHASE6_MULTITENANCY_COMPLETE.md) - Tenant isolation

**Additional Resources:**
- [Development Progress](docs/dev/DEVELOPMENT_PROGRESS.md) - Cumulative progress tracker
- [Multi-Tenancy Implementation](docs/dev/MULTI_TENANCY_IMPLEMENTATION.md) - Complete MT implementation
- [AI Development Summary](docs/dev/AI_DEVELOPMENT_SUMMARY.md) - AI development metrics

**Session Summaries:**
- [Phase 4 Session 2](docs/dev/PHASE4_SESSION2_COMPLETE.md) - AWS provider implementation
- [Phase 4 Testing](docs/dev/PHASE4_TESTING_COMPLETE.md) - Testing completion
- [Phase 5 Checkpoint 1](docs/dev/PHASE5_CHECKPOINT1.md) - CLI development checkpoint
- [Phases 1-2-3 Summary](docs/dev/PHASES_1_2_3_SUMMARY.md) - Combined summary

---

## 📖 Documentation by Topic

### Authentication & Access

| Document | Topic |
|----------|-------|
| [Corrected Login Flow](docs/quickstart/CORRECTED_LOGIN_FLOW.md) | How authentication works |
| [Setup AWS Credentials](docs/quickstart/SETUP_AWS_CREDENTIALS.md) | AWS credential configuration |
| [Multi-Tenancy](docs/MULTI_TENANCY.md) | Tenant authentication |
| [Platform Admin Guide](docs/PLATFORM_ADMIN_GUIDE.md) | Admin authentication |

### State Management

| Document | Topic |
|----------|-------|
| [State and Locking](docs/STATE_AND_LOCKING.md) | State management overview |
| [S3 State Structure](docs/reference/S3_STATE_STRUCTURE.md) | S3 bucket organization |
| [Complete Overview](docs/reference/COMPLETE_OVERVIEW.md) | State in context |

### Multi-Tenancy

| Document | Topic |
|----------|-------|
| [Multi-Tenancy Architecture](docs/MULTI_TENANCY.md) | Tenant isolation design |
| [Multi-Tenant Quickstart](docs/quickstart/MULTI_TENANT_QUICKSTART.md) | Setup guide |
| [Platform Admin Guide](docs/PLATFORM_ADMIN_GUIDE.md) | Managing tenants |
| [Multi-Tenancy Implementation](docs/dev/MULTI_TENANCY_IMPLEMENTATION.md) | Technical details |

### CLI Usage

| Document | Topic |
|----------|-------|
| [CLI Quickstart](docs/quickstart/QUICKSTART_CLI.md) | Command overview |
| [CLI Architecture](docs/CLI_ARCHITECTURE.md) | CLI design |
| [User Workflows](docs/USER_WORKFLOWS.md) | Common commands |
| [Corrected Login Flow](docs/quickstart/CORRECTED_LOGIN_FLOW.md) | Login commands |

---

## 📂 Directory Structure

```
panka/
├── README.md                           ← Start here
├── INDEX.md                            ← This file
│
├── docs/                               ← Main documentation
│   ├── ARCHITECTURE.md                 ← System architecture
│   ├── CLI_ARCHITECTURE.md             ← CLI design
│   ├── CONTRIBUTING.md                 ← Contributing guide
│   ├── GETTING_STARTED_GUIDE.md        ← Getting started
│   ├── MULTI_TENANCY.md                ← Multi-tenancy architecture
│   ├── PLATFORM_ADMIN_GUIDE.md         ← Platform admin guide
│   ├── STATE_AND_LOCKING.md            ← State management
│   ├── USER_WORKFLOWS.md               ← User workflows
│   │
│   ├── quickstart/                     ← Getting started guides
│   │   ├── QUICKSTART.md               ← Main quickstart
│   │   ├── QUICKSTART_CLI.md           ← CLI quickstart
│   │   ├── QUICKSTART_MULTI_TENANCY.md ← Multi-tenancy quickstart
│   │   ├── MULTI_TENANT_QUICKSTART.md  ← Alternative MT guide
│   │   ├── CORRECTED_LOGIN_FLOW.md     ← Login workflow
│   │   └── SETUP_AWS_CREDENTIALS.md    ← AWS setup
│   │
│   ├── reference/                      ← Reference documentation
│   │   ├── COMPLETE_OVERVIEW.md        ← System overview
│   │   ├── S3_STATE_STRUCTURE.md       ← S3 structure guide
│   │   ├── HOW_TEAMS_USE_PANKA.md      ← Team workflows
│   │   └── SUMMARY_FOR_TEAMS.md        ← Team summary
│   │
│   └── dev/                            ← Development & changelogs
│       ├── PHASE1_COMPLETE.md          ← Phase 1 summary
│       ├── PHASE2_COMPLETE.md          ← Phase 2 summary
│       ├── PHASE3_COMPLETE.md          ← Phase 3 summary
│       ├── PHASE4_COMPLETE_SUMMARY.md  ← Phase 4 summary
│       ├── PHASE5_COMPLETE.md          ← Phase 5 summary
│       ├── PHASE6_MULTITENANCY_COMPLETE.md ← Phase 6 summary
│       ├── AI_DEVELOPMENT_SUMMARY.md   ← AI metrics
│       ├── DEVELOPMENT_PROGRESS.md     ← Progress tracker
│       ├── MULTI_TENANCY_IMPLEMENTATION.md ← MT implementation
│       └── ... (more dev docs)
```

---

## 🎯 Common Questions

### "How do I get started?"
→ [Quickstart Guide](docs/quickstart/QUICKSTART.md)

### "How do I login?"
→ [Corrected Login Flow](docs/quickstart/CORRECTED_LOGIN_FLOW.md)

### "Where is my state stored?"
→ [S3 State Structure](docs/reference/S3_STATE_STRUCTURE.md)

### "How do I set up multi-tenancy?"
→ [Multi-Tenancy Quickstart](docs/quickstart/QUICKSTART_MULTI_TENANCY.md)

### "How do I create tenants?"
→ [Platform Admin Guide](docs/PLATFORM_ADMIN_GUIDE.md)

### "What AWS permissions do I need?"
→ [Setup AWS Credentials](docs/quickstart/SETUP_AWS_CREDENTIALS.md)

### "How does state locking work?"
→ [State and Locking](docs/STATE_AND_LOCKING.md)

### "How do I contribute?"
→ [Contributing Guide](docs/CONTRIBUTING.md)

---

## 📊 Documentation Statistics

- **Total Documents:** 41
- **User-Facing Docs:** 18
- **Development Docs:** 23
- **Quickstart Guides:** 6
- **Reference Docs:** 4

---

## 🔗 External Links

- **GitHub Repository:** (Add your repo link)
- **Issue Tracker:** (Add your issues link)
- **Discussions:** (Add your discussions link)

---

## 📝 License

See [LICENSE](LICENSE) file for details.

---

**Last Updated:** November 28, 2024  
**Version:** 0.1.0-dev

---

<p align="center">Made with ❤️ by the Panka Team</p>
