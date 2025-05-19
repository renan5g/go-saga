package saga_test

import (
	"context"
	"testing"
	"time"

	"github.com/renan5g/go-saga"
	"github.com/stretchr/testify/assert"
)

func TestSagaExecuteWithProgress(t *testing.T) {
	t.Run("success", func(t *testing.T) {

		assert := assert.New(t)
		ctx := context.Background()
		s := saga.NewSaga(ctx)

		step1Called := false
		s.AddStep(
			saga.NewStep(
				"step1",
				func(ctx context.Context) error {
					step1Called = true
					return nil
				},
				func(ctx context.Context) error {
					return nil
				},
			),
		)

		resultCh, _ := s.ExecuteWithProgress(
			saga.WithOnProgress(func(completed, total int) {
				assert.Equal(1, completed)
				assert.Equal(1, total)
			}),
		)
		result := <-resultCh

		assert.True(step1Called, "Expected step1 to be called")
		assert.False(result.Canceled, "Expected saga not to be canceled")
		assert.Nil(result.Error, "Expected no error")
	})

	t.Run("cancellation", func(t *testing.T) {
		assert := assert.New(t)
		ctx := context.Background()
		s := saga.NewSaga(ctx)

		s.AddStep(
			saga.NewStep(
				"long-step",
				func(ctx context.Context) error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(2 * time.Second):
						return nil
					}
				},
				func(ctx context.Context) error {
					return nil
				},
			),
		)

		resultCh, cancel := s.ExecuteWithProgress()

		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		result := <-resultCh

		assert.True(result.Canceled, "Expected saga to be canceled")
	})
}
