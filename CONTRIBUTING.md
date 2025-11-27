# Contributing to Panka

Thank you for your interest in contributing to Panka! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Pull Request Process](#pull-request-process)
- [Release Process](#release-process)

---

## Code of Conduct

### Our Pledge

We pledge to make participation in our project a harassment-free experience for everyone, regardless of age, body size, disability, ethnicity, gender identity and expression, level of experience, nationality, personal appearance, race, religion, or sexual identity and orientation.

### Our Standards

**Positive behavior includes:**
- Using welcoming and inclusive language
- Being respectful of differing viewpoints
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

**Unacceptable behavior includes:**
- Trolling, insulting/derogatory comments, and personal attacks
- Public or private harassment
- Publishing others' private information without permission
- Other conduct which could reasonably be considered inappropriate

---

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- AWS CLI configured
- Git
- Make

### Development Setup

1. **Fork and clone the repository**

```bash
git clone https://github.com/YOUR_USERNAME/panka.git
cd panka
```

2. **Install development tools**

```bash
make tools
```

3. **Start development environment**

```bash
make dev
```

4. **Verify setup**

```bash
make test
make lint
make build
```

---

## Development Workflow

### 1. Create a Branch

Always create a new branch for your work:

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
```

**Branch naming conventions:**
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions or changes
- `chore/` - Maintenance tasks

### 2. Make Changes

- Write clean, well-documented code
- Follow the coding standards (see below)
- Add tests for new functionality
- Update documentation as needed

### 3. Run Tests

```bash
# Run all tests
make test-all

# Run specific test types
make test              # Unit tests
make test-integration  # Integration tests
make test-e2e         # E2E tests
```

### 4. Check Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run pre-commit checks
make pre-commit
```

### 5. Commit Changes

Write clear, descriptive commit messages:

```bash
git add .
git commit -m "feat: add support for EKS deployments

- Implement EKS translator
- Add EKS schema definitions
- Update documentation
- Add integration tests

Closes #123"
```

**Commit message format:**
```
<type>: <subject>

<body>

<footer>
```

**Types:**
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `style` - Formatting
- `refactor` - Code restructuring
- `test` - Tests
- `chore` - Maintenance

### 6. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

---

## Coding Standards

### Go Style Guide

Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

**Key points:**

1. **Formatting**
   - Use `gofmt` (run via `make fmt`)
   - Use `goimports` for import organization
   - Line length: aim for 100 characters, max 120

2. **Naming**
   - Use camelCase for variables and functions
   - Use PascalCase for exported identifiers
   - Use descriptive names (no single-letter vars except loop counters)
   - Interface names end with `-er` (e.g., `StateManager`, `Executor`)

3. **Comments**
   - All exported functions must have comments
   - Comments should be complete sentences
   - Package comments should describe the package purpose

```go
// Good
// StateManager manages deployment state in S3.
// It provides methods for loading, saving, and versioning state.
type StateManager interface {
    // Load retrieves the current state for a stack and environment.
    Load(ctx context.Context, stack, environment string) (*State, error)
}

// Bad
// state manager
type StateManager interface {
    Load(ctx context.Context, stack, environment string) (*State, error) // load
}
```

4. **Error Handling**
   - Always check errors
   - Wrap errors with context using `fmt.Errorf`
   - Don't panic (except in truly unrecoverable situations)

```go
// Good
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to process request: %w", err)
}

// Bad
result, _ := someFunction()  // Ignoring error
```

5. **Context**
   - Always pass context as first parameter
   - Use context for cancellation and timeouts
   - Don't store context in structs

```go
// Good
func (s *Service) Process(ctx context.Context, data string) error {
    // ...
}

// Bad
func (s *Service) Process(data string) error {
    // ...
}
```

### YAML Style Guide

1. **Indentation**: 2 spaces
2. **Structure**: Group related fields
3. **Comments**: Use `#` for inline comments
4. **Quotes**: Use quotes for strings with special characters

```yaml
# Good
apiVersion: components.panka.io/v1
kind: MicroService

metadata:
  name: api
  description: "Main API server"
  
spec:
  image:
    repository: myapp/api
    tag: v1.0.0

# Bad
apiVersion: components.panka.io/v1
kind:        MicroService
metadata:
 name:api
spec: {image:{repository:myapp/api,tag:v1.0.0}}
```

---

## Testing

### Test Organization

```
test/
├── unit/           # Unit tests (next to source)
├── integration/    # Integration tests
├── e2e/           # End-to-end tests
└── fixtures/      # Test data
```

### Writing Tests

1. **Unit Tests** - Fast, isolated, no external dependencies

```go
func TestStateManager_Load(t *testing.T) {
    // Arrange
    manager := NewStateManager(mockS3Client, "test-bucket")
    
    // Act
    state, err := manager.Load(context.Background(), "test-stack", "dev")
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, "test-stack", state.Metadata.Stack)
}
```

2. **Table-Driven Tests** - For multiple test cases

```go
func TestValidator_ValidateMicroService(t *testing.T) {
    tests := []struct {
        name    string
        ms      *schema.MicroService
        wantErr bool
    }{
        {
            name: "valid microservice",
            ms:   createValidMicroService(),
            wantErr: false,
        },
        {
            name: "missing image",
            ms:   &schema.MicroService{},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            validator := NewValidator()
            err := validator.ValidateMicroService(tt.ms)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

3. **Integration Tests** - Use LocalStack

```go
// +build integration

func TestS3StateManager_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    client := setupLocalStackS3(t)
    manager := NewS3StateManager(client, "test-bucket")
    
    // Test with real S3
    // ...
}
```

### Test Coverage

- Aim for 80%+ overall coverage
- 100% coverage for critical paths
- Run coverage locally:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Documentation

### Types of Documentation

1. **Code Comments** - Explain why, not what
2. **README** - Project overview
3. **User Guides** - How to use the system
4. **API Documentation** - Generated from code
5. **Architecture Docs** - Design decisions

### Documentation Standards

1. **Markdown** - Use markdown for all docs
2. **Structure** - Use clear headings and sections
3. **Examples** - Include code examples
4. **Diagrams** - Use ASCII art or images

### Updating Documentation

When you make changes:
- Update relevant doc files
- Update code comments
- Update examples if needed
- Update README if public API changed

---

## Pull Request Process

### Before Submitting

- [ ] Tests pass (`make test-all`)
- [ ] Code is formatted (`make fmt`)
- [ ] Linter passes (`make lint`)
- [ ] Documentation updated
- [ ] Commit messages follow conventions
- [ ] Branch is up to date with main

### PR Description

Use this template:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Changes Made
- List of changes
- One per line

## Testing
- How was this tested?
- What test cases were added?

## Screenshots (if applicable)

## Checklist
- [ ] Tests pass
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
- [ ] Follows coding standards
```

### Review Process

1. **Automated Checks** - CI must pass
2. **Code Review** - At least 1 approval required
3. **Testing** - Reviewer tests changes
4. **Approval** - Maintainer approves

### After Approval

- Squash and merge (if multiple small commits)
- Or merge commit (if commits are meaningful)
- Delete branch after merge

---

## Release Process

### Version Numbers

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR.MINOR.PATCH** (e.g., 1.2.3)
- **MAJOR** - Breaking changes
- **MINOR** - New features (backward compatible)
- **PATCH** - Bug fixes

### Release Steps

1. **Update version**
   ```bash
   # Update version in relevant files
   vim VERSION
   ```

2. **Update changelog**
   ```bash
   vim CHANGELOG.md
   ```

3. **Create release branch**
   ```bash
   git checkout -b release/v1.2.3
   ```

4. **Run full test suite**
   ```bash
   make test-all
   ```

5. **Create tag**
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

6. **GitHub Release**
   - Create release on GitHub
   - Add release notes
   - Upload binaries

---

## Areas for Contribution

We welcome contributions in these areas:

### High Priority
- [ ] Additional component translators (EKS, DocumentDB, etc.)
- [ ] Performance optimizations
- [ ] Enhanced error messages
- [ ] Improved test coverage

### Medium Priority
- [ ] UI/UX improvements
- [ ] Additional output formats (JSON, YAML)
- [ ] Plugin system
- [ ] Advanced deployment strategies

### Documentation
- [ ] Tutorial videos
- [ ] More examples
- [ ] Architecture diagrams
- [ ] Best practices guide

### Nice to Have
- [ ] Web UI
- [ ] VS Code extension
- [ ] Slack bot
- [ ] Terraform provider

---

## Getting Help

### Questions?

- **Slack**: #panka-dev
- **Email**: platform-team@company.com
- **GitHub Discussions**: github.com/company/panka/discussions

### Found a Bug?

1. Check existing issues
2. Create detailed bug report with:
   - Steps to reproduce
   - Expected behavior
   - Actual behavior
   - Environment details
   - Logs/screenshots

### Want a Feature?

1. Check existing feature requests
2. Create feature request with:
   - Use case
   - Proposed solution
   - Alternatives considered
   - Willingness to contribute

---

## Recognition

Contributors are recognized in:
- README contributors section
- Release notes
- Annual contributor report

Thank you for contributing to Panka! 🚀



