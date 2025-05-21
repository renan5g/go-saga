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

type StepGroup struct {
	Step
	steps       []IStep
	execMode    ExecutionMode
	mu          sync.RWMutex
	logger      Logger
	errorLogger Logger
}

func NewStepGroup(id StepID) *StepGroup {
	sg := &StepGroup{
		Step: Step{
			ID:        id,
			Status:    StatusPending,
			CreatedAt: time.Now(),
			Metadata:  make(map[string]any),
		},
		steps:    make([]IStep, 0),
		execMode: Sequential,
	}

	sg.ExecuteFunc = sg.executeGroup
	sg.CompensateFunc = sg.compensateGroup

	return sg
}

func (sg *StepGroup) Parallel() *StepGroup {
	sg.execMode = Parallel
	return sg
}

func (sg *StepGroup) AddStep(step IStep) IStepGroup {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	sg.steps = append(sg.steps, step)
	sg.infof("Added step %s to group %s", step.GetID(), sg.ID)
	return sg
}

func (sg *StepGroup) GetSteps() []IStep {
	sg.mu.RLock()
	defer sg.mu.RUnlock()

	stepsCopy := make([]IStep, len(sg.steps))
	copy(stepsCopy, sg.steps)
	return stepsCopy
}

func (sg *StepGroup) SetExecutionMode(mode ExecutionMode) IStepGroup {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	sg.execMode = mode
	return sg
}

func (sg *StepGroup) SetLogger(logger Logger) IStepGroup {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	sg.logger = logger
	return sg
}

func (sg *StepGroup) SetErrorLogger(logger Logger) IStepGroup {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	sg.errorLogger = logger
	return sg
}

func (sg *StepGroup) GetExecutionMode() ExecutionMode {
	sg.mu.RLock()
	defer sg.mu.RUnlock()

	return sg.execMode
}

func (sg *StepGroup) executeGroup(ctx context.Context) error {
	sg.mu.RLock()
	if len(sg.steps) == 0 {
		sg.mu.RUnlock()
		sg.infof("StepGroup %s has no steps to execute", sg.ID)
		return nil
	}
	execMode := sg.execMode
	steps := slices.Clone(sg.steps)
	sg.mu.RUnlock()
	sg.infof("Executing StepGroup %s with %d steps in %s mode", sg.ID, len(steps), execMode)
	if execMode == Parallel {
		return sg.executeParallel(ctx, steps)
	}
	return sg.executeSequential(ctx, steps)
}

func (sg *StepGroup) executeSequential(ctx context.Context, steps []IStep) error {
	for i, step := range steps {
		if err := ctx.Err(); err != nil {
			sg.errorf("Context canceled during step group execution: %v", err)
			return fmt.Errorf("step group %s canceled: %w", sg.ID, err)
		}
		sg.infof("Executing step %s (%d/%d) of group %s", step.GetID(), i+1, len(sg.steps), sg.ID)
		if err := step.Execute(ctx); err != nil {
			step.SetStatus(StatusFailed)
			sg.errorf("Step %s of group %s failed: %v", step.GetID(), sg.ID, err)
			return fmt.Errorf("step %s failed in group %s: %w", step.GetID(), sg.ID, err)
		}
		step.SetStatus(StatusExecuted)
		sg.infof("Step %s of group %s executed successfully", step.GetID(), sg.ID)
	}
	sg.infof("StepGroup %s executed all steps successfully", sg.ID)
	return nil
}

func (sg *StepGroup) executeParallel(ctx context.Context, steps []IStep) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(steps))
	for _, step := range sg.steps {
		wg.Add(1)
		go func(s IStep) {
			defer wg.Done()
			sg.infof("Executing step %s of group %s in parallel", s.GetID(), sg.ID)
			if err := s.Execute(ctx); err != nil {
				s.SetStatus(StatusFailed)
				sg.errorf("Step %s of group %s failed: %v", s.GetID(), sg.ID, err)
				errCh <- fmt.Errorf("step %s failed in group %s: %w", s.GetID(), sg.ID, err)
				return
			}
			s.SetStatus(StatusExecuted)
			sg.infof("Step %s of group %s executed successfully", s.GetID(), sg.ID)
		}(step)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("step group %s canceled: %w", sg.ID, ctx.Err())
	case err := <-errCh:
		return err
	case <-done:
		sg.infof("StepGroup %s executed all steps in parallel successfully", sg.ID)
		return nil
	}
}

func (sg *StepGroup) compensateGroup(ctx context.Context) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	sg.infof("Compensating StepGroup %s", sg.ID)
	if len(sg.steps) == 0 {
		return nil
	}
	var errs error

	for i := len(sg.steps) - 1; i >= 0; i-- {
		step := sg.steps[i]
		if step.GetStatus() == StatusExecuted {
			sg.infof("Compensating step %s of group %s", step.GetID(), sg.ID)

			if err := step.Compensate(ctx); err != nil {
				step.SetStatus(StatusFailed)
				sg.errorf("Compensation failed for step %s: %v", step.GetID(), err)
				errs = errors.Join(errs, fmt.Errorf("compensation failed for step %s: %w", step.GetID(), err))
			} else {
				step.SetStatus(StatusCompensated)
				sg.infof("Step %s compensated successfully", step.GetID())
			}
		}
	}

	if errs != nil {
		return fmt.Errorf("step group %s compensation had multiple errors: %w", sg.ID, errs)
	}

	return nil
}

func (s *StepGroup) infof(format string, args ...any) {
	if s.logger != nil {
		s.logger.Printf(format, args...)
	}
}

func (s *StepGroup) errorf(format string, args ...any) {
	if s.errorLogger != nil {
		s.errorLogger.Printf(format, args...)
	}
}
