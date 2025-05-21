package saga

import (
	"time"
)

// ExecutionResult represents the result of a saga execution
type ExecutionResult struct {
	Success           bool          // Saga execution success status
	ExecutedSteps     []StepID      // Executed steps IDs
	FailedStepID      StepID        // Failed step ID (if any)
	OriginalError     error         // Original error that caused the saga to fail
	CompensationError error         // Compensation error (if any)
	CompensatedSteps  []StepID      // Steps compensated with success (if any)
	Duration          time.Duration // Duration total of the saga execution
}

func NewExecutionResult() *ExecutionResult {
	return &ExecutionResult{
		ExecutedSteps:    make([]StepID, 0),
		CompensatedSteps: make([]StepID, 0),
	}
}

// IsCompensated returns true if the saga execution was compensated successfully
func (r *ExecutionResult) IsCompensated() bool {
	if r.Success {
		return false
	}
	if len(r.ExecutedSteps) == 0 {
		return true
	}
	executedMap := make(map[StepID]bool)
	for _, id := range r.ExecutedSteps {
		executedMap[id] = true
	}
	for _, id := range r.CompensatedSteps {
		delete(executedMap, id)
	}
	return len(executedMap) == 0
}

// HasErrors returns true if the saga execution resulted in errors
func (r *ExecutionResult) HasErrors() bool {
	return r.OriginalError != nil || r.CompensationError != nil
}
