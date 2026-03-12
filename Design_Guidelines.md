# Design Guidelines: Resource-Oriented Design for Go Projects

This document codifies design conventions for Go services and APIs using **resource-oriented design**, inspired by Google API Improvement Proposals (AIPs).

These rules help both **humans and AI agents** produce consistent architectures across:

- APIs
- services
- repositories
- workflows
- protobuf definitions
- HTTP handlers
- background workers

The goal is to model systems around **resources (nouns)** rather than **actions (verbs)**, producing interfaces that are easier to understand, automate, and evolve.

Primary references:

- AIP-121 Resource-oriented design
- AIP-122 Resource names
- AIP-123 Resource types
- AIP-124 Resource associations
- AIP-128 Declarative-friendly interfaces
- AIP-130–136 Standard methods
- AIP-148 Standard fields
- AIP-156 Singleton resources
- AIP-161 Field masks

---

# 1. Core Principles

All designs must follow these rules.

### 1.1 Design around resources

Systems should be modeled around **domain nouns**.

Good:

```

Agent
Run
Task
Session
Workflow

```

Bad:

```

ExecuteWorkflow
ProcessTask
HandleJob

```

If something is durable or meaningful in the domain, it is likely a **resource**.

---

### 1.2 Resources should have stable identities

Each resource must have:

- a **canonical identifier**
- a **stable resource name**
- a **clear parent resource**

Example:

```

projects/{project}/agents/{agent}
projects/{project}/agents/{agent}/runs/{run}

```

---

### 1.3 Prefer standard lifecycle methods

Before creating custom actions, use the standard methods:

| Method | Meaning                  |
| ------ | ------------------------ |
| Get    | Fetch one resource       |
| List   | Enumerate resources      |
| Create | Create new resource      |
| Update | Modify existing resource |
| Delete | Remove resource          |

Custom actions should only exist when lifecycle semantics do not fit.

---

### 1.4 Prefer declarative and automation-friendly interfaces

Design resources so automation and agents can operate on them.

Prefer:

```

CreateRun
UpdateRun(state=running)

```

Instead of:

```

ExecuteWorkflow
ProcessTask

```

---

# 2. Resource Modeling Rules

### 2.1 Every domain noun should be evaluated as a resource

Example resources:

```

Agent
Workflow
Run
Task
Session
Tool

```

Avoid turning meaningful domain entities into anonymous blobs.

---

### 2.2 Resources must have a single canonical parent

Good:

```

projects/{project}/agents/{agent}

```

Avoid:

```

projects/{project}/agents/{agent}
organizations/{org}/agents/{agent}

```

Cross references should be **fields**, not parents.

---

### 2.3 Many-to-many relationships

Use one of:

**Reference lists**

```

task.assignees = [user_id]

```

**Association resources**

```

projects/{project}/taskAssignments/{assignment}

```

Use association resources if the relationship has metadata.

---

### 2.4 Singleton resources

Some resources exist only once per parent.

Example:

```

projects/{project}/settings
projects/{project}/agentConfig

```

These are **singleton resources**, not collections.

---

# 3. Resource Naming

### 3.1 Naming rules

Resource paths should alternate:

```

collection / identifier / collection / identifier

```

Example:

```

projects/{project}/agents/{agent}
projects/{project}/agents/{agent}/runs/{run}

```

---

### 3.2 Identifiers

Identifiers should be:

- URL safe
- stable
- lowercase when possible

Example:

```

agent_123
run_456

```

Avoid random names that leak internal storage implementation.

---

### 3.3 Resource name fields

Resources should expose a `name` field containing the canonical resource path.

Example:

```

name: "projects/demo/agents/a1"

```

---

# 4. Standard Method Mapping

All services should follow these conventions.

### RPC and HTTP verb mapping

| Method     | HTTP Verb | RPC Name           | Requirement                                  |
| :--------- | :-------- | :----------------- | :------------------------------------------- |
| **List**   | GET       | `List<Resources>`  | Must support `page_size` and `page_token`.   |
| **Get**    | GET       | `Get<Resource>`    | Must return the specific resource message.   |
| **Create** | POST      | `Create<Resource>` | Must take a parent and the resource message. |
| **Update** | PATCH     | `Update<Resource>` | Must use `google.protobuf.FieldMask`.        |
| **Delete** | DELETE    | `Delete<Resource>` | Should return `google.protobuf.Empty`.       |

### Get

Fetch one resource.

Example:

```

GetAgent(name)
GetRun(name)

```

---

### List

Enumerate resources under a parent.

Example:

```

ListAgents(parent)
ListRuns(parent)

```

---

### Create

Create a resource in a collection.

Example:

```

CreateAgent(parent, agent)
CreateRun(parent, run)

```

---

### Update

Modify an existing resource.

Prefer partial updates.

Example:

```

UpdateAgent(agent)
UpdateRun(run)

```

---

### Delete

Remove a resource.

Example:

```

DeleteAgent(name)
DeleteRun(name)

```

---

# 5. Custom Methods

Use custom methods only when lifecycle semantics do not apply.

Examples:

```

CancelRun
ApprovePlan
RetryTask

```

Avoid:

```

ExecuteWorkflow
HandleAgent
ProcessTask

```

If an action creates a durable entity, that entity should be modeled as a **resource**.

Example:

Bad:

```

ExecuteWorkflow

```

Better:

```

CreateRun

```

---

# 6. Standard Resource Fields

Durable resources should include common metadata.

Recommended fields:

```

name
uid
display_name
create_time
update_time
delete_time (optional)
annotations (optional)
etag (optional)

```

Descriptions:

| Field        | Purpose                 |
| ------------ | ----------------------- |
| name         | canonical resource name |
| uid          | immutable identifier    |
| display_name | human friendly label    |
| create_time  | creation timestamp      |
| update_time  | last modification time  |
| delete_time  | soft deletion timestamp |
| annotations  | system metadata         |
| etag         | concurrency control     |

---

# 7. Update Semantics

Updates should be predictable.

Rules:

- Updates should modify only intended fields.
- Updates must use `google.protobuf.FieldMask` for partial updates.
- Output-only fields must not be mutable.
- APIs should support partial updates where possible.

Avoid ambiguous upserts.

Prefer explicit update requests.

---

# 7.5. Protobuf Best Practices

### Field naming

- All field names in `.proto` files must be `snake_case`.

### Enums

- Every enum must have a `0` value named `<TYPE>_UNSPECIFIED`.

### Resource annotations

- Use `google.api.resource` and `google.api.http` annotations.

```proto
// Example Resource Definition
message Book {
  option (google.api.resource) = {
    type: "library.googleapis.com/Book"
    pattern: "publishers/{publisher}/books/{book}"
  };
  string name = 1;
  string title = 2;
}
```

### ID fields

- Do not use `id` or `uuid` fields as primary identifiers in Proto. Use the `name` field.

---

# 8. Declarative Design for Agentic Workflows

Agent systems benefit from declarative models.

Preferred approach:

```

desired_state
observed_state
status
phase

```

Example run resource:

```

Run {
name
workflow
state
create_time
update_time
}

```

Agents interact by:

```

CreateRun
UpdateRun
CancelRun

```

Not by opaque imperative commands.

---

# 9. Go Project Structure

Resources should map to Go packages.

Example:

```

internal/
agent/
workflow/
run/
task/

```

Service interfaces:

```

AgentService
RunService
TaskService

```

Repositories:

```

AgentStore
RunStore
TaskStore

```

Handlers mirror resource hierarchy.

### Separation of concerns

- gRPC service handlers should handle request validation and Proto-to-Internal mapping.
- Business logic belongs in internal packages, not in generated code.
- Use `google.golang.org/grpc/status` with standard codes (e.g., `codes.NotFound`) for errors.

---

# 10. Anti-Patterns

Avoid these patterns.

### Verb-centric APIs

Bad:

```

ExecuteWorkflow
ProcessJob
HandleAgent

```

---

### Multiple canonical parents

Bad:

```

orgs/{org}/agents/{agent}
projects/{project}/agents/{agent}

```

---

### Mixing identifiers with display names

IDs must be stable.

Display names are mutable.

---

### Inconsistent naming across layers

Do not rename the same concept across layers.

Example:

```

Agent
AgentEntity
AgentRecord
AgentDTO

```

Prefer one canonical term.

---

# 11. Review Checklist

Before implementing a feature, verify:

- What is the **resource**?
- What is its **canonical parent**?
- What is the **resource name format**?
- Does it use **standard lifecycle methods**?
- Is a **custom action truly necessary**?
- Are resource **fields consistent**?
- Are **update semantics explicit**?
- Is the design **automation friendly**?
- Are **names consistent across code layers**?

---

# 12. Guidance for AI Agents

Agents generating Go code should:

1. Identify domain nouns.
2. Model them as resources.
3. Assign canonical resource names.
4. Implement standard lifecycle methods.
5. Use custom actions only when necessary.
6. Keep naming consistent across API, service, repository, and storage layers.
7. Prefer declarative models for long-running workflows.

```

```
