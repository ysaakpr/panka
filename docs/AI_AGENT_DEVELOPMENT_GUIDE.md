# AI Agent Development Guide for Panka

This guide provides best practices and strategies for safely using AI agents (like Claude, GitHub Copilot, Cursor, etc.) to accelerate the development of the Panka project.

---

## Table of Contents

1. [Overview](#overview)
2. [AI Agent Safety Principles](#ai-agent-safety-principles)
3. [Phase-by-Phase AI Integration](#phase-by-phase-ai-integration)
4. [Recommended AI Agents](#recommended-ai-agents)
5. [Prompt Engineering Best Practices](#prompt-engineering-best-practices)
6. [Review and Verification](#review-and-verification)
7. [Security Considerations](#security-considerations)
8. [Testing Requirements](#testing-requirements)
9. [Common Pitfalls and Solutions](#common-pitfalls-and-solutions)

---

## Overview

AI agents can significantly accelerate development by:
- Generating boilerplate code
- Implementing well-defined interfaces
- Writing comprehensive test suites
- Creating documentation
- Refactoring code
- Identifying bugs and security issues

However, they must be used safely with proper human oversight.

---

## AI Agent Safety Principles

### 1. Human-in-the-Loop

**ALWAYS:**
✅ Review all AI-generated code before committing
✅ Understand what the code does
✅ Test thoroughly
✅ Verify security implications
✅ Check for edge cases

**NEVER:**
❌ Blindly commit AI-generated code
❌ Deploy without testing
❌ Skip security review
❌ Ignore compiler warnings
❌ Override your judgment

### 2. Incremental Development

- Start with small, well-defined tasks
- Build complexity gradually
- Verify each component before moving forward
- Keep context windows manageable

### 3. Test-Driven Approach

- Write tests first (or alongside code)
- Ensure AI understands test requirements
- Verify test coverage
- Run tests before committing

### 4. Domain Expertise Required

AI agents are tools, not replacements for:
- System design decisions
- Architecture choices
- Security reviews
- Performance optimization
- Production debugging

---

## Phase-by-Phase AI Integration

### Phase 0: Prerequisites & Setup ⭐⭐⭐ (High AI Suitability)

**AI-Friendly Tasks:**

✅ **Project Structure Setup** (90% AI)
```
Prompt: "Create a Go project structure for a CLI tool called Panka with 
packages for state management, locking, parsing, and execution. Include 
proper Go module setup, Makefile, and .gitignore."
```

✅ **CI/CD Configuration** (85% AI)
```
Prompt: "Create a GitHub Actions workflow for a Go project with linting, 
testing, coverage reporting, and LocalStack integration tests."
```

✅ **Development Tools** (90% AI)
```
Prompt: "Create a comprehensive Makefile for a Go project with targets for 
build, test, lint, coverage, and Docker image creation."
```

**Human Review Focus:**
- Verify directory structure matches design
- Check CI/CD includes security scanning
- Ensure development scripts work locally

---

### Phase 1: Core Infrastructure ⭐⭐⭐ (High AI Suitability)

**AI-Friendly Tasks:**

✅ **S3 State Backend** (75% AI)
```
Prompt: "Implement an S3 state backend in Go with methods for:
- SaveState(ctx, key, state) that writes JSON with versioning
- LoadState(ctx, key) that reads the latest version
- ListVersions(ctx, key) that returns all versions
- GetVersion(ctx, key, version) that retrieves a specific version

Use aws-sdk-go-v2. Include error handling and logging."
```

✅ **DynamoDB Lock Manager** (80% AI)
```
Prompt: "Implement a DynamoDB lock manager in Go with:
- AcquireLock(ctx, lockKey, ttl) using conditional writes
- ReleaseLock(ctx, lockKey, lockID)
- RefreshLock(ctx, lockKey, lockID) for heartbeat
- ForceUnlock(ctx, lockKey) for admin operations

Include TTL attribute, error handling, and proper AWS SDK usage."
```

✅ **Configuration Management** (85% AI)
```
Prompt: "Create a configuration manager that reads from:
1. Config file (~/.panka/config.yaml)
2. Environment variables (PANKA_*)
3. Command-line flags

With precedence: flags > env > file. Include validation."
```

**Human Review Focus:**
- Verify S3 versioning is correctly configured
- Test DynamoDB lock race conditions
- Security audit of AWS credentials handling
- Performance testing of lock acquisition

**Testing Strategy:**
```
1. Unit tests for each method (AI: 90%)
2. Integration tests with LocalStack (AI: 85%)
3. Race condition tests (Human: write scenarios, AI: implement)
4. Error handling tests (AI: 90%)
```

---

### Phase 2: YAML Parser & Validator ⭐⭐ (Medium AI Suitability)

**AI-Friendly Tasks:**

✅ **Schema Definitions** (70% AI)
```
Prompt: "Create Go structs for Panka YAML schemas:
- Stack (with metadata, spec.provider, spec.infrastructure)
- Service (with metadata, spec.infrastructure)
- MicroService component (with image, ports, environment, secrets)
- RDS component (with engine, instance, database)

Include YAML tags, validation tags, and documentation comments."
```

✅ **YAML Parser** (75% AI)
```
Prompt: "Implement a YAML parser that:
1. Parses YAML files into Go structs
2. Validates against schemas
3. Handles multi-document YAML
4. Provides detailed error messages with line numbers

Use gopkg.in/yaml.v3 for parsing."
```

⚠️ **Variable Interpolation** (60% AI)
```
Prompt: "Implement variable interpolation for expressions like:
- ${VERSION}
- ${COMPONENT.OUTPUT}
- ${ENV.VAR}

Include escaping, error handling, and circular reference detection."
```

**Human Review Focus:**
- Validate schema completeness against design docs
- Test edge cases in YAML parsing
- Verify variable interpolation security (no code injection)
- Test complex nested structures

**AI Limitations:**
- May not handle all YAML edge cases
- Variable interpolation logic needs careful review
- Schema validation rules need domain expertise

---

### Phase 3: Dependency Resolution ⭐⭐ (Medium AI Suitability)

**AI-Friendly Tasks:**

✅ **Graph Data Structure** (80% AI)
```
Prompt: "Implement a directed acyclic graph (DAG) in Go with:
- AddNode(id, data)
- AddEdge(from, to)
- TopologicalSort() that returns nodes in dependency order
- DetectCycles() that returns cycle paths if any
- GetWaves() that groups nodes that can execute in parallel

Include comprehensive tests."
```

✅ **Dependency Extractor** (70% AI)
```
Prompt: "Extract dependencies from Panka components:
1. Explicit dependsOn references
2. Implicit valueFrom references (component outputs)
3. Secret references

Return a list of (component, dependencies) tuples."
```

⚠️ **Wave Generation** (60% AI)
```
Prompt: "Group components into deployment waves where:
- Wave 1: No dependencies
- Wave 2: Depends only on Wave 1
- Wave N: Depends only on Waves 1..N-1

Components in same wave can deploy in parallel."
```

**Human Review Focus:**
- Verify topological sort correctness
- Test cycle detection with complex graphs
- Validate parallel execution safety
- Performance test with large graphs (1000+ nodes)

---

### Phase 4: Reconciliation Engine ⭐ (Low-Medium AI Suitability)

**AI-Friendly Tasks:**

✅ **State Differ** (65% AI)
```
Prompt: "Implement a state differ that compares current vs desired state:
- Returns CREATE for new resources
- Returns UPDATE for changed resources (with diff)
- Returns DELETE for removed resources
- Returns NO_OP for unchanged resources

Handle nested structures and arrays."
```

⚠️ **Execution Plan Generator** (50% AI)
```
Prompt: "Generate an execution plan showing:
1. Resources to create/update/delete
2. Deployment waves
3. Estimated duration
4. Estimated cost (placeholder for now)
5. Risk level (based on resource types)

Format for human readability with colors and tables."
```

**Human Review Focus:**
- Verify diff algorithm handles all cases
- Test plan generation with complex scenarios
- Validate risk assessment logic
- Check cost estimation accuracy

**AI Limitations:**
- Domain-specific logic (cost estimation, risk scoring)
- Business rules require human input
- Plan formatting needs UX review

---

### Phase 5: Pulumi Integration ⭐ (Low AI Suitability)

**AI-Friendly Tasks:**

✅ **Pulumi Wrapper** (60% AI)
```
Prompt: "Create a Go wrapper around Pulumi automation API:
- Initialize workspace
- Set stack configuration
- Run preview/update/destroy
- Capture outputs
- Handle errors

Include proper context cancellation."
```

⚠️ **Resource Translators - Simple Components** (55% AI)

Examples of what AI can help with:
- S3 bucket translator (straightforward)
- SQS queue translator (straightforward)
- Basic RDS translator (needs review)

```
Prompt: "Implement Pulumi translator for S3 bucket from Panka YAML:

Input YAML:
```yaml
apiVersion: components.panka.io/v1
kind: S3
metadata:
  name: storage
spec:
  bucket:
    name: my-bucket
  versioning:
    enabled: true
```

Output: Pulumi Go code using pulumi-aws SDK."
```

❌ **Resource Translators - Complex Components** (30% AI)

Requires deep AWS knowledge:
- ECS/Fargate (many interconnected resources)
- Load balancers with target groups
- VPC and networking
- IAM roles and policies

**Approach:** Use AI for boilerplate, human for logic

**Human Review Focus:**
- Verify Pulumi resource properties
- Test with real AWS (or LocalStack)
- Check IAM permissions
- Validate networking configuration

---

### Phase 6: CLI & User Experience ⭐⭐⭐ (High AI Suitability)

**AI-Friendly Tasks:**

✅ **CLI Framework** (85% AI)
```
Prompt: "Create a CLI using cobra with commands:
- panka init
- panka apply --stack X --env Y
- panka status --stack X
- panka logs --component X
- panka rollback --stack X

Include flags, help text, and command validation."
```

✅ **Progress Indicators** (80% AI)
```
Prompt: "Add progress indicators using github.com/schollz/progressbar:
- Show deployment progress
- Display current operation
- Show time elapsed
- Update in real-time"
```

✅ **Formatted Output** (85% AI)
```
Prompt: "Create formatted output using tablewriter:
- Resource status tables
- Deployment history
- Cost breakdown
- Color-coded status (green=success, red=fail, yellow=warning)"
```

**Human Review Focus:**
- UX testing with real users
- Error message clarity
- Help text accuracy
- Terminal compatibility

---

### Phase 7: Testing ⭐⭐⭐ (High AI Suitability)

**AI-Friendly Tasks:**

✅ **Unit Tests** (90% AI)
```
Prompt: "Generate unit tests for [function/package] with:
- Happy path tests
- Error cases
- Edge cases
- Boundary conditions
- Table-driven test style

Aim for 90%+ coverage."
```

✅ **Integration Tests** (75% AI)
```
Prompt: "Create integration test for S3 state backend:
1. Setup: Create LocalStack S3 bucket
2. Test: Save state
3. Verify: Read state matches
4. Test: List versions
5. Cleanup: Delete bucket

Use testify/suite for setup/teardown."
```

✅ **Test Fixtures** (85% AI)
```
Prompt: "Generate test YAML fixtures for:
- Valid stack definition
- Stack with multiple services
- Service with all component types
- Invalid stack (for error testing)
- Edge cases (empty, minimal, maximal)"
```

**Human Review Focus:**
- Test coverage completeness
- Test scenarios match requirements
- Integration test stability
- Test data quality

---

## Recommended AI Agents

### 1. Claude (Anthropic) ⭐⭐⭐⭐⭐
**Best For:**
- Large context tasks (100K+ tokens)
- Complex code generation
- Refactoring entire files
- Architecture discussions
- Documentation generation

**Use Cases:**
- "Refactor this entire package to use interfaces"
- "Generate comprehensive documentation"
- "Review this code for security issues"

### 2. GitHub Copilot ⭐⭐⭐⭐
**Best For:**
- Line-by-line code completion
- Function implementations
- Test generation
- Code patterns

**Use Cases:**
- Writing boilerplate code
- Implementing defined interfaces
- Generating test cases

### 3. Cursor ⭐⭐⭐⭐⭐
**Best For:**
- Codebase-aware edits
- Multi-file refactoring
- Bug fixes
- Feature implementation

**Use Cases:**
- "Update all callers of this function"
- "Add error handling throughout"
- "Implement this interface in all components"

### 4. ChatGPT (Code Interpreter) ⭐⭐⭐
**Best For:**
- Algorithm design
- Data structure implementation
- Code explanation
- Learning new concepts

---

## Prompt Engineering Best Practices

### 1. Be Specific and Contextual

❌ **Bad:** "Write code for S3"
✅ **Good:**
```
"Implement an S3StateBackend struct in Go that implements this interface:

type StateBackend interface {
    Save(ctx context.Context, key string, state *State) error
    Load(ctx context.Context, key string) (*State, error)
}

Requirements:
- Use aws-sdk-go-v2
- Store state as JSON
- Enable S3 versioning
- Handle errors with fmt.Errorf wrapping
- Add logging using logrus
- Include context cancellation
- Thread-safe operations
```

### 2. Provide Examples

✅ **Good:**
```
"Generate unit tests similar to this pattern:

func TestStateManager_Save(t *testing.T) {
    tests := []struct {
        name    string
        state   *State
        wantErr bool
    }{
        {"valid state", validState(), false},
        {"nil state", nil, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}

Generate tests for Load, Delete, and List methods."
```

### 3. Specify Constraints

```
"Implement with these constraints:
- Maximum function length: 50 lines
- No global variables
- Use dependency injection
- Include godoc comments
- Error messages must be user-friendly
- No external dependencies except aws-sdk"
```

### 4. Ask for Explanations

```
"Implement X and explain:
1. Key design decisions
2. Potential edge cases
3. Performance implications
4. Security considerations"
```

### 5. Iterate and Refine

```
Step 1: "Generate basic implementation"
Step 2: "Add error handling and logging"
Step 3: "Add tests with 80%+ coverage"
Step 4: "Add performance optimizations"
Step 5: "Add comprehensive documentation"
```

---

## Review and Verification

### Code Review Checklist

#### Correctness ✓
- [ ] Code compiles without warnings
- [ ] Logic matches requirements
- [ ] Error handling is complete
- [ ] Edge cases are handled
- [ ] Tests pass

#### Security ✓
- [ ] No hardcoded credentials
- [ ] Input validation present
- [ ] SQL injection prevention (if applicable)
- [ ] No arbitrary code execution
- [ ] Secure defaults used

#### Performance ✓
- [ ] No obvious performance issues
- [ ] Efficient algorithms used
- [ ] No resource leaks
- [ ] Proper context cancellation
- [ ] Database queries optimized

#### Style ✓
- [ ] Follows Go conventions
- [ ] Consistent naming
- [ ] Proper error wrapping
- [ ] Useful comments (why, not what)
- [ ] No code duplication

#### Testing ✓
- [ ] Unit tests present
- [ ] Integration tests if needed
- [ ] Test coverage adequate (80%+)
- [ ] Tests are readable
- [ ] No flaky tests

---

## Security Considerations

### 1. Secrets Management

❌ **AI May Generate:**
```go
password := "hardcoded-password-123"
```

✅ **You Must Change To:**
```go
password := os.Getenv("DB_PASSWORD")
if password == "" {
    return fmt.Errorf("DB_PASSWORD not set")
}
```

### 2. Input Validation

❌ **AI May Generate:**
```go
func Deploy(stackName string) error {
    cmd := exec.Command("sh", "-c", "deploy " + stackName)
    return cmd.Run()
}
```

✅ **You Must Change To:**
```go
func Deploy(stackName string) error {
    if !isValidStackName(stackName) {
        return fmt.Errorf("invalid stack name: %s", stackName)
    }
    cmd := exec.Command("deploy", stackName) // No shell
    return cmd.Run()
}
```

### 3. AWS Credentials

Always verify AI-generated code:
- Uses IAM roles (not access keys)
- Follows least privilege principle
- Includes proper error handling
- Doesn't log sensitive data

### 4. YAML Parsing

AI may not consider:
- YAML bombs (deeply nested structures)
- Arbitrary code execution via yaml tags
- Resource exhaustion

**Always add limits:**
```go
decoder := yaml.NewDecoder(reader)
decoder.KnownFields(true) // Strict mode
```

---

## Testing Requirements

### For All AI-Generated Code

#### 1. Unit Tests (Required)
```go
// AI generates implementation
func (s *StateManager) Save(ctx context.Context, key string, state *State) error {
    // ... implementation
}

// AI generates tests
func TestStateManager_Save(t *testing.T) {
    // ... comprehensive tests
}
```

**Human Verification:**
- [ ] Tests cover happy path
- [ ] Tests cover error cases
- [ ] Tests cover edge cases
- [ ] Coverage >= 80%
- [ ] Tests are readable

#### 2. Integration Tests (For External Dependencies)

```go
// +build integration

func TestS3StateBackend_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Use LocalStack
    client := setupLocalStack(t)
    backend := NewS3StateBackend(client, "test-bucket")
    
    // Test with real S3
    // ...
}
```

#### 3. Manual Testing Checklist

Before committing AI-generated code:
- [ ] Run locally
- [ ] Test with real AWS (sandbox account)
- [ ] Test error scenarios
- [ ] Verify logs are useful
- [ ] Check resource cleanup

---

## Common Pitfalls and Solutions

### Pitfall 1: Over-Optimization

**Problem:** AI generates overly complex code
```go
// AI might generate fancy caching, premature optimization
```

**Solution:** Ask for simple implementation first
```
Prompt: "Implement simple version first. No caching, no optimization.
We'll add those later if needed."
```

### Pitfall 2: Missing Error Context

**Problem:** Generic error handling
```go
if err != nil {
    return err  // No context!
}
```

**Solution:** Explicitly request error wrapping
```
Prompt: "All errors must be wrapped with context using fmt.Errorf
with %w verb for error chains."
```

### Pitfall 3: Poor Test Quality

**Problem:** AI generates tests that always pass
```go
func TestAlwaysPasses(t *testing.T) {
    // Test that doesn't actually test anything
}
```

**Solution:** Review and improve tests
- Verify tests actually test the behavior
- Add negative test cases
- Check test coverage

### Pitfall 4: Inconsistent Patterns

**Problem:** Different AI sessions use different patterns

**Solution:** Provide style guide
```
Prompt: "Follow these patterns:
- Use constructor functions (NewXxx)
- Interfaces in separate files (*_interface.go)
- Mocks in *_mock.go
- Tests in *_test.go
- Error messages lowercase without punctuation"
```

### Pitfall 5: Missing Domain Logic

**Problem:** AI doesn't understand business requirements

**Solution:** Provide detailed requirements
```
Prompt: "When deploying to production:
1. Must show approval prompt
2. Must verify cost < $1000/month
3. Must check for breaking changes
4. Must create backup before update
5. Must send Slack notification"
```

---

## Example: Complete AI-Assisted Workflow

### Task: Implement S3 State Backend

#### Step 1: Define Interface (Human)
```go
// pkg/state/interface.go
type StateBackend interface {
    Save(ctx context.Context, key string, state *State) error
    Load(ctx context.Context, key string) (*State, error)
    List(ctx context.Context) ([]string, error)
    Delete(ctx context.Context, key string) error
}
```

#### Step 2: Generate Implementation (AI)
```
Prompt to Claude:

"Implement S3StateBackend that implements the StateBackend interface.

Requirements:
- Use aws-sdk-go-v2 S3 client
- Store state as JSON in S3
- Key format: stacks/{stack}/environments/{env}/state.json
- Enable versioning
- Use zap logger
- Handle context cancellation
- Thread-safe
- No global variables

Include:
- Constructor: NewS3StateBackend(client *s3.Client, bucket string, logger *zap.Logger) *S3StateBackend
- Proper error handling with fmt.Errorf and %w
- Godoc comments
"
```

#### Step 3: Review (Human)
- Check S3 operations are correct
- Verify error handling
- Test context cancellation
- Check thread safety

#### Step 4: Generate Tests (AI)
```
Prompt: "Generate comprehensive unit tests for S3StateBackend:
- Use testify/mock for S3 client
- Table-driven tests
- Cover all methods
- Include error cases
- Target 90%+ coverage"
```

#### Step 5: Generate Integration Tests (AI)
```
Prompt: "Generate integration tests for S3StateBackend:
- Use LocalStack
- Test with real S3 operations
- Setup/teardown with testify/suite
- Test state versioning
- Tag with +build integration"
```

#### Step 6: Manual Testing (Human)
```bash
# Run unit tests
go test -v ./pkg/state/...

# Run integration tests
go test -v -tags=integration ./pkg/state/...

# Test with real AWS (sandbox)
AWS_PROFILE=sandbox go run examples/state-backend/main.go
```

#### Step 7: Code Review (Human)
- [ ] Passes all tests
- [ ] No security issues
- [ ] Performance acceptable
- [ ] Documentation complete
- [ ] Ready to commit

---

## Metrics and Success Criteria

### AI Effectiveness Metrics

Track these to improve AI usage:

1. **AI-Generated Code Acceptance Rate**
   - Target: > 70% of AI code committed without major changes
   - Measure: (Lines committed / Lines generated)

2. **Bug Rate in AI Code**
   - Target: Same or lower than human-written code
   - Measure: Bugs found in AI vs human code

3. **Development Velocity**
   - Target: 2-3x faster than manual coding
   - Measure: Story points per sprint

4. **Test Coverage**
   - Target: > 80% for AI-generated code
   - Measure: go test -cover

5. **Code Review Time**
   - Target: < 30 min per PR
   - Measure: Time from PR creation to approval

---

## Emergency Procedures

### When AI Code Causes Issues

#### 1. Immediate Response
```bash
# Revert the commit
git revert <commit-hash>

# Or rollback
git reset --hard HEAD~1  # If not pushed

# Deploy previous version
panka apply --stack X --var VERSION=previous
```

#### 2. Post-Incident Review
- Identify what AI misunderstood
- Document the issue
- Update prompts to prevent recurrence
- Add tests for the failure case

#### 3. Prevention
- More thorough code review
- Better test coverage
- Improved prompts
- Human validation of critical paths

---

## Summary: AI Usage Guidelines

### ✅ High AI Suitability (80-90% AI)
- Boilerplate code
- Test generation
- Configuration files
- CLI commands
- Documentation

### ⚠️ Medium AI Suitability (50-70% AI)
- Business logic (needs review)
- Complex algorithms (verify correctness)
- Security-sensitive code (audit required)
- Performance-critical code (benchmark)

### ❌ Low AI Suitability (< 50% AI)
- System architecture decisions
- Security policy design
- Production debugging
- Performance optimization
- Complex AWS infrastructure

### Golden Rules

1. **Trust but Verify** - Review everything
2. **Test Thoroughly** - Write comprehensive tests
3. **Iterate** - Start simple, add complexity
4. **Document** - Explain AI-assisted code
5. **Learn** - Understand what AI generates
6. **Improve** - Refine prompts over time

---

## Conclusion

AI agents are powerful tools that can accelerate development significantly. By following these guidelines, you can:

- ✅ Develop 2-3x faster
- ✅ Maintain high code quality
- ✅ Reduce bugs
- ✅ Improve test coverage
- ✅ Generate better documentation

**Remember:** AI is a tool to augment your skills, not replace your judgment. Use it wisely! 🚀

---

**Next Steps:**
1. Review this guide with your team
2. Set up recommended AI tools
3. Start with Phase 0 using AI assistance
4. Track metrics and improve
5. Share learnings with the team

**Questions?** See [CONTRIBUTING.md](../CONTRIBUTING.md) or reach out to the platform team.

