# Saga Package

The `saga` package is a Go implementation of the Saga pattern for managing distributed transactions. It provides a robust framework for orchestrating a sequence of steps, each with execution and compensation logic, ensuring reliable transaction management with support for retries, parallel execution, and asynchronous processing. This package is ideal for microservices architectures where distributed transactions require coordination and fault tolerance.

## Features

- **Saga Orchestration**: Define and execute a series of steps with automatic compensation on failure.
- **Step Groups**: Group steps for sequential or parallel execution, enhancing flexibility.
- **Retry Policies**: Configurable retry mechanisms with exponential, linear, or fixed backoff strategies.
- **Asynchronous Execution**: Run sagas asynchronously with progress tracking and cancellation support.
- **Logging**: Customizable logging with support for stdout, file, or no-op loggers.
- **Metadata and Status Tracking**: Attach metadata to steps and track their status (e.g., pending, executed, failed, compensated).
- **Context Support**: Integrate with Go’s `context` package for cancellation and timeouts.
- **Event Callbacks**: Configurable hooks for step success, failure, compensation, and saga completion.

## Installation

To use this package, ensure you have Go installed, then include it in your project:

```bash
go get github.com/renan5g/go-saga
```

No external dependencies are required beyond the Go standard library.

## Usage

### Creating a Saga

A `Saga` is created with a context and optional configuration. Steps are added to define the transaction sequence, each with an execution and compensation function.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/renan5g/go-saga"
)

func main() {
	ctx := context.Background()
	s := saga.NewSaga(ctx, saga.WithLogger(saga.NewStdLogger("[Saga]: ")))

	// Define a step
	step1 := saga.
    NewStep(
      "step1",
      func(ctx context.Context) error {
        fmt.Println("Executing step 1")
        return nil
      },
      func(ctx context.Context) error {
        fmt.Println("Compensating step 1")
        return nil
      },
    ).
    WithDescription("First transaction step").
    WithRetryPolicy(saga.NewRetryPolicy(3, time.Second))

	// Add step to saga
	s.AddStep(step1)

	// Execute saga
	if err := s.Execute(); err != nil {
		fmt.Printf("Saga failed: %v\n", err)
		return
	}

	fmt.Println("Saga completed successfully")
}
```

### Using Step Groups

Step groups allow executing multiple steps sequentially or in parallel within a single logical step.

```go
// Create a step group
group := saga.NewStepGroup("group1").Parallel()
group.AddStep(saga.NewStep("step1", saga.NoOp, saga.NoOp))
group.AddStep(saga.NewStep("step2", saga.NoOp, saga.NoOp))

// Add group to saga
s.AddStep(group)
```

### Asynchronous Execution

Run a saga asynchronously with progress updates and cancellation support.

```go
resultCh, cancel := s.ExecuteWithProgress(
	saga.WithProgressUpdates(true),
	saga.WithOnProgress(func(completed, total int) {
		fmt.Printf("Progress: %d/%d steps completed\n", completed, total)
	}),
	saga.WithOnPartialError(func(err error, stepID saga.StepID) {
		fmt.Printf("Step %s failed: %v\n", stepID, err)
	}),
)

defer cancel()

result := <-resultCh
if result.Error != nil {
	fmt.Printf("Saga failed: %v\n", result.Error)
} else {
	fmt.Printf("Saga completed in %v\n", result.EndTime.Sub(result.StartTime))
}
```

### Configuring Retry Policies

Steps can have retry policies with customizable backoff strategies.

```go
retryPolicy := saga.NewRetryPolicy(3, time.Second).
	WithLinearBackoff(2 * time.Second) // Linear backoff: 2s, 4s, 6s

step := saga.NewStep("step1",
	func(ctx context.Context) error { return fmt.Errorf("failed") },
	saga.NoOp,
).WithRetryPolicy(retryPolicy)

s.AddStep(step)
```

## Configuration Options

### Saga Options

- `WithLogger(logger Logger)`: Sets a custom logger (e.g., `NewStdLogger`, `NewFileLogger`).
- `WithDefaultRetryPolicy(policy *RetryPolicy)`: Sets a default retry policy for all steps.
- `WithOnFailureHook(handler func(stepID StepID, err error))`: Callback for step failures.
- `WithOnStepSuccessHook(handler func(stepID StepID))`: Callback for successful step execution.
- `WithOnStepCompensatedHook(handler func(stepID StepID))`: Callback for successful compensation.
- `WithOnCompleteHook(handler func(result ExecutionResult))`: Callback for saga completion.

### Step Group Options

- `SetLogger(logger Logger)`: Sets a logger for the step group.
- `SetErrorLogger(logger Logger)`: Sets a error logger for the step group.
- `SetExecutionMode(mode ExecutionMode)`: Sets sequential or parallel execution (`Sequential` or `Parallel`).

### Retry Policy Options

- `WithLinearBackoff(baseDelay time.Duration)`: Linear backoff for retries.
- `WithFixedBackoff(delay time.Duration)`: Fixed delay for retries.
- `WithCustomBackoff(backoffFunc func(attempt int) time.Duration)`: Custom backoff function.

### Async Execution Options

- `WithProgressUpdates(enabled bool)`: Enable/disable progress updates.
- `WithOnStepStart(callback func(stepID StepID))`: Callback when a step starts.
- `WithOnProgress(callback func(completed, total int))`: Callback for progress updates.
- `WithOnPartialError(callback func(err error, stepID StepID))`: Callback for step errors.

## Important Notes

- **Compensation**: On failure, the saga automatically compensates executed steps in reverse order.
- **Thread Safety**: The `Saga` and `StepGroup` types use mutexes for safe concurrent access.
- **Context Cancellation**: Use a cancellable context to stop saga execution gracefully.
- **Retry Policies**: Steps without a retry policy execute once; configure policies for fault tolerance.
- **Logging**: Use `NewFileLogger` for persistent logs or `NewNoOpLogger` to disable logging.
- **Error Handling**: Check `ExecutionResult` for detailed failure information, including failed step ID and compensation errors.

## Dependencies

This package relies only on the Go standard library, ensuring minimal external dependencies.

## License

This package is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
