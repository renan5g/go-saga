package saga_test

import (
	"context"
	"errors"
	"testing"

	"github.com/renan5g/go-saga"
	"github.com/stretchr/testify/assert"
)

func TestSaga_Execute(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		// Arrange
		s := saga.NewSaga(context.Background())
		executed := false

		s.AddStep(saga.NewStep(
			"test-step",
			func(ctx context.Context) error {
				executed = true
				return nil
			},
			func(ctx context.Context) error { return nil },
		))

		// Act
		err := s.Execute()

		// Assert
		assert.NoError(t, err)
		assert.True(t, executed, "Step should have been executed")
		assert.True(t, s.IsExecuted(), "Saga should be marked as executed")
		assert.True(t, s.IsSuccessful(), "Saga should be marked as successful")
	})

	t.Run("execution with compensation", func(t *testing.T) {
		// Arrange
		s := saga.NewSaga(context.Background())
		step1Executed := false
		step1Compensated := false

		s.AddStep(saga.NewStep(
			"step-1",
			func(ctx context.Context) error {
				step1Executed = true
				return nil
			},
			func(ctx context.Context) error {
				step1Compensated = true
				return nil
			},
		))

		s.AddStep(saga.NewStep(
			"step-2",
			func(ctx context.Context) error {
				return errors.New("step 2 failed")
			},
			func(ctx context.Context) error {
				// Esta função não deve ser chamada porque step-2 falha antes de ser executado com sucesso
				return nil
			},
		))

		// Act
		err := s.Execute()
		result := s.GetResult()

		// Assert
		assert.Error(t, err)
		assert.True(t, step1Executed, "Step 1 should have been executed")
		assert.True(t, step1Compensated, "Step 1 should have been compensated")
		assert.Contains(t, result.ExecutedSteps, saga.StepID("step-1"))
		assert.Contains(t, result.CompensatedSteps, saga.StepID("step-1"))
		assert.Equal(t, saga.StepID("step-2"), result.FailedStepID)
	})
}
