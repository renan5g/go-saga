package saga

import (
	"context"
	"sync"
	"time"
)

// AsyncExecutionOptions configura o comportamento da execução assíncrona
type AsyncExecutionOptions struct {
	ProgressUpdates bool                         // Habilita atualizações de progresso
	OnStepStart     func(stepID StepID)          // Callback quando um passo inicia
	OnProgress      func(completed, total int)   // Callback para atualização de progresso
	OnPartialError  func(err error, step StepID) // Callback para erro em passo específico
}

type AsyncExecutionOption func(*AsyncExecutionOptions)

// WithProgressUpdates habilita atualizações de progresso
func WithProgressUpdates(enabled bool) AsyncExecutionOption {
	return func(opts *AsyncExecutionOptions) {
		opts.ProgressUpdates = enabled
	}
}

// WithOnStepStart define um callback para o início de um passo
func WithOnStepStart(callback func(stepID StepID)) AsyncExecutionOption {
	return func(opts *AsyncExecutionOptions) {
		opts.OnStepStart = callback
	}
}

// WithOnProgress define um callback para atualização de progresso
func WithOnProgress(callback func(completed, total int)) AsyncExecutionOption {
	return func(opts *AsyncExecutionOptions) {
		opts.OnProgress = callback
	}
}

// WithOnPartialError define um callback para erro em passo específico
func WithOnPartialError(callback func(err error, step StepID)) AsyncExecutionOption {
	return func(opts *AsyncExecutionOptions) {
		opts.OnPartialError = callback
	}
}

// AsyncResult contém o resultado completo de uma execução assíncrona
type AsyncResult struct {
	Error           error
	ExecutionResult *ExecutionResult
	StartTime       time.Time
	EndTime         time.Time
	Canceled        bool
}

// ExecuteWithProgress executa a saga de forma assíncrona com suporte a monitoramento
func (s *Saga) ExecuteWithProgress(options ...AsyncExecutionOption) (<-chan AsyncResult, context.CancelFunc) {
	opts := AsyncExecutionOptions{
		ProgressUpdates: true,
	}
	for _, opt := range options {
		opt(&opts)
	}

	// Criar contexto derivado que pode ser cancelado
	ctx, cancel := context.WithCancel(s.ctx)

	// Canal para resultado final
	resultCh := make(chan AsyncResult, 1)

	// Criar nova saga com o contexto cancelável
	sagaWithCancel := &Saga{
		ctx:         ctx,
		steps:       s.steps,
		config:      s.config,
		hasExecuted: s.hasExecuted,
		result:      s.result,
	}

	// Adicionar hooks para monitoramento se solicitado
	if opts.ProgressUpdates {
		// Mutex para proteger os contadores
		var progMu sync.Mutex
		completed := 0
		total := len(s.steps)

		// Hook para início de passo
		origConfig := sagaWithCancel.config
		sagaWithCancel.config.OnStepSuccess = func(stepID StepID) {
			// Chamar o hook original se existir
			if origConfig.OnStepSuccess != nil {
				origConfig.OnStepSuccess(stepID)
			}

			// Atualizar progresso
			if opts.OnProgress != nil {
				progMu.Lock()
				completed++
				opts.OnProgress(completed, total)
				progMu.Unlock()
			}
		}

		// Hook para falhas parciais
		if opts.OnPartialError != nil {
			sagaWithCancel.config.OnFailure = func(stepID StepID, err error) {
				// Chamar o hook original se existir
				if origConfig.OnFailure != nil {
					origConfig.OnFailure(stepID, err)
				}

				opts.OnPartialError(err, stepID)
			}
		}
	}

	// Executar saga em goroutine separada
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

// ExecuteAsync executa a saga de forma assíncrona
func (s *Saga) ExecuteAsync() <-chan error {
	resultCh := make(chan error, 1)
	go func() {
		err := s.Execute()
		resultCh <- err
		close(resultCh)
	}()
	return resultCh
}
