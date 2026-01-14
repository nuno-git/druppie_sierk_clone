# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Druppie** is a spec-driven AI orchestration platform for government contexts (Water Board/Municipality). It combines AI agents with human-in-the-loop workflows, emphasizing security, privacy (GDPR), and compliance (BIO/NIS2/AI Act). The architecture follows a "Build Plane vs Runtime" model where AI agents transform specifications into running software.

### Core Architecture

The system consists of:

1. **Core Engine** (`core/`) - Go-based backend with modular components:
   - **Router** (`internal/router`): LLM-based intent analysis
   - **Planner** (`internal/planner`): Generates execution plans as DAGs
   - **Executor** (`internal/executor`): Dispatches steps to appropriate handlers
   - **MCP Manager** (`internal/mcp`): Model Context Protocol server management
   - **Registry** (`internal/registry`): Source of truth for all capabilities
   - **Store** (`internal/store`): File-based persistence in `.druppie/`

2. **Request Flow**: User intent → Router → Planner → Task Manager → Executor (MCP/Workflow/Plugin)

3. **UI Layer** (`ui/`) - Progressive Web App with chat interface and Kanban board

4. **Specs & Agents**: Agents (`agents/`) with Skills (`skills/`) operate on Building Blocks (`blocks/`) based on Compliance rules (`compliance/`)

## Development Commands

### Local Development

```bash
# Navigate to core directory first for Go commands
cd core

# Run server (from core directory)
go run ./druppie serve

# Build CLI tool
go build -o druppie ./druppie

# Run CLI directly
go run ./druppie

# Server with demo mode (skip authentication)
go run ./druppie serve --demo
```

### Docker Development

```bash
# Build from repository root
docker build -t druppie .

# Run with persistent storage (recommended)
docker run -d \
  -p 8080:80 \
  -v $(pwd)/.druppie:/app/.druppie \
  --name druppie-server \
  druppie

# Access UI at http://localhost:8080
```

### CLI Commands

```bash
./druppie serve                  # Start REST API server
./druppie chat                   # Interactive chat mode
./druppie run "prompt"           # Generate and execute plan
./druppie resume <plan-id>       # Resume stopped/crashed plan
./druppie registry               # List all capabilities
./druppie login                  # Authenticate (if not in demo mode)
./druppie mcp list               # List MCP servers
./druppie mcp add <name> <url>   # Add MCP server
```

### Testing

```bash
# From core directory
cd core
go test ./...
go test ./internal/planner       # Test specific package
go test -v ./internal/...        # Verbose output
```

## Configuration

Configuration is loaded from `.druppie/config.yaml` (auto-generated) with defaults from `core/config_default.yaml`. Environment variables override YAML settings.

### Key Configuration Options

```yaml
llm:
  default_provider: zai          # gemini, ollama, openrouter, zai
  timeout_seconds: 120
  providers:
    gemini:
      model: gemini-2.5-flash
      # OAuth setup required (see core/README.md)

build:
  default_provider: docker       # docker, local, tekton

iam:
  provider: local                # local, keycloak, demo

general:
  max_unattended_cost: 1.0       # Euro threshold for auto-pause
  server_port: "8080"
```

## Architecture Patterns

### Spec-Driven Development
Everything is defined through specifications in markdown files. Agents execute based on specifications, not ad-hoc commands, ensuring consistency and reproducibility.

### Human-in-the-Loop
- Critical steps require human approval (configured via `approval_groups`)
- Kanban board shows workflow status
- File uploads provide task context

### Multi-Provider Architecture
- **LLM Providers**: Gemini, Ollama, OpenRouter, Z.AI (configured in `llm.providers`)
- **Build Providers**: Docker, Local, Tekton
- **IAM Providers**: Local, Keycloak, Demo
- **Git Providers**: Internal Gitea or external GitHub/GitLab

### Model Context Protocol (MCP)
- **Static Servers**: Predefined in registry (e.g., filesystem access)
- **Dynamic Servers**: Plan-scoped, auto-provisioned from templates in `mcp/`
- Each plan gets isolated MCP context for security

## Working with the Codebase

### Adding New Features

**For Go-based logic (workflows):**
1. Create file in `core/internal/workflows/`
2. Implement `Workflow` interface (see `dev-doc/workflows.md`)
3. Register in `core/internal/workflows/registry.go`

**For agent capabilities:**
1. Add agent definition in `agents/*.md` (frontmatter with metadata)
2. Define skills/tools that map to MCP tools or native executors
3. Registry loads these automatically at startup

**For infrastructure blocks:**
1. Add definition in `blocks/` directory
2. Include compliance metadata and dependencies
3. Register in appropriate category (security, data, gis, observability)

### Module Structure

The `core/go.mod` uses local replace directives for internal packages:
```
github.com/sjhoeksma/druppie/core/internal/* => ./internal/*
```

When working with internal packages, import paths are:
```go
import "github.com/sjhoeksma/druppie/core/internal/planner"
```

### Key Files to Understand

- `core/internal/planner/planner.go:123` - Plan generation logic
- `core/internal/executor/dispatcher.go:89` - Step routing to handlers
- `core/druppie/task_manager.go:45` - Background execution loop
- `core/internal/registry/registry.go` - Capability loading

## Persistence

The system uses file-based storage in `.druppie/`:
- `plans/<id>/plan.json` - Execution plans
- `plans/<id>/logs/` - Execution logs
- `config.yaml` - Runtime configuration
- `iam/users.json` - Local user store (if using local IAM)

## Important Notes

- **Always run from project root or use `cd core` first** for Go commands
- **OAuth for Gemini** requires Google Cloud Console setup (see `core/README.md:26`)
- **Demo mode** (`--demo` flag or `iam.provider: demo`) skips authentication for testing
- **Plans can be resumed** after crashes using the plan ID shown in logs
- **MCP templates** in `mcp/` directory are automatically provisioned when plans reference tools that need them
- **Token usage and costs** are tracked per plan to enforce `max_unattended_cost` limits

## See Also

- `dev-doc/README.md` - Technical documentation index
- `dev-doc/architecture.md` - Detailed system architecture
- `dev-doc/workflows.md` - Workflow development guide
- `dev-doc/extensions.md` - Adding executors
- `dev-doc/mcp.md` - MCP server configuration
