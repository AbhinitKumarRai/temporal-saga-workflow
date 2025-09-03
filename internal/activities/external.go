package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/AbhinitKumarRai/temporal-saga-workflow/pkg/config"
)

type ExternalClient struct {
	httpClient *http.Client
	cfg        config.Config
}

func NewExternalClient(cfg config.Config) *ExternalClient {
	return &ExternalClient{
		httpClient: &http.Client{Timeout: cfg.HTTPTimeout()},
		cfg:        cfg,
	}
}

type RequestPayload struct {
	Operation string                 `json:"operation"`
	Data      map[string]any         `json:"data"`
	Meta      map[string]any         `json:"meta,omitempty"`
}

type ResponsePayload struct {
	ID      string         `json:"id,omitempty"`
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Echo    map[string]any `json:"echo,omitempty"`
}

// Activity inputs
type StepInput struct {
	BaseURL    string         `json:"base_url"`
	Method     string         `json:"method"`
	ResourceID string         `json:"resource_id,omitempty"`
	Payload    RequestPayload `json:"payload"`
}

type StepResult struct {
	ResourceID string `json:"resource_id"`
}

// callExternal executes the HTTP call based on StepInput.Method.
func (c *ExternalClient) crudOperation(ctx context.Context, in StepInput) (StepResult, error) {
	var result StepResult

	// Mock mode: simulate success
	if c.cfg.MockMode {
		if in.Method == http.MethodPost {
			result.ResourceID = fmt.Sprintf("mock-%s-%d", in.Payload.Operation, time.Now().Unix())
			return result, nil
		}
		result.ResourceID = in.ResourceID
		return result, nil
	}

	var url string
	var body []byte
	switch in.Method {
	case http.MethodPost:
		url = in.BaseURL + "/create"
		body, _ = json.Marshal(in.Payload)
	case http.MethodDelete:
		if in.ResourceID == "" {
			return result, fmt.Errorf("resource_id required for DELETE")
		}
		url = fmt.Sprintf("%s/%s", in.BaseURL, in.ResourceID)
	case http.MethodPut:
		if in.ResourceID == "" {
			return result, fmt.Errorf("resource_id required for PUT")
		}
		url = fmt.Sprintf("%s/%s", in.BaseURL, in.ResourceID)
		body, _ = json.Marshal(in.Payload)
	default:
		return result, fmt.Errorf("unsupported method: %s", in.Method)
	}

	req, err := http.NewRequestWithContext(ctx, in.Method, url, bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("external API error: %d %s", resp.StatusCode, string(b))
	}
	if in.Method == http.MethodPost || in.Method == http.MethodPut {
		var out map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&out); err == nil {
			if v, ok := out["id"].(string); ok {
				result.ResourceID = v
			} else if v, ok := out["_id"].(string); ok {
				result.ResourceID = v
			} else if v, ok := out["ID"].(string); ok {
				result.ResourceID = v
			}
		}
	} else if in.Method == http.MethodDelete {
		result.ResourceID = in.ResourceID
	}
	return result, nil
}

func (c *ExternalClient) rollback(ctx context.Context, baseURL, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/%s", baseURL, id), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rollback failed: %d %s", resp.StatusCode, string(b))
	}
	return nil
}

// Temporal Activity wrappers

type Activities struct {
	Cfg config.Config
}

func (a *Activities) Step1(ctx context.Context, in StepInput) (StepResult, error) {
	client := NewExternalClient(a.Cfg)
	return client.crudOperation(ctx, in)
}

func (a *Activities) Step2(ctx context.Context, in StepInput) (StepResult, error) {
	client := NewExternalClient(a.Cfg)
	return client.crudOperation(ctx, in)
}

func (a *Activities) Step3(ctx context.Context, in StepInput) (StepResult, error) {
	client := NewExternalClient(a.Cfg)
	return client.crudOperation(ctx, in)
}

func (a *Activities) Rollback(ctx context.Context, baseURL, id string) error {
	client := NewExternalClient(a.Cfg)
	return client.rollback(ctx, baseURL, id)
}


