# How Workflows Work in Druppie Core

## 📋 Executive Summary

This document explains how Druppie Core's workflow system works, how users interact with it, and how it could be extended to enable full-stack application building.

---

## 🏗️ Current Architecture

### Two Types of Workflows

| Type | Location | Purpose | Example |
|------|-----------|---------|---------|
| **Native Workflows** | `core/internal/workflows/` | Hard-coded Go business logic | Video production, state machines |
| **Agent-Generated Plans** | `planner.CreatePlan()` | LLM-generated step sequences | Most tasks (create_repo, build_code, etc.) |

### How Agent-Generated Plans Work

Looking at [`core/druppie/task_manager.go`](core/druppie/task_manager.go:177):

```go
// Plans are just JSON step definitions
type ExecutionPlan struct {
    Steps []Step
    Intent Intent
}

// Execution happens via executors
func (tm *TaskManager) runTaskLoop(task *Task) {
    for {
        // Find runnable steps
        // Execute step via dispatcher
        // Handle results
    }
}
```

**Key Point**: Plans are just **JSON step definitions**. The actual execution happens via **executors**.

### How Executors Work

Looking at [`core/internal/executor/dispatcher.go`](core/internal/executor/dispatcher.go:51):

```go
type Dispatcher struct {
    executors []Executor
}

func (d *Dispatcher) GetExecutor(action string) (Executor, error) {
    for _, e := range d.executors {
        if e.CanHandle(action) {
            return e, nil
        }
    }
}
```

**Available Executors**:
- `MCPExecutor` - Calls MCP tools
- `DeveloperExecutor` - Creates files via `create_repo`
- `BuildExecutor` - Builds via Docker
- `RunExecutor` - Runs containers
- `WorkflowExecutor` - Executes native workflows
- `AudioCreatorExecutor`, `VideoCreatorExecutor`, `ImageCreatorExecutor` - AI content generation

### How Native Workflows Work

Looking at [`core/internal/workflows/manager.go`](core/internal/workflows/manager.go:1):

```go
type Workflow interface {
    Name() string
    Run(wc *WorkflowContext, initialPrompt string) error
}

type WorkflowContext struct {
    Ctx               context.Context
    LLM               llm.Provider
    Dispatcher        *executor.Dispatcher
    Store             store.Store
    PlanID            string
    OutputChan        chan<- string
    InputChan         <-chan string
    UpdateStatus      func(status string)
    UpdateTokenUsage  func(usage model.TokenUsage)
    AppendStep        func(step model.Step) int
    FindCompletedStep func(action string, paramKey string, paramValue interface{}) *model.Step
    GetAgent          func(id string) (model.AgentDefinition, error)
}
```

**Key Points**:
1. Workflows are **hard-coded Go functions** (like `VideoCreationWorkflow`)
2. They have full access to LLM, Dispatcher, Store
3. They can append steps to the plan
4. They run **asynchronously** and report progress via OutputChan

### Current Limitation for User-Created Workflows

**Problem**: When agents generate plans with actions like "create_ui" or "build_workflow", there's **no executor** that can handle them!

**Example**:
```json
{
  "action": "create_ui",
  "params": {
    "framework": "react",
    "components": [...]
  }
}
```

The dispatcher will return: `no executor found for action: create_ui`

---

## 🎯 Recommended Solution: Workflow Builder

### Option 1: Workflow Definition Language (Recommended)

Add a **workflow definition format** that agents can generate:

```yaml
---
id: my-workflow
name: "My Custom Workflow"
version: 1.0.0
type: workflow
steps:
  - name: "Scan Files"
    action: "mcp:filesystem_scanner__scan_directory"
    params:
      path: "/data"
      patterns: ["*.pdf"]
  - name: "Classify Documents"
    action: "mcp:document_processor__classify_document"
    params:
      text: "${scan_files.result}"
  - name: "Redact PII"
    action: "mcp:anonymizer__redact_bsn"
    params:
      text: "${classify.result}"
  - name: "Create Case"
    action: "mcp:zaaksysteem__create_case"
    params:
      case_type: "${classify.document_type}"
```

**Benefits**:
- ✅ Declarative and version-controlled
- ✅ Can be shared across projects
- ✅ Easy to understand and modify
- ✅ Can be executed by a new `WorkflowExecutor`

### Option 2: UI Builder MCP Server (Alternative)

Create an MCP server that generates UI components:

```yaml
---
id: ui-builder
name: "UI Builder MCP"
type: mcp
tools:
  - name: generate_react_component
    description: "Generate React component code"
    inputSchema:
      type: object
      properties:
        component_type:
          type: string
        props:
          type: object
  - name: generate_vue_component
    description: "Generate Vue component code"
```

**Benefits**:
- ✅ Language-agnostic (Node.js)
- ✅ Can be called from agents
- ✅ Generates production-ready code

### Option 3: Workflow Composition (Advanced)

Allow workflows to include other workflows:

```yaml
---
id: composite_workflow
name: "Composite Workflow"
version: 1.0.0
type: workflow
steps:
  - name: "Run Sub-Workflow"
    action: "workflow:execute"
    params:
      workflow_id: "document-scanner"
```

---

## 🔄 How Users Would Interact

### Scenario 1: User Creates a Workflow via Chat

**User Input** (in Druppie UI Chat):
```
Create a workflow that scans files and classifies documents
```

**What Happens**:
1. **Planner** receives the request and generates a plan
2. **Planner** creates a new workflow definition file (e.g., `.druppie/workflows/my-workflow.md`)
3. **WorkflowManager** registers the new workflow
4. **TaskManager** can now execute steps with `action: "workflow:execute"`

### Scenario 2: User Executes a Workflow

**User Input**:
```
Execute workflow: document-scanner
```

**What Happens**:
1. **TaskManager** receives the step
2. **WorkflowExecutor** is called with `workflow_id: "document-scanner"`
3. **WorkflowManager** finds the workflow and calls its `Run()` method
4. The workflow executes its steps (scan, classify, etc.) and reports progress

### Scenario 3: User Views Workflow Progress

**User Action**: Click on a workflow in the UI

**What Happens**:
1. UI queries the plan status
2. TaskManager returns the current plan state
3. UI shows the workflow steps and their status

---

## 🚀 Implementation Plan for Workflow Builder

### Phase 1: Workflow Definition Format

**File**: `core/internal/workflows/definition.go` (NEW)

```go
package workflows

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type WorkflowDefinition struct {
    ID          string
    Name        string
    Version     string
    Description string
    Steps       []WorkflowStep
    CreatedAt   time.Time
    CreatedBy   string
}

type WorkflowStep struct {
    ID          string
    Name        string
    Action      string
    ActionMCP   string // e.g., "mcp:filesystem_scanner__scan_directory"
    Params      map[string]interface{}
    DependsOn   []string // Step IDs
}
```

### Phase 2: Workflow Loader

**File**: `core/internal/workflows/loader.go` (NEW)

```go
func LoadUserWorkflows() ([]WorkflowDefinition, error) {
    // Load from .druppie/workflows/
    dir := filepath.Join(paths.GetDataDir(), "workflows")
    files, _ := os.ReadDir(dir)
    
    var workflows []WorkflowDefinition
    for _, file := range files {
        if filepath.Ext(file) == ".md" {
            data, _ := os.ReadFile(filepath.Join(dir, file))
            var wf WorkflowDefinition
            json.Unmarshal(data, &wf)
            workflows = append(workflows, wf)
        }
    }
    
    return workflows, nil
}
```

### Phase 3: Workflow Executor Enhancement

**File**: `core/internal/executor/workflow_executor.go` - UPDATE

Add support for workflow definitions:

```go
func (e *WorkflowExecutor) Execute(ctx context.Context, step model.Step, outputChan chan<- string) error {
    // Check if this is a workflow execution step
    if step.Action == "workflow:execute" {
        workflowID := step.Params["workflow_id"].(string)
        
        // Load workflow definition
        wf, err := workflows.LoadUserWorkflows()
        if err != nil {
            return err
        }
        
        // Execute workflow steps
        for _, wfStep := range wf.Steps {
            // Execute each step
            result, err := e.executeWorkflowStep(ctx, wfStep, outputChan)
            if err != nil {
                return err
            }
            
            // Append result to workflow output
        }
        
        return nil
    } else {
        // Existing behavior for native workflows
        return e.WorkflowManager.StartStateMachine(...)
    }
}
```

### Phase 4: UI Integration

**File**: `core/api/workflows.go` (NEW)

```go
package api

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/sjhoeksma/druppie/core/internal/workflows"
)

func RegisterRoutes(r chi.Router, wm *workflows.Manager) {
    r.Route("/v1/workflows", func(w http.ResponseWriter, r *http.Request) {
        workflows, _ := workflows.LoadUserWorkflows()
        json.NewEncoder(w).Encode(workflows)
    })
    
    r.Post("/v1/workflows", func(w http.ResponseWriter, r *http.Request) {
        var wf WorkflowDefinition
        json.NewDecoder(r.Body).Decode(&wf)
        
        // Save workflow definition
        workflows.SaveUserWorkflow(wf)
        
        w.WriteHeader(http.StatusCreated)
    })
    
    r.Post("/v1/workflows/{id}/execute", func(w http.ResponseWriter, r *http.Request) {
        id := chi.URLParam(r, "id")
        
        // Trigger workflow execution
        planID := r.URL.Query().Get("plan_id")
        
        // Create a step in the plan
        step := model.Step{
            Action:  "workflow:execute",
            Params: map[string]interface{}{
                "workflow_id": id,
            },
        }
        
        // Submit to TaskManager
        // ... (implementation depends on TaskManager API)
    })
}
```

---

## 📊 Comparison: Current vs Proposed

| Aspect | Current | With Workflow Builder |
|--------|---------|---------------------|
| **Workflow Definition** | Hard-coded in Go | Declarative YAML files |
| **User Workflows** | Only native workflows | User-defined workflows via YAML |
| **Extensibility** | Requires code changes | Add YAML files |
| **UI Integration** | None | REST API for workflow management |
| **Version Control** | Requires full rebuild | Git-tracked YAML files |
| **Testing** | Requires full rebuild | Can test independently |

---

## 🎯 Success Criteria

The workflow builder is successful when:

1. ✅ Users can create workflow definitions via YAML
2. ✅ Users can execute workflows via API
3. ✅ Workflows can use MCP tools
4. ✅ Workflows can be version-controlled
5. ✅ UI can display and trigger workflows

---

## 📝 Example User Workflow

```yaml
---
id: vergunning-scanner
name: "Vergunning Scanner Workflow"
version: 1.0.0
description: "Scans SMB shares for permit documents and processes them"
steps:
  - id: step-1
    name: "Scan Directory"
    action: "mcp:filesystem_scanner__scan_directory"
    params:
      path: "/mnt/s-share"
      patterns: ["*vergunning*", "*beschikking*", "*.pdf"]
      max_depth: 3
  
  - id: step-2
    name: "Classify Documents"
    action: "mcp:document_processor__classify_document"
    params:
      text: "${step-1.result}"
  
  - id: step-3
    name: "Detect PII"
    action: "mcp:anonymizer__detect_pii"
    params:
      text: "${step-2.result}"
  
  - id: step-4
    depends_on: ["step-3"]
    name: "Redact PII"
    action: "mcp:anonymizer__redact_bsn"
    params:
      text: "${step-3.result}"
      replacement: "[BSN VERWIJDERD]"
  
  - id: step-5
    depends_on: ["step-4"]
    name: "Create Case"
    action: "mcp:zaaksysteem__create_case"
    params:
      case_type: "${step-2.document_type}"
      title: "Vergunning ${step-2.metadata.perceel}"
```

---

## 🔗 Integration Points

| Component | Integration Required |
|-----------|------------------|
| `WorkflowManager` | Add workflow definition loading/saving |
| `WorkflowExecutor` | Add workflow definition execution |
| `Dispatcher` | Add `workflow:execute` action handling |
| `TaskManager` | Add workflow execution support |
| `API` | Add workflow management endpoints |
| `Registry` | Add workflow definitions to load path |

---

## 📚 References

- [`core/internal/workflows/manager.go`](core/internal/workflows/manager.go:1)
- [`core/internal/executor/workflow_executor.go`](core/internal/executor/workflow_executor.go:1)
- [`core/internal/workflows/definition.go`](core/internal/workflows/definition.go) - NEW
- [`core/internal/workflows/loader.go`](core/internal/workflows/loader.go) - NEW
- [`core/api/workflows.go`](core/api/workflows.go) - NEW
