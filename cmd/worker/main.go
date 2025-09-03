package main

import (
	"log"

	"github.com/AbhinitKumarRai/temporal-saga-workflow/internal/activities"
	workflowpkg "github.com/AbhinitKumarRai/temporal-saga-workflow/internal/workflow"
	"github.com/AbhinitKumarRai/temporal-saga-workflow/pkg/config"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	cl, err := client.NewClient(client.Options{HostPort: cfg.TemporalAddress, Namespace: cfg.TemporalNamespace})
	if err != nil {
		log.Fatalf("unable to create Temporal client: %v", err)
	}
	defer cl.Close()

	w := worker.New(cl, cfg.TemporalTaskQueue, worker.Options{})
	w.RegisterWorkflow(workflowpkg.SagaWorkflow)

	acts := &activities.Activities{Cfg: cfg}
	w.RegisterActivity(acts.Step1)
	w.RegisterActivity(acts.Step2)
	w.RegisterActivity(acts.Step3)
	w.RegisterActivity(acts.Rollback)

	log.Printf("Worker started. TaskQueue=%s", cfg.TemporalTaskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("worker failed: %v", err)
	}
}


