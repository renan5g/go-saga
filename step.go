package saga

import (
	"context"
	"time"
)

// StepID identifica unicamente cada passo na saga
type StepID string

// StepFunc é um tipo para funções executáveis em um passo
type StepFunc func(ctx context.Context) error

// StepStatus representa o estado atual de um passo
type StepStatus string

// Constantes para status dos passos
const (
	StatusPending     StepStatus = "pending"
	StatusExecuted    StepStatus = "executed"
	StatusFailed      StepStatus = "failed"
	StatusCompensated StepStatus = "compensated"
	StatusRetrying    StepStatus = "retrying"
	StatusSkipped     StepStatus = "skipped"
)

// NoOp é uma função step que não faz nada e sempre retorna nil
var NoOp = func(ctx context.Context) error { return nil }

// Step representa um passo transacional na saga
type Step struct {
	ID             StepID
	ExecuteFunc    StepFunc
	CompensateFunc StepFunc
	Status         StepStatus
	Description    string
	CreatedAt      time.Time
	RetryPolicy    *RetryPolicy   // Política de tentativas para este passo
	Metadata       map[string]any // Metadata para uso personalizado
	retryCount     int            // contador de tentativas atual
}

// NewStep cria uma nova instância de Step com valores padrão
func NewStep(id StepID, exec, comp StepFunc) *Step {
	return &Step{
		ID:             id,
		ExecuteFunc:    exec,
		CompensateFunc: comp,
		Status:         StatusPending,
		CreatedAt:      time.Now(),
		Metadata:       make(map[string]any),
	}
}

// GetID retorna o ID do passo
func (s *Step) GetID() StepID {
	return s.ID
}

// GetStatus retorna o status atual do passo
func (s *Step) GetStatus() StepStatus {
	return s.Status
}

// SetStatus define o status do passo
func (s *Step) SetStatus(status StepStatus) {
	s.Status = status
}

// Execute executa a lógica do passo
func (s *Step) Execute(ctx context.Context) error {
	return s.ExecuteFunc(ctx)
}

// Compensate executa a lógica de compensação do passo
func (s *Step) Compensate(ctx context.Context) error {
	return s.CompensateFunc(ctx)
}

// GetDescription retorna a descrição do passo
func (s *Step) GetDescription() string {
	return s.Description
}

// GetMetadata retorna os metadados do passo
func (s *Step) GetMetadata() map[string]any {
	return s.Metadata
}

// GetRetryPolicy retorna a política de retry do passo
func (s *Step) GetRetryPolicy() *RetryPolicy {
	return s.RetryPolicy
}

// GetRetryCount retorna o número atual de tentativas
func (s *Step) GetRetryCount() int {
	return s.retryCount
}

// IncrementRetryCount incrementa o contador de tentativas
func (s *Step) IncrementRetryCount() {
	s.retryCount++
}

// WithDescription adiciona uma descrição ao passo
func (s *Step) WithDescription(desc string) *Step {
	s.Description = desc
	return s
}

// WithRetryPolicy define uma política de retry para o passo
func (s *Step) WithRetryPolicy(policy *RetryPolicy) *Step {
	s.RetryPolicy = policy
	return s
}

// WithMetadata adiciona metadados ao passo
func (s *Step) WithMetadata(key string, value any) *Step {
	if s.Metadata == nil {
		s.Metadata = make(map[string]any)
	}
	s.Metadata[key] = value
	return s
}
