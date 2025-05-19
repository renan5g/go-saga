package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Saga implementa o padrão saga para transações distribuídas
type Saga struct {
	steps       []Step
	ctx         context.Context
	mu          sync.RWMutex
	config      SagaConfig
	hasExecuted bool
	result      *ExecutionResult
}

// NewSaga cria uma nova instância de Saga
func NewSaga(ctx context.Context, options ...SagaOption) *Saga {
	saga := &Saga{
		ctx:    ctx,
		steps:  make([]Step, 0),
		config: DefaultSagaConfig(),
		result: NewExecutionResult(),
	}
	for _, option := range options {
		option(saga)
	}
	return saga
}

// AddStep adiciona um novo passo à saga
func (s *Saga) AddStep(step Step) *Saga {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Gerar ID automático se não fornecido
	if step.ID == "" {
		step.ID = StepID(fmt.Sprintf("step-%d", len(s.steps)))
	}

	// Configurar valores padrão se necessário
	if step.Status == "" {
		step.Status = StatusPending
	}

	if step.CreatedAt.IsZero() {
		step.CreatedAt = time.Now()
	}

	// Usar política de retry padrão se disponível e não especificada no step
	if step.RetryPolicy == nil && s.config.DefaultRetryPolicy != nil {
		step.RetryPolicy = s.config.DefaultRetryPolicy
	}

	if step.Metadata == nil {
		step.Metadata = make(map[string]any)
	}

	s.steps = append(s.steps, step)
	s.logf("Added step: %s", step.ID)
	return s
}

// GetStepStatus retorna o status atual de um passo
func (s *Saga) GetStepStatus(id StepID) (StepStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, step := range s.steps {
		if step.ID == id {
			return step.Status, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrStepNotFound, id)
}

// GetSteps retorna uma cópia dos passos atuais
func (s *Saga) GetSteps() []Step {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stepsCopy := make([]Step, len(s.steps))
	copy(stepsCopy, s.steps)
	return stepsCopy
}

// GetStepByID retorna um passo específico pelo ID
func (s *Saga) GetStepByID(id StepID) (Step, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, step := range s.steps {
		if step.ID == id {
			return step, nil
		}
	}
	return Step{}, fmt.Errorf("%w: %s", ErrStepNotFound, id)
}

// GetResult retorna o resultado da execução da saga
func (s *Saga) GetResult() *ExecutionResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.result == nil {
		return NewExecutionResult()
	}
	resultCopy := *s.result
	return &resultCopy
}

// IsExecuted verifica se a saga já foi executada
func (s *Saga) IsExecuted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hasExecuted
}

// IsSuccessful verifica se a saga foi executada com sucesso
func (s *Saga) IsSuccessful() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hasExecuted && s.result != nil && s.result.Success
}

// Execute executa todos os passos da saga em sequência
func (s *Saga) Execute() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime := time.Now()

	// Verificar se a saga já foi executada
	if s.hasExecuted {
		return ErrSagaAlreadyExecuted
	}

	s.hasExecuted = true
	s.logf("Starting saga execution with %d steps", len(s.steps))

	// Executar cada passo
	for i := range s.steps {
		step := &s.steps[i]

		// Verificar cancelamento do contexto
		if err := s.ctx.Err(); err != nil {
			s.logf("Context canceled during saga execution: %v", err)
			return s.handleExecutionFailure(step, i-1, fmt.Errorf("%w: %v", ErrSagaCanceled, err), startTime)
		}

		// Executar o passo com retry se configurado
		if err := s.executeStepWithRetry(step); err != nil {
			step.Status = StatusFailed
			s.logf("Step %s failed: %v", step.ID, err)
			return s.handleExecutionFailure(step, i-1, err, startTime)
		}

		// Sucesso na execução do passo
		step.Status = StatusExecuted
		s.result.ExecutedSteps = append(s.result.ExecutedSteps, step.ID)
		s.logf("Step %s executed successfully", step.ID)

		if s.config.OnStepSuccess != nil {
			s.config.OnStepSuccess(step.ID)
		}
	}

	// Todos os passos executados com sucesso
	s.result.Success = true
	s.result.Duration = time.Since(startTime)
	s.logf("Saga execution completed successfully in %v", s.result.Duration)

	if s.config.OnComplete != nil {
		s.config.OnComplete(*s.result)
	}

	return nil
}

// executeStepWithRetry executa um passo com tentativas conforme política
func (s *Saga) executeStepWithRetry(step *Step) error {
	var lastErr error

	// Se não há política de retry, executa apenas uma vez
	if step.RetryPolicy == nil {
		return step.Execute(s.ctx)
	}

	for attempt := 0; attempt <= step.RetryPolicy.MaxRetries; attempt++ {
		step.retryCount = attempt

		// Se não é a primeira tentativa, aguarda conforme backoff
		if attempt > 0 {
			step.Status = StatusRetrying
			s.logf("Retrying step %s (attempt %d/%d)", step.ID, attempt, step.RetryPolicy.MaxRetries)

			waitTime := step.RetryPolicy.BackoffFunc(attempt)
			select {
			case <-time.After(waitTime):
				// Continua após espera
			case <-s.ctx.Done():
				return fmt.Errorf("%w: %v", ErrSagaCanceled, s.ctx.Err())
			}
		}

		if err := step.Execute(s.ctx); err != nil {
			lastErr = err
			s.logf("Step %s execution failed: %v", step.ID, err)
			continue // Tenta novamente se ainda há tentativas restantes
		}

		return nil // Sucesso
	}

	return fmt.Errorf("step %s failed after %d attempts: %w",
		step.ID, step.retryCount+1, lastErr)
}

// handleExecutionFailure processa uma falha de execução e inicia compensação
func (s *Saga) handleExecutionFailure(
	failedStep *Step,
	lastSuccessIndex int,
	originalErr error,
	startTime time.Time,
) error {

	// Registra detalhes da falha no resultado
	s.result.Success = false
	s.result.FailedStepID = failedStep.ID
	s.result.OriginalError = originalErr
	s.result.Duration = time.Since(startTime)

	// Executa callback de falha se configurado
	if s.config.OnFailure != nil {
		s.config.OnFailure(failedStep.ID, originalErr)
	}

	// Inicia processo de compensação
	compErr := s.compensate(lastSuccessIndex, originalErr)
	if compErr != nil && !errors.Is(compErr, originalErr) {
		s.result.CompensationError = compErr
	}

	// Callback de conclusão
	if s.config.OnComplete != nil {
		s.config.OnComplete(*s.result)
	}

	return originalErr
}

// compensate executa as compensações em ordem reversa
func (s *Saga) compensate(lastExecutedIndex int, originalError error) error {
	if lastExecutedIndex < 0 {
		return originalError
	}

	s.logf("Starting compensation from step index %d", lastExecutedIndex)
	var compensationErrors []error

	// Compensa passos em ordem reversa
	for i := lastExecutedIndex; i >= 0; i-- {
		step := &s.steps[i]
		if step.Status == StatusExecuted {
			s.logf("Compensating step %s", step.ID)

			// Executa compensação
			if err := step.Compensate(s.ctx); err != nil {
				compensationErrors = append(
					compensationErrors,
					fmt.Errorf("compensation failed for step %s: %w", step.ID, err),
				)
				step.Status = StatusFailed
				s.logf("Compensation failed for step %s: %v", step.ID, err)
			} else {
				step.Status = StatusCompensated
				s.result.CompensatedSteps = append(s.result.CompensatedSteps, step.ID)
				s.logf("Step %s compensated successfully", step.ID)

				if s.config.OnStepCompensated != nil {
					s.config.OnStepCompensated(step.ID)
				}
			}
		}
	}

	// Se houve erros na compensação
	if len(compensationErrors) > 0 {
		return fmt.Errorf("%w: compensation errors: %v (original error: %v)",
			ErrSagaCompensation,
			errors.Join(compensationErrors...),
			originalError,
		)
	}

	return originalError
}

// logf registra mensagens de log se o logger estiver configurado
func (s *Saga) logf(format string, args ...any) {
	if s.config.Logger != nil {
		s.config.Logger.Printf(format, args...)
	}
}
