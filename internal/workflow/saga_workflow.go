package workflow

import (
	"time"

	"github.com/AbhinitKumarRai/temporal-saga-workflow/internal/activities"
	saga "github.com/AbhinitKumarRai/temporal-saga-workflow/internal/saga"
	configpkg "github.com/AbhinitKumarRai/temporal-saga-workflow/pkg/config"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type OperationInput struct {
	Method string         `json:"method"`
	Data   map[string]any `json:"data"`
	// Optional resource IDs for PUT/DELETE
	ID1 string `json:"id1,omitempty"`
	ID2 string `json:"id2,omitempty"`
	ID3 string `json:"id3,omitempty"`
}

type OperationResult struct {
	Step1ID string `json:"step1_id"`
	Step2ID string `json:"step2_id"`
	Step3ID string `json:"step3_id"`
}

func SagaWorkflow(ctx workflow.Context, cfg configpkg.Config, in OperationInput) (OperationResult, error) {
	var result OperationResult
	s := saga.New()

	ao := workflow.ActivityOptions{
		StartToCloseTimeout:    cfg.HTTPTimeout(),
		ScheduleToCloseTimeout: cfg.TransactionTimeout(),
		ScheduleToStartTimeout: cfg.HTTPTimeout(),
		HeartbeatTimeout:      cfg.HTTPTimeout() / 2,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumAttempts:    3,
		},
	}
	ctx = saga.WithActivityOptions(ctx, ao)

	acts := &activities.Activities{Cfg: cfg}

	// Helper to decide if we should add compensation (only for POST create)
	shouldRollback := in.Method == "POST"

	// Step 1
	rollback1 := func(c workflow.Context) error {
		if !shouldRollback || result.Step1ID == "" {
			return nil
		}
		return workflow.ExecuteActivity(c, acts.Rollback, cfg.API1BaseURL, result.Step1ID).Get(c, nil)
	}
	res1, err := saga.ExecuteActivity[activities.StepResult](ctx, s, acts.Step1, rollback1, activities.StepInput{
		BaseURL:    cfg.API1BaseURL,
		Method:     in.Method,
		ResourceID: in.ID1,
		Payload:    activities.RequestPayload{Operation: "step1", Data: in.Data},
	})
	if err != nil {
		return result, err
	}
	result.Step1ID = res1.ResourceID

	// Step 2
	comp2 := func(c workflow.Context) error {
		if !shouldRollback || result.Step2ID == "" {
			return nil
		}
		return workflow.ExecuteActivity(c, acts.Rollback, cfg.API2BaseURL, result.Step2ID).Get(c, nil)
	}
	res2, err := saga.ExecuteActivity[activities.StepResult](ctx, s, acts.Step2, comp2, activities.StepInput{
		BaseURL:    cfg.API2BaseURL,
		Method:     in.Method,
		ResourceID: in.ID2,
		Payload:    activities.RequestPayload{Operation: "step2", Data: in.Data},
	})
	if err != nil {
		return result, err
	}
	result.Step2ID = res2.ResourceID

	// Step 3
	comp3 := func(c workflow.Context) error {
		if !shouldRollback || result.Step3ID == "" {
			return nil
		}
		return workflow.ExecuteActivity(c, acts.Rollback, cfg.API3BaseURL, result.Step3ID).Get(c, nil)
	}
	res3, err := saga.ExecuteActivity[activities.StepResult](ctx, s, acts.Step3, comp3, activities.StepInput{
		BaseURL:    cfg.API3BaseURL,
		Method:     in.Method,
		ResourceID: in.ID3,
		Payload:    activities.RequestPayload{Operation: "step3", Data: in.Data},
	})
	if err != nil {
		return result, err
	}
	result.Step3ID = res3.ResourceID

	return result, nil
}


