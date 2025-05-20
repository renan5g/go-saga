package saga

import (
	"math"
	"time"
)

// RetryPolicy define a política de novas tentativas para passos
type RetryPolicy struct {
	MaxRetries  int                             // Número máximo de tentativas
	BackoffFunc func(attempt int) time.Duration // Função para calcular tempo de espera
}

// DefaultBackoff retorna uma função de backoff exponencial (2^tentativa * base)
func DefaultBackoff(baseDelay time.Duration) func(attempt int) time.Duration {
	return func(attempt int) time.Duration {
		// 2^tentativa * baseDelay com limite para evitar overflow
		multiplier := math.Min(math.Pow(2, float64(attempt)), 10)
		return time.Duration(float64(baseDelay) * multiplier)
	}
}

// LinearBackoff retorna uma função de backoff linear (tentativa * base)
func LinearBackoff(baseDelay time.Duration) func(attempt int) time.Duration {
	return func(attempt int) time.Duration {
		return baseDelay * time.Duration(attempt)
	}
}

// FixedBackoff retorna uma função de backoff com delay fixo
func FixedBackoff(delay time.Duration) func(attempt int) time.Duration {
	return func(attempt int) time.Duration {
		return delay
	}
}

// NewRetryPolicy cria uma política de retry com valores padrão
func NewRetryPolicy(maxRetries int, baseDelay time.Duration) *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:  maxRetries,
		BackoffFunc: DefaultBackoff(baseDelay),
	}
}

// WithLinearBackoff configura backoff linear para esta política
func (r *RetryPolicy) WithLinearBackoff(baseDelay time.Duration) *RetryPolicy {
	r.BackoffFunc = LinearBackoff(baseDelay)
	return r
}

// WithFixedBackoff configura backoff fixo para esta política
func (r *RetryPolicy) WithFixedBackoff(delay time.Duration) *RetryPolicy {
	r.BackoffFunc = FixedBackoff(delay)
	return r
}

// WithCustomBackoff configura uma função de backoff personalizada
func (r *RetryPolicy) WithCustomBackoff(backoffFunc func(attempt int) time.Duration) *RetryPolicy {
	r.BackoffFunc = backoffFunc
	return r
}
