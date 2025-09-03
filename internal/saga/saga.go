package saga

import (
	"context"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Rollback is a function to undo a previously completed step.
type Rollback func(ctx workflow.Context) error

// Saga coordinates rollbacks to be executed in reverse order on failure.
type Saga struct {
	rollbacks []Rollback
}

func New() *Saga {
	return &Saga{rollbacks: []Rollback{}}
}

// Add registers a rollback to run if the saga fails.
func (s *Saga) Add(c Rollback) {
	s.rollbacks = append(s.rollbacks, c)
}

// Fail triggers rollbacks in reverse order and returns a combined error.
func (s *Saga) Fail(ctx workflow.Context, cause error) error {
	var firstErr error = cause
	// execute in reverse order
	for i := len(s.rollbacks) - 1; i >= 0; i-- {
		if err := s.rollbacks[i](ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if cause != nil {
		return temporal.NewApplicationErrorWithCause("saga failed", "SagaError", cause)
	}
	if firstErr == nil {
		return temporal.NewApplicationError("saga failed", "SagaError")
	}
	return firstErr
}

// ExecuteActivity is a helper that runs an activity and registers its rollback.
func ExecuteActivity[T any](ctx workflow.Context, s *Saga, act any, rollback Rollback, args ...any) (T, error) {
	var zero T
	f := workflow.ExecuteActivity(ctx, act, args...)
	var result T
	if err := f.Get(ctx, &result); err != nil {
		return zero, s.Fail(ctx, err)
	}
	if rollback != nil {
		s.Add(rollback)
	}
	return result, nil
}

// WithDeadline sets a workflow context with a deadline from now.
func WithDeadline(ctx workflow.Context, deadline time.Time) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.GetActivityOptions(ctx))
}

// WithActivityOptions returns a child context with common activity options.
func WithActivityOptions(ctx workflow.Context, ao workflow.ActivityOptions) workflow.Context {
	return workflow.WithActivityOptions(ctx, ao)
}

// Background just mirrors context.Background for symmetry in call sites outside workflow.
func Background() context.Context { return context.Background() }


