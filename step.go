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
	ID          StepID
	Execute     StepFunc
	Compensate  StepFunc
	Status      StepStatus
	Description string
	CreatedAt   time.Time
	RetryPolicy *RetryPolicy   // Política de tentativas para este passo
	Metadata    map[string]any // Metadata para uso personalizado
	retryCount  int            // contador de tentativas atual
}

// NewStep cria uma nova instância de Step com valores padrão
func NewStep(id StepID, exec, comp StepFunc) Step {
	return Step{
		ID:         id,
		Execute:    exec,
		Compensate: comp,
		Status:     StatusPending,
		CreatedAt:  time.Now(),
		Metadata:   make(map[string]any),
	}
}

// WithDescription adiciona uma descrição ao passo
func (s Step) WithDescription(desc string) Step {
	s.Description = desc
	return s
}

// WithRetryPolicy define uma política de retry para o passo
func (s Step) WithRetryPolicy(policy *RetryPolicy) Step {
	s.RetryPolicy = policy
	return s
}

// WithMetadata adiciona metadados ao passo
func (s Step) WithMetadata(key string, value any) Step {
	if s.Metadata == nil {
		s.Metadata = make(map[string]any)
	}
	s.Metadata[key] = value
	return s
}
