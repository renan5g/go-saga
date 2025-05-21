package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Saga struct {
	steps       []IStep
	ctx         context.Context
	mu          sync.RWMutex
	config      SagaConfig
	hasExecuted bool
	result      *ExecutionResult
}

func NewSaga(ctx context.Context, options ...SagaOption) *Saga {
	saga := &Saga{
		ctx:    ctx,
		steps:  make([]IStep, 0),
		config: DefaultSagaConfig(),
		result: NewExecutionResult(),
	}
	for _, option := range options {
		option(saga)
	}
	return saga
}

func (s *Saga) AddStep(step IStep) ISaga {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.steps = append(s.steps, step)
	s.infof("Added step: %s", step.GetID())
	return s
}

func (s *Saga) GetStepStatus(id StepID) (StepStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, step := range s.steps {
		if step.GetID() == id {
			return step.GetStatus(), nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrStepNotFound, id)
}

func (s *Saga) GetSteps() []IStep {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stepsCopy := make([]IStep, len(s.steps))
	copy(stepsCopy, s.steps)
	return stepsCopy
}

func (s *Saga) GetStepByID(id StepID) (IStep, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, step := range s.steps {
		if step.GetID() == id {
			return step, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrStepNotFound, id)
}

func (s *Saga) GetResult() *ExecutionResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.result == nil {
		return NewExecutionResult()
	}
	resultCopy := *s.result
	return &resultCopy
}

func (s *Saga) IsExecuted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hasExecuted
}

func (s *Saga) IsSuccessful() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hasExecuted && s.result != nil && s.result.Success
}

func (s *Saga) Execute() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime := time.Now()

	if s.hasExecuted {
		return ErrSagaAlreadyExecuted
	}

	s.hasExecuted = true
	s.infof("Starting saga execution with %d steps", len(s.steps))

	for i, step := range s.steps {
		if err := s.ctx.Err(); err != nil {
			s.errorf("Context canceled during saga execution: %v", err)
			return s.handleExecutionFailure(step, i-1, fmt.Errorf("%w: %v", ErrSagaCanceled, err), startTime)
		}

		if err := s.executeStepWithRetry(step); err != nil {
			step.SetStatus(StatusFailed)
			s.errorf("Step %s failed: %v", step.GetID(), err)
			return s.handleExecutionFailure(step, i-1, err, startTime)
		}

		step.SetStatus(StatusExecuted)
		s.result.ExecutedSteps = append(s.result.ExecutedSteps, step.GetID())
		s.infof("Step %s executed successfully", step.GetID())

		if s.config.OnStepSuccess != nil {
			s.config.OnStepSuccess(step.GetID())
		}
	}

	s.result.Success = true
	s.result.Duration = time.Since(startTime)
	s.infof("Saga execution completed successfully in %v", s.result.Duration)

	if s.config.OnComplete != nil {
		s.config.OnComplete(*s.result)
	}

	return nil
}

func (s *Saga) executeStepWithRetry(step IStep) error {
	var lastErr error
	retryPolicy := step.GetRetryPolicy()

	if retryPolicy == nil {
		return step.Execute(s.ctx)
	}

	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			step.IncrementRetryCount()
		}

		if attempt > 0 {
			step.SetStatus(StatusRetrying)
			s.infof("Retrying step %s (attempt %d/%d)", step.GetID(), attempt, retryPolicy.MaxRetries)

			waitTime := retryPolicy.BackoffFunc(attempt)
			select {
			case <-time.After(waitTime):
			case <-s.ctx.Done():
				return fmt.Errorf("%w: %v", ErrSagaCanceled, s.ctx.Err())
			}
		}

		if err := step.Execute(s.ctx); err != nil {
			lastErr = err
			s.errorf("Step %s execution failed: %v", step.GetID(), err)
			continue
		}

		return nil
	}

	return fmt.Errorf("step %s failed after %d attempts: %w", step.GetID(), step.GetRetryCount()+1, lastErr)
}

func (s *Saga) handleExecutionFailure(
	failedStep IStep,
	lastSuccessIndex int,
	originalErr error,
	startTime time.Time,
) error {

	s.result.Success = false
	s.result.FailedStepID = failedStep.GetID()
	s.result.OriginalError = originalErr
	s.result.Duration = time.Since(startTime)

	if s.config.OnFailure != nil {
		s.config.OnFailure(failedStep.GetID(), originalErr)
	}

	compErr := s.compensate(lastSuccessIndex, originalErr)
	if compErr != nil && !errors.Is(compErr, originalErr) {
		s.result.CompensationError = compErr
	}

	if s.config.OnComplete != nil {
		s.config.OnComplete(*s.result)
	}

	return originalErr
}

func (s *Saga) compensate(lastExecutedIndex int, originalError error) error {
	if lastExecutedIndex < 0 {
		return originalError
	}

	s.infof("Starting compensation from step index %d", lastExecutedIndex)
	var compensationErrors []error

	for i := lastExecutedIndex; i >= 0; i-- {
		step := s.steps[i]
		if step.GetStatus() == StatusExecuted {
			s.infof("Compensating step %s", step.GetID())

			if err := step.Compensate(s.ctx); err != nil {
				compensationErrors = append(
					compensationErrors,
					fmt.Errorf("compensation failed for step %s: %w", step.GetID(), err),
				)
				step.SetStatus(StatusFailed)
				s.errorf("Compensation failed for step %s: %v", step.GetID(), err)
			} else {
				step.SetStatus(StatusCompensated)
				s.result.CompensatedSteps = append(s.result.CompensatedSteps, step.GetID())
				s.infof("Step %s compensated successfully", step.GetID())

				if s.config.OnStepCompensated != nil {
					s.config.OnStepCompensated(step.GetID())
				}
			}
		}
	}

	if len(compensationErrors) > 0 {
		return fmt.Errorf("%w: compensation errors: %v (original error: %v)",
			ErrSagaCompensation,
			errors.Join(compensationErrors...),
			originalError,
		)
	}

	return originalError
}

func (s *Saga) infof(format string, args ...any) {
	if s.config.Logger != nil {
		s.config.Logger.Printf(format, args...)
	}
}

func (s *Saga) errorf(format string, args ...any) {
	if s.config.ErrorLogger != nil {
		s.config.ErrorLogger.Printf(format, args...)
	}
}
