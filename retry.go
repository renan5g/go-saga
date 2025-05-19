package saga

import (
	"math/rand"
	"time"
)

// RetryPolicy define a política de tentativas para execução de um passo
type RetryPolicy struct {
	MaxRetries  int                             // Número máximo de tentativas
	BackoffFunc func(attempt int) time.Duration // Função para calcular o tempo de espera entre tentativas
}

// NewRetryPolicy cria uma nova política de retry com backoff padrão
func NewRetryPolicy(maxRetries int) *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:  maxRetries,
		BackoffFunc: DefaultBackoff,
	}
}

// WithBackoffFunc personaliza a função de backoff
func (r *RetryPolicy) WithBackoffFunc(fn func(attempt int) time.Duration) *RetryPolicy {
	r.BackoffFunc = fn
	return r
}

// DefaultBackoff retorna um tempo de espera exponencial com jitter
func DefaultBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	// Exponential backoff com jitter (100ms * 2^attempt ± 20%)
	baseDelay := time.Duration(100*(1<<attempt)) * time.Millisecond
	jitter := time.Duration(float64(baseDelay) * 0.2 * (0.5 - float64(time.Now().Nanosecond())/1e9))
	return baseDelay + jitter
}

// LinearBackoff retorna um tempo de espera linear
func LinearBackoff(baseDelay time.Duration) func(attempt int) time.Duration {
	return func(attempt int) time.Duration {
		return baseDelay * time.Duration(attempt)
	}
}

// FixedBackoff retorna sempre o mesmo tempo de espera
func FixedBackoff(delay time.Duration) func(attempt int) time.Duration {
	return func(_ int) time.Duration {
		return delay
	}
}

func ExponentialBackoff(baseDelay time.Duration) func(attempt int) time.Duration {
	return func(attempt int) time.Duration {
		return baseDelay * time.Duration(1<<attempt)
	}
}

func JitterBackoff(baseDelay time.Duration) func(attempt int) time.Duration {
	return func(attempt int) time.Duration {
		jitter := time.Duration(rand.Int63n(int64(baseDelay)))
		return baseDelay + jitter
	}
}
