package saga

import (
	"context"
	"sync"
	"time"
)

type AsyncExecutionOptions struct {
	ProgressUpdates bool
	OnStepStart     func(stepID StepID)
	OnProgress      func(completed, total int)
	OnPartialError  func(step StepID, err error)
}

type AsyncExecutionOption func(*AsyncExecutionOptions)

func WithProgressUpdates(enabled bool) AsyncExecutionOption {
	return func(opts *AsyncExecutionOptions) {
		opts.ProgressUpdates = enabled
	}
}

func WithOnStepStart(callback func(stepID StepID)) AsyncExecutionOption {
	return func(opts *AsyncExecutionOptions) {
		opts.OnStepStart = callback
	}
}

func WithOnProgress(callback func(completed, total int)) AsyncExecutionOption {
	return func(opts *AsyncExecutionOptions) {
		opts.OnProgress = callback
	}
}

func WithOnPartialError(callback func(step StepID, err error)) AsyncExecutionOption {
	return func(opts *AsyncExecutionOptions) {
		opts.OnPartialError = callback
	}
}

type AsyncResult struct {
	Error           error
	ExecutionResult *ExecutionResult
	StartTime       time.Time
	EndTime         time.Time
	Canceled        bool
}

func (s *Saga) ExecuteWithProgress(options ...AsyncExecutionOption) (<-chan AsyncResult, context.CancelFunc) {
	opts := AsyncExecutionOptions{
		ProgressUpdates: true,
	}
	for _, opt := range options {
		opt(&opts)
	}

	ctx, cancel := context.WithCancel(s.ctx)
	resultCh := make(chan AsyncResult, 1)

	sagaWithCancel := &Saga{
		ctx:         ctx,
		steps:       s.steps,
		config:      s.config,
		hasExecuted: s.hasExecuted,
		result:      s.result,
	}

	if opts.ProgressUpdates {
		var progMu sync.Mutex
		completed := 0
		total := len(s.steps)

		origConfig := sagaWithCancel.config
		sagaWithCancel.config.OnStepSuccess = func(stepID StepID) {
			if origConfig.OnStepSuccess != nil {
				origConfig.OnStepSuccess(stepID)
			}

			if opts.OnProgress != nil {
				progMu.Lock()
				completed++
				opts.OnProgress(completed, total)
				progMu.Unlock()
			}
		}

		if opts.OnPartialError != nil {
			sagaWithCancel.config.OnFailure = func(stepID StepID, err error) {
				if origConfig.OnFailure != nil {
					origConfig.OnFailure(stepID, err)
				}
				if opts.OnPartialError != nil {
					opts.OnPartialError(stepID, err)
				}
			}
		}
	}

	go func() {
		startTime := time.Now()
		err := sagaWithCancel.Execute()
		endTime := time.Now()

		result := AsyncResult{
			Error:           err,
			ExecutionResult: sagaWithCancel.GetResult(),
			StartTime:       startTime,
			EndTime:         endTime,
			Canceled:        ctx.Err() != nil,
		}

		resultCh <- result
		close(resultCh)
	}()

	return resultCh, cancel
}

func (s *Saga) ExecuteAsync() <-chan error {
	resultCh := make(chan error, 1)
	go func() {
		err := s.Execute()
		resultCh <- err
		close(resultCh)
	}()
	return resultCh
}
