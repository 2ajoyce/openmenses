# Go Code Guidelines for Agentic Projects

This document defines implementation rules for writing Go code in this repository.

The goals are:

- readability
- consistency
- predictable behavior for AI agents
- maintainable service architecture

These guidelines are derived primarily from:

- Uber Go Style Guide
- Effective Go
- Go Proverbs

Agents generating Go code **must follow these rules**.

---

# 1. Core Philosophy

Go code should prioritize:

1. Simplicity
2. Explicit behavior
3. Small abstractions
4. Readable control flow
5. Predictable naming

Prefer **clear code over clever code**.

Bad:

```
func Process(items []Item) []Result
```

Better:

```
func BuildResults(items []Item) ([]Result, error)
```

---

# 2. Package Design

Packages should be organized around **domain concepts**, not technical layers.

Good:

```
internal/
  agent/
  run/
  workflow/
  task/
```

Bad:

```
internal/
  controllers/
  handlers/
  models/
  services/
```

Each package should represent a **resource or domain concept**.

---

# 3. Package Size

Packages should remain small and focused.

Guidelines:

- 1 domain concept per package
- ideally <10 files
- avoid large "utility" packages

If a package exceeds ~1000 lines, consider splitting it.

---

# 4. Package Naming

Follow standard Go conventions.

Rules:

- lowercase
- no underscores
- no plurals unless natural
- short and descriptive

Good:

```
agent
run
task
workflow
```

Bad:

```
agent_service
workflow_manager
task_handlers
```

---

# 5. File Naming

Files should reflect the concept they implement.

Examples:

```
agent.go
service.go
store.go
handler.go
types.go
```

Tests:

```
agent_test.go
service_test.go
```

---

# 6. Interface Design

Interfaces should be **small and behavior-focused**.

Prefer:

```
type RunStore interface {
    Get(ctx context.Context, id string) (*Run, error)
    List(ctx context.Context, agentID string) ([]Run, error)
    Create(ctx context.Context, run *Run) error
}
```

Avoid large interfaces.

Bad:

```
type RunManager interface {
    CreateRun(...)
    UpdateRun(...)
    CancelRun(...)
    RetryRun(...)
    ValidateRun(...)
    ExecuteRun(...)
}
```

Define interfaces **where they are used**, not where implemented.

---

# 7. Struct Design

Structs represent domain resources.

Example:

```
type Run struct {
    ID        string
    AgentID   string
    State     RunState
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Guidelines:

- avoid unnecessary getters/setters
- export fields only when needed
- keep structs simple

---

# 8. Constructor Functions

Use constructors when initialization logic exists.

Example:

```
func NewService(store RunStore) *Service {
    return &Service{
        store: store,
    }
}
```

Avoid complex constructors.

---

# 9. Dependency Injection

Dependencies should be passed explicitly.

Prefer:

```
service := run.NewService(store)
```

Avoid:

- global variables
- implicit dependencies
- service locators

---

# 10. Error Handling

Errors must be handled explicitly.

Example:

```
run, err := store.Get(ctx, id)
if err != nil {
    return nil, err
}
```

Never ignore errors.

Bad:

```
run, _ := store.Get(ctx, id)
```

---

# 11. Error Messages

Error messages should:

- start lowercase
- not end with punctuation
- include context

Good:

```
return fmt.Errorf("fetch run: %w", err)
```

Bad:

```
return fmt.Errorf("Failed to fetch run.")
```

---

# 12. Context Usage

All I/O or request-based functions must accept `context.Context`.

Example:

```
func (s *Service) GetRun(ctx context.Context, id string) (*Run, error)
```

Rules:

- `context.Context` must be the **first parameter**
- never store context in structs

---

# 13. Logging

Logging should be structured and contextual.

Example:

```
logger.Info("run started",
    "run_id", run.ID,
    "agent_id", run.AgentID,
)
```

Avoid excessive logging.

Log events that matter.

---

# 14. Control Flow

Prefer simple control flow.

Bad:

```
if cond {
    if other {
        ...
    }
}
```

Better:

```
if !cond {
    return nil
}

if !other {
    return nil
}
```

---

# 15. Avoid Premature Abstractions

Only introduce abstractions when necessary.

Bad:

```
type Processor interface {
    Process()
}
```

Better:

```
func ProcessTask(...)
```

---

# 16. Avoid Utility Packages

Avoid large `util` or `helpers` packages.

If functionality is domain-specific, place it in the domain package.

---

# 17. Test Structure

Tests should mirror package structure.

```
run/
  run.go
  service.go
  store.go
  service_test.go
```

Prefer table-driven tests.

Example:

```
tests := []struct{
    name string
    input int
    want int
}{
    {"basic",1,2},
}
```

---

# 18. Avoid Deep Nesting

Prefer early returns.

Bad:

```
if err == nil {
    if ok {
        ...
    }
}
```

Better:

```
if err != nil {
    return err
}

if !ok {
    return nil
}
```

---

# 19. Constants

Use typed constants when possible.

Example:

```
type RunState string

const (
    RunPending RunState = "pending"
    RunRunning RunState = "running"
)
```

---

# 20. Generics

Generics should be used sparingly.

Use them when they:

- reduce duplication
- remain readable

Avoid clever generic abstractions.

---

# 21. Concurrency

Use goroutines only when necessary.

Rules:

- prefer simple synchronous code
- manage goroutines with context
- avoid goroutine leaks

Example:

```
go worker(ctx)
```

Always support cancellation.

---

# 22. Documentation

Public types and functions must have doc comments.

Example:

```
 // Run represents an execution of a workflow.
 type Run struct {
```

---

# 23. Naming Conventions

Prefer short, meaningful names.

Good:

```
Run
Agent
Task
Store
Service
```

Avoid redundant names:

```
RunRecord
RunEntity
RunObject
```

---

# 24. Code Generation Rules for Agents

Agents generating Go code should:

1. follow package-per-resource structure
2. generate small interfaces
3. inject dependencies explicitly
4. implement explicit error handling
5. keep control flow simple
6. avoid unnecessary abstractions
7. align names with resource models from `Design_Guidelines.md`

Agents should prefer **boring Go code** over clever solutions.

---

# 25. Final Checklist

Before committing code, verify:

- package follows domain concept
- interfaces are small
- errors handled explicitly
- context passed properly
- naming consistent with resource model
- no unnecessary abstractions
- code is readable without comments

If unsure, choose the **simplest implementation**.
