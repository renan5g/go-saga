package saga

// SagaConfig define as configurações de uma saga
type SagaConfig struct {
	OnFailure          func(stepID StepID, err error)
	OnStepSuccess      func(stepID StepID)
	OnStepCompensated  func(stepID StepID)
	OnComplete         func(result ExecutionResult)
	Logger             Logger
	DefaultRetryPolicy *RetryPolicy
}

// Logger define a interface para logging
type Logger interface {
	Printf(format string, v ...any)
}

// SagaOption permite configurar a saga na criação
type SagaOption func(*Saga)

func DefaultSagaConfig() SagaConfig {
	return SagaConfig{Logger: NewStdLogger("[SAGA] ")}
}

// WithOnFailureHook configura um handler para falhas
func WithOnFailureHook(handler func(stepID StepID, err error)) SagaOption {
	return func(s *Saga) {
		s.config.OnFailure = handler
	}
}

// WithOnStepSuccessHook configura um handler para sucesso nos passos
func WithOnStepSuccessHook(handler func(stepID StepID)) SagaOption {
	return func(s *Saga) {
		s.config.OnStepSuccess = handler
	}
}

// WithOnStepCompensatedHook configura um handler para compensação de passos
func WithOnStepCompensatedHook(handler func(stepID StepID)) SagaOption {
	return func(s *Saga) {
		s.config.OnStepCompensated = handler
	}
}

// WithOnCompleteHook configura um handler para conclusão da saga
func WithOnCompleteHook(handler func(result ExecutionResult)) SagaOption {
	return func(s *Saga) {
		s.config.OnComplete = handler
	}
}

// WithLogger configura um logger personalizado
func WithLogger(logger Logger) SagaOption {
	return func(s *Saga) {
		s.config.Logger = logger
	}
}

// WithDefaultRetryPolicy configura uma política de tentativas padrão
func WithDefaultRetryPolicy(policy *RetryPolicy) SagaOption {
	return func(s *Saga) {
		s.config.DefaultRetryPolicy = policy
	}
}
