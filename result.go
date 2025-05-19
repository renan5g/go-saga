package saga

import (
	"time"
)

// ExecutionResult contém o resultado da execução de uma saga
type ExecutionResult struct {
	Success           bool
	FailedStepID      StepID
	OriginalError     error
	CompensationError error
	ExecutedSteps     []StepID
	CompensatedSteps  []StepID
	Duration          time.Duration
}

// IsCompensationFailed verifica se ocorreu falha na compensação
func (er *ExecutionResult) IsCompensationFailed() bool {
	return er.CompensationError != nil
}

// HasExecutedSteps verifica se algum passo foi executado com sucesso
func (er *ExecutionResult) HasExecutedSteps() bool {
	return len(er.ExecutedSteps) > 0
}

// HasCompensatedSteps verifica se algum passo foi compensado
func (er *ExecutionResult) HasCompensatedSteps() bool {
	return len(er.CompensatedSteps) > 0
}

// NewExecutionResult cria um novo resultado vazio
func NewExecutionResult() *ExecutionResult {
	return &ExecutionResult{
		ExecutedSteps:    make([]StepID, 0),
		CompensatedSteps: make([]StepID, 0),
	}
}
