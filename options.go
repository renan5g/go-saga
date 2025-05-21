package saga

type SagaConfig struct {
	OnFailure          func(stepID StepID, err error)
	OnStepSuccess      func(stepID StepID)
	OnStepCompensated  func(stepID StepID)
	OnComplete         func(result ExecutionResult)
	Logger             Logger
	ErrorLogger        Logger
	DefaultRetryPolicy *RetryPolicy
}

type Logger interface {
	Printf(format string, v ...any)
}

type SagaOption func(*Saga)

func DefaultSagaConfig() SagaConfig {
	return SagaConfig{}
}

func WithOnFailureHook(handler func(stepID StepID, err error)) SagaOption {
	return func(s *Saga) {
		s.config.OnFailure = handler
	}
}

func WithOnStepSuccessHook(handler func(stepID StepID)) SagaOption {
	return func(s *Saga) {
		s.config.OnStepSuccess = handler
	}
}

func WithOnStepCompensatedHook(handler func(stepID StepID)) SagaOption {
	return func(s *Saga) {
		s.config.OnStepCompensated = handler
	}
}

func WithOnCompleteHook(handler func(result ExecutionResult)) SagaOption {
	return func(s *Saga) {
		s.config.OnComplete = handler
	}
}

func WithLogger(logger Logger) SagaOption {
	return func(s *Saga) {
		s.config.ErrorLogger = logger
	}
}

func WithErrorLogger(logger Logger) SagaOption {
	return func(s *Saga) {
		s.config.ErrorLogger = logger
	}
}

func WithDefaultRetryPolicy(policy *RetryPolicy) SagaOption {
	return func(s *Saga) {
		s.config.DefaultRetryPolicy = policy
	}
}
