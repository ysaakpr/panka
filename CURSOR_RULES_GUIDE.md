# How to Use Cursor Rules Effectively

## What is `.cursorrules`?

The `.cursorrules` file is a project-specific instruction file that provides context and guidelines to AI assistants (like Claude in Cursor). It ensures **consistent behavior** across all AI interactions, regardless of who is using the tool or what they're asking.

---

## How Cursor Rules Work

### 1. **Automatic Context**
When you interact with an AI in Cursor, the `.cursorrules` file is automatically:
- ✅ Read at the start of every conversation
- ✅ Injected into the AI's context
- ✅ Used to guide all responses and code generation

### 2. **No Manual Mention Needed**
You don't need to say:
- ❌ "Follow the cursor rules"
- ❌ "Remember to check .cursorrules"
- ❌ "Use the project standards"

The AI **automatically knows** to follow the rules!

### 3. **Scope**
- **Project-wide**: Applies to all AI interactions in this workspace
- **Persistent**: Works across sessions
- **Automatic**: No activation needed

---

## What to Put in `.cursorrules`

### ✅ Include:

**1. Project Structure**
```
panka/
├── cmd/         # CLI entry point
├── internal/    # Private packages
├── pkg/         # Public packages
└── docs/        # Documentation
```
→ Helps AI understand where to place new files

**2. Code Patterns & Standards**
```go
// ✅ GOOD
func NewService(cfg *Config) (*Service, error)

// ❌ BAD  
func NewService(cfg Config) Service
```
→ Ensures consistent code style

**3. Architecture Decisions**
```
- Multi-tenancy: All operations must respect tenant isolation
- State: S3 with versioning
- Locks: DynamoDB with TTL
```
→ AI makes decisions aligned with your architecture

**4. Common Anti-Patterns**
```go
// ❌ NEVER: Use global variables
var globalDB *sql.DB

// ✅ ALWAYS: Pass dependencies
type Service struct {
    db *sql.DB
}
```
→ Prevents AI from generating problematic code

**5. Testing Requirements**
```
- Every public function needs tests
- Test file: <source>_test.go
- Use table-driven tests for multiple cases
```
→ AI generates tests automatically

**6. Documentation Standards**
```
- Update INDEX.md when adding docs
- User docs → docs/
- Dev changelogs → docs/dev/
```
→ AI knows where to place documentation

### ❌ Avoid:

**1. Implementation Details**
- Don't put actual code implementations
- Put patterns and principles instead

**2. Changing Requirements**
- Don't put temporary decisions
- Put stable architectural choices

**3. Project-Specific Data**
- Don't put API keys, endpoints, etc.
- Put general patterns instead

---

## How AI Uses `.cursorrules`

### When Writing Code

**You ask:** "Add a new S3 provider"

**AI thinks (automatically):**
1. Check `.cursorrules` for provider pattern
2. See: "All providers go in `pkg/provider/aws/`"
3. See: "Implement Provider interface"
4. See: "Add tests in `<name>_test.go`"
5. See: "Use AWS SDK v2"
6. Generate code following all these patterns!

### When Creating Documentation

**You ask:** "Document the new feature"

**AI thinks (automatically):**
1. Check `.cursorrules` for doc structure
2. See: "User docs go in `docs/quickstart/`"
3. See: "Update INDEX.md with links"
4. See: "Use structured format with examples"
5. Create docs following all conventions!

### When Fixing Bugs

**You ask:** "Fix this error handling"

**AI thinks (automatically):**
1. Check `.cursorrules` for error patterns
2. See: "Wrap errors with context using fmt.Errorf"
3. See: "Log errors before returning"
4. See: "Use structured logging with zap"
5. Fix using proper patterns!

---

## Best Practices for `.cursorrules`

### 1. **Be Specific with Examples**

**❌ Vague:**
```
Use good error handling
```

**✅ Specific:**
```go
// ✅ GOOD: Wrap errors with context
if err := save(); err != nil {
    return fmt.Errorf("failed to save state: %w", err)
}

// ❌ BAD: Generic errors
if err := save(); err != nil {
    return err
}
```

### 2. **Show Both Good and Bad**

Always show anti-patterns:
```go
// ✅ DO THIS
func New(deps Dependencies) *Service

// ❌ DON'T DO THIS
var globalService *Service
```

### 3. **Organize by Topic**

Structure your rules:
```
## Code Structure
## Testing Requirements  
## Documentation Standards
## Security Requirements
## Performance Guidelines
```

### 4. **Include File Paths**

Be explicit about where files go:
```
Add CLI command → internal/cli/<command>.go
Add AWS resource → pkg/provider/aws/<resource>.go
Add user doc → docs/quickstart/<NAME>.md
```

### 5. **Add Quick Reference**

Include a cheat sheet:
```
| Task | File to Edit |
|------|-------------|
| CLI command | internal/cli/ |
| AWS resource | pkg/provider/aws/ |
```

---

## Testing Your `.cursorrules`

### Method 1: Ask AI to Explain

**You:** "Explain our error handling pattern"

**AI should respond** with the pattern from `.cursorrules`:
- Wrap with fmt.Errorf
- Add context
- Use %w for error wrapping
- Log before returning

### Method 2: Generate Code

**You:** "Add a new provider function"

**AI should:**
- Place it in correct directory (per rules)
- Follow interface patterns (per rules)
- Include error handling (per rules)
- Add tests automatically (per rules)

### Method 3: Review Generated Code

After AI generates code, check:
- ✅ Follows patterns in `.cursorrules`?
- ✅ In correct directory?
- ✅ Tests included?
- ✅ Documentation added?

---

## Common Use Cases

### Use Case 1: Onboarding New Team Members

**Without `.cursorrules`:**
```
New dev: "How do we handle errors?"
You: "Let me show you..."
(Repeat for each new person)
```

**With `.cursorrules`:**
```
New dev: Ask AI "How do we handle errors?"
AI: Shows exact pattern from .cursorrules
(Consistent answer every time!)
```

### Use Case 2: Maintaining Consistency

**Without `.cursorrules`:**
```
Dev A: Uses pattern X
Dev B: Uses pattern Y
AI: Generates inconsistent code
```

**With `.cursorrules`:**
```
Dev A: Gets pattern from .cursorrules
Dev B: Gets same pattern from .cursorrules
AI: Always generates consistent code
```

### Use Case 3: Preventing Mistakes

**Without `.cursorrules`:**
```
You: "Add a provider"
AI: Creates in wrong directory
AI: Forgets tests
AI: Uses wrong error pattern
```

**With `.cursorrules`:**
```
You: "Add a provider"
AI: Checks .cursorrules
AI: Correct directory ✓
AI: Tests included ✓
AI: Proper errors ✓
```

---

## Your Panka `.cursorrules`

I've created a comprehensive `.cursorrules` file for Panka that includes:

### ✅ What's Covered:

1. **Project Overview** - What Panka is
2. **Code Structure** - Where everything goes
3. **Go Standards** - Coding patterns
4. **Multi-Tenancy** - Critical isolation rules
5. **CLI Patterns** - Cobra command structure
6. **Documentation** - Where/how to document
7. **Testing** - Test requirements
8. **Security** - Credential handling
9. **AWS Integration** - SDK usage patterns
10. **State Management** - State/lock operations
11. **Common Patterns** - Factory, interface design
12. **Anti-Patterns** - What to avoid
13. **Quick Reference** - Cheat sheet

### 📋 Quick Reference Sections:

- File locations for common tasks
- Commands to run
- Questions to ask before committing
- Always/Never lists

---

## How to Update `.cursorrules`

### When to Update:

**✅ DO Update When:**
- Adding new architectural patterns
- Changing file structure
- Adding new coding standards
- Learning from mistakes (add anti-pattern)

**❌ DON'T Update For:**
- Temporary decisions
- Project-specific values (URLs, keys)
- Implementation details that change

### How to Update:

1. **Add new pattern:**
```
## New Pattern Name

### Context
Why this matters...

### Good Example
```go
// ✅ DO THIS
...
```

### Bad Example
```go
// ❌ DON'T DO THIS
...
```
```

2. **Test it:**
- Ask AI to use the new pattern
- Verify AI follows it correctly
- Adjust if needed

3. **Share with team:**
- Commit to git
- Everyone gets updates automatically

---

## Advantages of Cursor Rules

### 1. **Consistency**
✅ Every AI interaction follows same rules
✅ Code looks like it's from one person
✅ Standards enforced automatically

### 2. **Onboarding**
✅ New team members learn patterns from AI
✅ No need for separate style guide
✅ Interactive learning

### 3. **Quality**
✅ Prevents common mistakes
✅ Enforces best practices
✅ Tests generated automatically

### 4. **Documentation**
✅ Single source of truth
✅ Living document (update as you learn)
✅ Examples always current

### 5. **Productivity**
✅ Less time explaining patterns
✅ Less code review feedback
✅ Faster development

---

## Limitations & Caveats

### 1. **Token Limits**
- ⚠️ `.cursorrules` consumes context tokens
- Keep it focused (yours is ~2000 lines, that's good)
- Don't put entire codebases

### 2. **Not Enforcement**
- ⚠️ AI can still make mistakes
- Still need code review
- Use as guide, not guarantee

### 3. **Maintenance**
- ⚠️ Needs updating as project evolves
- Can become outdated
- Periodic review recommended

---

## Tips for Effective Use

### 1. **Start Conversations with Context**

**Good:**
```
"Add a new SNS provider"
```

AI will:
- Check .cursorrules for provider pattern
- Generate in correct location
- Follow all standards

**Even Better:**
```
"Add SNS provider following our AWS provider patterns"
```

AI will:
- Double-check provider patterns in rules
- Extra attention to consistency

### 2. **Reference Specific Sections**

```
"Add error handling following our security requirements"
"Create docs following our documentation standards"
"Write tests using our testing patterns"
```

### 3. **Ask AI to Review**

```
"Does this code follow our .cursorrules?"
"Review my implementation against project standards"
```

AI will check your code against rules!

---

## Example Workflows

### Workflow 1: Adding New Feature

```
You: "Add RDS provider"

AI (automatically):
1. Reads .cursorrules
2. Sees: providers go in pkg/provider/aws/
3. Sees: implement Provider interface
4. Sees: add tests
5. Sees: document in INDEX.md

AI generates:
✓ pkg/provider/aws/rds.go (correct location)
✓ pkg/provider/aws/rds_test.go (tests included)
✓ Following interface pattern
✓ With proper error handling
✓ With structured logging
```

### Workflow 2: Fixing Bug

```
You: "This error handling is wrong"

AI (automatically):
1. Reads .cursorrules
2. Sees: error handling patterns
3. Sees: logging requirements

AI fixes:
✓ Wraps error with context
✓ Adds structured logging
✓ Returns proper error type
```

### Workflow 3: Documentation

```
You: "Document the login flow"

AI (automatically):
1. Reads .cursorrules
2. Sees: user docs go in docs/quickstart/
3. Sees: update INDEX.md
4. Sees: documentation format

AI creates:
✓ docs/quickstart/LOGIN_FLOW.md (correct location)
✓ Updates INDEX.md (as required)
✓ Follows doc format (from rules)
```

---

## Summary

### ✅ Key Takeaways:

1. **`.cursorrules` is automatically used** - No need to mention it
2. **Put patterns, not implementation** - Guidelines, not code
3. **Show good and bad examples** - Help AI learn what to avoid
4. **Keep it organized** - Easy to reference
5. **Update as you learn** - Living document
6. **Test periodically** - Ensure AI follows rules
7. **Share with team** - Everyone benefits

### 🎯 Your Panka Rules Are Ready!

Your `.cursorrules` file now contains:
- ✅ Complete project structure
- ✅ All coding patterns
- ✅ Multi-tenancy requirements
- ✅ Testing standards
- ✅ Documentation guidelines
- ✅ Security requirements
- ✅ Quick reference

**Every AI interaction will now follow these rules automatically!**

---

## Quick Start

1. **File created**: `.cursorrules` ✓
2. **Start coding**: Just ask AI to add features
3. **AI will follow**: All patterns automatically
4. **No manual reminders**: It's automatic!

**Try it now:**
```
"Add a DynamoDB provider"
```

Watch AI follow all the rules! 🎉

