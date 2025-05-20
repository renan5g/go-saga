package saga

import (
	"time"
)

// ExecutionResult contém o resultado da execução de uma saga
type ExecutionResult struct {
	Success           bool          // Indicador de sucesso
	ExecutedSteps     []StepID      // Passos executados com sucesso
	FailedStepID      StepID        // ID do passo que falhou (se houver)
	OriginalError     error         // Erro original que causou a falha
	CompensationError error         // Erro durante compensação (se houver)
	CompensatedSteps  []StepID      // Passos compensados com sucesso
	Duration          time.Duration // Duração total da execução
}

// NewExecutionResult cria um novo resultado de execução
func NewExecutionResult() *ExecutionResult {
	return &ExecutionResult{
		ExecutedSteps:    make([]StepID, 0),
		CompensatedSteps: make([]StepID, 0),
	}
}

// IsCompensated verifica se todos os passos executados foram compensados
func (r *ExecutionResult) IsCompensated() bool {
	if r.Success {
		return false
	}

	// Se não há passos executados, não há o que compensar
	if len(r.ExecutedSteps) == 0 {
		return true
	}

	// Verifica se todos os passos executados foram compensados
	executedMap := make(map[StepID]bool)
	for _, id := range r.ExecutedSteps {
		executedMap[id] = true
	}

	// Remove passos compensados do mapa
	for _, id := range r.CompensatedSteps {
		delete(executedMap, id)
	}

	// Se o mapa estiver vazio, todos foram compensados
	return len(executedMap) == 0
}

// HasErrors verifica se ocorreram erros na execução ou compensação
func (r *ExecutionResult) HasErrors() bool {
	return r.OriginalError != nil || r.CompensationError != nil
}
