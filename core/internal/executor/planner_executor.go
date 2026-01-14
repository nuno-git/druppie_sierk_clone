package executor

import (
	"context"

	"github.com/sjhoeksma/druppie/core/internal/model"
)

// PlannerExecutor handles internal planner actions like "replanning"
// These are not actual agent actions that should be executed by an executor,
// but rather internal operations of the planner itself.
type PlannerExecutor struct {
	LLMProvider interface{}
}

// NewPlannerExecutor creates a new planner executor
func NewPlannerExecutor(llmProvider interface{}) *PlannerExecutor {
	return &PlannerExecutor{
		LLMProvider: llmProvider,
	}
}

// CanHandle checks if this executor can handle a given action
func (e *PlannerExecutor) CanHandle(action string) bool {
	// Planner executor handles internal planner actions
	return action == "replanning"
}

// Execute handles the planner action
// Since replanning is an internal operation, we don't execute anything.
// We just return nil to indicate success.
func (e *PlannerExecutor) Execute(ctx context.Context, step model.Step, outputChan chan<- string) error {
	// Replanning is an internal planner operation, not an agent action
	// Return success without executing anything
	return nil
}
