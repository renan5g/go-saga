package saga

import (
	"context"
)

// IStep define a interface para os passos da saga
type IStep interface {
	// GetID retorna o identificador único do passo
	GetID() StepID

	// GetStatus retorna o status atual do passo
	GetStatus() StepStatus

	// SetStatus define o status do passo
	SetStatus(status StepStatus)

	// Execute executa a lógica do passo
	Execute(ctx context.Context) error

	// Compensate executa a lógica de compensação do passo
	Compensate(ctx context.Context) error

	// GetDescription retorna a descrição do passo
	GetDescription() string

	// GetMetadata retorna os metadados do passo
	GetMetadata() map[string]any

	// GetRetryPolicy retorna a política de retry do passo
	GetRetryPolicy() *RetryPolicy

	// GetRetryCount retorna o número atual de tentativas
	GetRetryCount() int

	// IncrementRetryCount incrementa o contador de tentativas
	IncrementRetryCount()
}

// ISaga define a interface para implementações de saga
type ISaga interface {
	// AddStep adiciona um novo passo à saga
	AddStep(step IStep) ISaga

	// Execute executa todos os passos da saga
	Execute() error

	// GetStepStatus retorna o status atual de um passo
	GetStepStatus(id StepID) (StepStatus, error)

	// GetSteps retorna uma cópia dos passos atuais
	GetSteps() []IStep

	// GetStepByID retorna um passo específico pelo ID
	GetStepByID(id StepID) (IStep, error)

	// GetResult retorna o resultado da execução da saga
	GetResult() *ExecutionResult

	// IsExecuted verifica se a saga já foi executada
	IsExecuted() bool

	// IsSuccessful verifica se a saga foi executada com sucesso
	IsSuccessful() bool
}

// IStepGroup define uma interface para grupos de passos
type IStepGroup interface {
	IStep

	// AddStep adiciona um passo ao grupo
	AddStep(step IStep) IStepGroup

	// GetSteps retorna os passos do grupo
	GetSteps() []IStep

	// SetExecutionMode define o modo de execução dos passos
	SetExecutionMode(mode ExecutionMode) IStepGroup

	// GetExecutionMode retorna o modo de execução atual
	GetExecutionMode() ExecutionMode
}

// ExecutionMode define o modo de execução de um grupo de passos
type ExecutionMode int

const (
	// Sequential executa os passos sequencialmente
	Sequential ExecutionMode = iota

	// Parallel executa os passos em paralelo
	Parallel
)
