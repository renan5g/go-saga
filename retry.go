package saga

import (
	"math"
	"time"
)

// RetryPolicy defines the retry policy for a saga step
type RetryPolicy struct {
	MaxRetries  int                             // Maximum number of retries
	BackoffFunc func(attempt int) time.Duration // Backoff function used to calculate the delay between retries
}

func DefaultBackoff(baseDelay time.Duration) func(attempt int) time.Duration {
	return func(attempt int) time.Duration {
		multiplier := math.Min(math.Pow(2, float64(attempt)), 10)
		return time.Duration(float64(baseDelay) * multiplier)
	}
}

func LinearBackoff(baseDelay time.Duration) func(attempt int) time.Duration {
	return func(attempt int) time.Duration {
		return baseDelay * time.Duration(attempt)
	}
}

func FixedBackoff(delay time.Duration) func(attempt int) time.Duration {
	return func(attempt int) time.Duration {
		return delay
	}
}

func NewRetryPolicy(maxRetries int, baseDelay time.Duration) *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:  maxRetries,
		BackoffFunc: DefaultBackoff(baseDelay),
	}
}

func (r *RetryPolicy) WithLinearBackoff(baseDelay time.Duration) *RetryPolicy {
	r.BackoffFunc = LinearBackoff(baseDelay)
	return r
}

func (r *RetryPolicy) WithFixedBackoff(delay time.Duration) *RetryPolicy {
	r.BackoffFunc = FixedBackoff(delay)
	return r
}

func (r *RetryPolicy) WithCustomBackoff(backoffFunc func(attempt int) time.Duration) *RetryPolicy {
	r.BackoffFunc = backoffFunc
	return r
}
