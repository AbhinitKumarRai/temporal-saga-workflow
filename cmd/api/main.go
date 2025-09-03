package main

import (
	"log"
	"net/http"

	workflowpkg "github.com/AbhinitKumarRai/temporal-saga-workflow/internal/workflow"
	"github.com/AbhinitKumarRai/temporal-saga-workflow/pkg/config"
	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
)

type startRequest struct {
	WorkflowID string         `json:"workflow_id"`
	Data       map[string]any `json:"data"`
	ID1        string         `json:"id1,omitempty"`
	ID2        string         `json:"id2,omitempty"`
	ID3        string         `json:"id3,omitempty"`
}

type startResponse struct {
	RunID      string                        `json:"run_id"`
	WorkflowID string                        `json:"workflow_id"`
	Result     *workflowpkg.OperationResult  `json:"result,omitempty"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	r := gin.Default()
	r.POST("/create", func(c *gin.Context) {
		var req startRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cl, err := client.NewClient(client.Options{HostPort: cfg.TemporalAddress, Namespace: cfg.TemporalNamespace})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cl.Close()

		input := workflowpkg.OperationInput{Method: http.MethodPost, Data: req.Data}
		we, err := cl.ExecuteWorkflow(c, client.StartWorkflowOptions{TaskQueue: cfg.TemporalTaskQueue, ID: req.WorkflowID}, workflowpkg.SagaWorkflow, cfg, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		resp := startResponse{RunID: we.GetRunID(), WorkflowID: we.GetID()}
		if c.Query("wait") == "true" {
			var out workflowpkg.OperationResult
			if err := we.Get(c, &out); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "workflow_id": we.GetID(), "run_id": we.GetRunID()})
				return
			}
			resp.Result = &out
		}
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/delete", func(c *gin.Context) {
		var req startRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cl, err := client.NewClient(client.Options{HostPort: cfg.TemporalAddress, Namespace: cfg.TemporalNamespace})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cl.Close()

		input := workflowpkg.OperationInput{Method: http.MethodDelete, Data: req.Data, ID1: req.ID1, ID2: req.ID2, ID3: req.ID3}
		we, err := cl.ExecuteWorkflow(c, client.StartWorkflowOptions{TaskQueue: cfg.TemporalTaskQueue, ID: req.WorkflowID}, workflowpkg.SagaWorkflow, cfg, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		resp := startResponse{RunID: we.GetRunID(), WorkflowID: we.GetID()}
		if c.Query("wait") == "true" {
			var out workflowpkg.OperationResult
			if err := we.Get(c, &out); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "workflow_id": we.GetID(), "run_id": we.GetRunID()})
				return
			}
			resp.Result = &out
		}
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/update", func(c *gin.Context) {
		var req startRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cl, err := client.NewClient(client.Options{HostPort: cfg.TemporalAddress, Namespace: cfg.TemporalNamespace})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cl.Close()

		input := workflowpkg.OperationInput{Method: http.MethodPut, Data: req.Data, ID1: req.ID1, ID2: req.ID2, ID3: req.ID3}
		we, err := cl.ExecuteWorkflow(c, client.StartWorkflowOptions{TaskQueue: cfg.TemporalTaskQueue, ID: req.WorkflowID}, workflowpkg.SagaWorkflow, cfg, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		resp := startResponse{RunID: we.GetRunID(), WorkflowID: we.GetID()}
		if c.Query("wait") == "true" {
			var out workflowpkg.OperationResult
			if err := we.Get(c, &out); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "workflow_id": we.GetID(), "run_id": we.GetRunID()})
				return
			}
			resp.Result = &out
		}
		c.JSON(http.StatusOK, resp)
	})

	log.Printf("API listening on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("api server failed: %v", err)
	}
}


