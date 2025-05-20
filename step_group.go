package saga

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

type StepGroupOption func(*StepGroup)

// StepGroup representa um grupo de passos que podem ser executados
// sequencialmente ou em paralelo como um único passo
type StepGroup struct {
	Step                   // StepGroup herda de Step
	steps    []IStep       // Passos contidos no grupo
	execMode ExecutionMode // Modo de execução (Sequential ou Parallel)
	mu       sync.RWMutex  // Mutex para acesso concorrente
	logger   Logger        // Logger para mensagens
}

func WithStepGroupLogger(logger Logger) StepGroupOption {
	return func(sg *StepGroup) {
		sg.logger = logger
	}
}

// NewStepGroup cria um novo grupo de passos
func NewStepGroup(id StepID, opts ...StepGroupOption) *StepGroup {
	sg := &StepGroup{
		Step: Step{
			ID:        id,
			Status:    StatusPending,
			CreatedAt: time.Now(),
			Metadata:  make(map[string]any),
		},
		steps:    make([]IStep, 0),
		execMode: Sequential,
		logger:   NewNoOpLogger(),
	}

	for _, opt := range opts {
		opt(sg)
	}

	// Define as funções execute e compensate para o grupo
	sg.ExecuteFunc = sg.executeGroup
	sg.CompensateFunc = sg.compensateGroup

	return sg
}

func (sg *StepGroup) Parallel() *StepGroup {
	sg.execMode = Parallel
	return sg
}

// AddStep adiciona um passo ao grupo
func (sg *StepGroup) AddStep(step IStep) IStepGroup {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	sg.steps = append(sg.steps, step)
	sg.logf("Added step %s to group %s", step.GetID(), sg.ID)
	return sg
}

// GetSteps retorna uma cópia dos passos do grupo
func (sg *StepGroup) GetSteps() []IStep {
	sg.mu.RLock()
	defer sg.mu.RUnlock()

	stepsCopy := make([]IStep, len(sg.steps))
	copy(stepsCopy, sg.steps)
	return stepsCopy
}

// SetExecutionMode altera o modo de execução do grupo
func (sg *StepGroup) SetExecutionMode(mode ExecutionMode) IStepGroup {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	sg.execMode = mode
	return sg
}

// GetExecutionMode retorna o modo de execução atual
func (sg *StepGroup) GetExecutionMode() ExecutionMode {
	sg.mu.RLock()
	defer sg.mu.RUnlock()

	return sg.execMode
}

// executeGroup é a implementação da execução do grupo
func (sg *StepGroup) executeGroup(ctx context.Context) error {
	sg.mu.RLock()
	if len(sg.steps) == 0 {
		sg.mu.RUnlock()
		sg.logf("StepGroup %s has no steps to execute", sg.ID)
		return nil
	}
	execMode := sg.execMode
	steps := slices.Clone(sg.steps) // Create a copy of steps
	sg.mu.RUnlock()
	sg.logf("Executing StepGroup %s with %d steps in %s mode", sg.ID, len(steps), execMode)

	// Executa os passos de acordo com o modo configurado
	if execMode == Parallel {
		return sg.executeParallel(ctx, steps)
	}
	return sg.executeSequential(ctx, steps)
}

// executeSequential executa os passos em sequência
func (sg *StepGroup) executeSequential(ctx context.Context, steps []IStep) error {
	for i, step := range steps {
		// Verifica cancelamento do contexto
		if err := ctx.Err(); err != nil {
			sg.logf("Context canceled during step group execution: %v", err)
			return fmt.Errorf("step group %s canceled: %w", sg.ID, err)
		}

		sg.logf("Executing step %s (%d/%d) of group %s", step.GetID(), i+1, len(sg.steps), sg.ID)

		if err := step.Execute(ctx); err != nil {
			step.SetStatus(StatusFailed)
			sg.logf("Step %s of group %s failed: %v", step.GetID(), sg.ID, err)
			return fmt.Errorf("step %s failed in group %s: %w", step.GetID(), sg.ID, err)
		}

		step.SetStatus(StatusExecuted)
		sg.logf("Step %s of group %s executed successfully", step.GetID(), sg.ID)
	}

	sg.logf("StepGroup %s executed all steps successfully", sg.ID)
	return nil
}

// executeParallel executa os passos em paralelo
func (sg *StepGroup) executeParallel(ctx context.Context, steps []IStep) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(steps))

	// Inicia cada passo em uma goroutine separada
	for _, step := range sg.steps {
		wg.Add(1)
		go func(s IStep) {
			defer wg.Done()

			sg.logf("Executing step %s of group %s in parallel",
				s.GetID(), sg.ID)

			if err := s.Execute(ctx); err != nil {
				s.SetStatus(StatusFailed)
				sg.logf("Step %s of group %s failed: %v",
					s.GetID(), sg.ID, err)
				errCh <- fmt.Errorf("step %s failed in group %s: %w",
					s.GetID(), sg.ID, err)
				return
			}

			s.SetStatus(StatusExecuted)
			sg.logf("Step %s of group %s executed successfully",
				s.GetID(), sg.ID)
		}(step)
	}

	// Canal para sinalizar quando todas as goroutines terminarem
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Aguarda conclusão ou erro
	select {
	case <-ctx.Done():
		return fmt.Errorf("step group %s canceled: %w", sg.ID, ctx.Err())
	case err := <-errCh:
		return err
	case <-done:
		sg.logf("StepGroup %s executed all steps in parallel successfully", sg.ID)
		return nil
	}
}

// compensateGroup é a implementação da compensação do grupo
func (sg *StepGroup) compensateGroup(ctx context.Context) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	sg.logf("Compensating StepGroup %s", sg.ID)

	if len(sg.steps) == 0 {
		return nil
	}

	var errs error

	// Compensa os passos em ordem reversa
	for i := len(sg.steps) - 1; i >= 0; i-- {
		step := sg.steps[i]

		// Só compensa passos executados com sucesso
		if step.GetStatus() == StatusExecuted {
			sg.logf("Compensating step %s of group %s", step.GetID(), sg.ID)

			if err := step.Compensate(ctx); err != nil {
				step.SetStatus(StatusFailed)
				sg.logf("Compensation failed for step %s: %v", step.GetID(), err)
				errs = errors.Join(errs, fmt.Errorf("compensation failed for step %s: %w", step.GetID(), err))
			} else {
				step.SetStatus(StatusCompensated)
				sg.logf("Step %s compensated successfully", step.GetID())
			}
		}
	}

	// Se houve erros na compensação
	if errs != nil {
		return fmt.Errorf("step group %s compensation had multiple errors: %w",
			sg.ID, errs)
	}

	return nil
}

// logf registra mensagens de log se o logger estiver configurado
func (sg *StepGroup) logf(format string, args ...any) {
	if sg.logger != nil {
		sg.logger.Printf(format, args...)
	}
}
