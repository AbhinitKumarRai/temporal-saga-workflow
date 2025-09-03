package workflow

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/AbhinitKumarRai/temporal-saga-workflow/internal/activities"
	configpkg "github.com/AbhinitKumarRai/temporal-saga-workflow/pkg/config"
	"go.temporal.io/sdk/testsuite"
)

type mockStore struct {
    mu            sync.Mutex
    deletions     []string
    nextIDCounter int
}

func (m *mockStore) recordDelete(tag string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.deletions = append(m.deletions, tag)
}

func setupServer(t *testing.T, behavior map[string]func(http.ResponseWriter, *http.Request)) *httptest.Server {
    t.Helper()
    mux := http.NewServeMux()
    for path, h := range behavior {
        mux.HandleFunc(path, h)
    }
    return httptest.NewServer(mux)
}

func defaultHandlers(t *testing.T, store *mockStore, fail map[string]bool, sleep map[string]time.Duration) map[string]func(http.ResponseWriter, *http.Request) {
    return map[string]func(http.ResponseWriter, *http.Request){
        "/api1/create": func(w http.ResponseWriter, r *http.Request) {
            if d := sleep["api1"]; d > 0 {
                time.Sleep(d)
            }
            if fail["api1"] {
                http.Error(w, "fail1", http.StatusInternalServerError)
                return
            }
            _ = json.NewDecoder(r.Body).Decode(&struct{}{})
            _ = json.NewEncoder(w).Encode(activities.ResponsePayload{Status: "ok", ID: "a1"})
        },
        "/api2/create": func(w http.ResponseWriter, r *http.Request) {
            if d := sleep["api2"]; d > 0 {
                time.Sleep(d)
            }
            if fail["api2"] {
                http.Error(w, "fail2", http.StatusInternalServerError)
                return
            }
            _ = json.NewDecoder(r.Body).Decode(&struct{}{})
            _ = json.NewEncoder(w).Encode(activities.ResponsePayload{Status: "ok", ID: "b2"})
        },
        "/api3/create": func(w http.ResponseWriter, r *http.Request) {
            if d := sleep["api3"]; d > 0 {
                time.Sleep(d)
            }
            if fail["api3"] {
                http.Error(w, "fail3", http.StatusInternalServerError)
                return
            }
            _ = json.NewDecoder(r.Body).Decode(&struct{}{})
            _ = json.NewEncoder(w).Encode(activities.ResponsePayload{Status: "ok", ID: "c3"})
        },
        "/api1/a1": func(w http.ResponseWriter, r *http.Request) {
            if r.Method == http.MethodDelete {
                store.recordDelete("api1:a1")
                w.WriteHeader(200)
                _, _ = w.Write([]byte("{\"status\":\"deleted\"}"))
                return
            }
            w.WriteHeader(405)
        },
        "/api2/b2": func(w http.ResponseWriter, r *http.Request) {
            if r.Method == http.MethodDelete {
                store.recordDelete("api2:b2")
                w.WriteHeader(200)
                _, _ = w.Write([]byte("{\"status\":\"deleted\"}"))
                return
            }
            w.WriteHeader(405)
        },
        "/api3/c3": func(w http.ResponseWriter, r *http.Request) {
            if r.Method == http.MethodDelete {
                store.recordDelete("api3:c3")
                w.WriteHeader(200)
                _, _ = w.Write([]byte("{\"status\":\"deleted\"}"))
                return
            }
            w.WriteHeader(405)
        },
    }
}

func newCfg(base string) configpkg.Config {
    return configpkg.Config{
        TemporalAddress:            "",
        TemporalNamespace:          "default",
        TemporalTaskQueue:          "saga-task-queue-test",
        TransactionTimeoutSeconds:  10,
        API1BaseURL:                base + "/api1",
        API2BaseURL:                base + "/api2",
        API3BaseURL:                base + "/api3",
        MockMode:                   false,
        HTTPTimeoutSeconds:         2,
        ServerPort:                 "0",
    }
}

func registerActivities(env *testsuite.TestWorkflowEnvironment, cfg configpkg.Config) {
    acts := &activities.Activities{Cfg: cfg}
    env.RegisterActivity(acts.Step1)
    env.RegisterActivity(acts.Step2)
    env.RegisterActivity(acts.Step3)
    env.RegisterActivity(acts.Rollback)
}

func Test_Saga_Success(t *testing.T) {
    var suite testsuite.WorkflowTestSuite
    env := suite.NewTestWorkflowEnvironment()
    env.SetTestTimeout(10 * time.Second)
    store := &mockStore{}
    srv := setupServer(t, defaultHandlers(t, store, map[string]bool{}, map[string]time.Duration{}))
    defer srv.Close()

    cfg := newCfg(srv.URL)
    env.RegisterWorkflow(SagaWorkflow)
    registerActivities(env, cfg)

    env.ExecuteWorkflow(SagaWorkflow, cfg, OperationInput{Method: http.MethodPost, Data: map[string]any{"k": "v"}})
    if !env.IsWorkflowCompleted() || env.GetWorkflowError() != nil {
        t.Fatalf("workflow failed: %v", env.GetWorkflowError())
    }
    var out OperationResult
    _ = env.GetWorkflowResult(&out)
    if out.Step1ID != "a1" || out.Step2ID != "b2" || out.Step3ID != "c3" {
        t.Fatalf("unexpected output: %+v", out)
    }
    if len(store.deletions) != 0 {
        t.Fatalf("unexpected compensations: %+v", store.deletions)
    }
}

func Test_Saga_Fail_Step2_Rollback1(t *testing.T) {
    var suite testsuite.WorkflowTestSuite
    env := suite.NewTestWorkflowEnvironment()
    env.SetTestTimeout(10 * time.Second)
    store := &mockStore{}
    srv := setupServer(t, defaultHandlers(t, store, map[string]bool{"api2": true}, map[string]time.Duration{}))
    defer srv.Close()

    cfg := newCfg(srv.URL)
    env.RegisterWorkflow(SagaWorkflow)
    registerActivities(env, cfg)

    env.ExecuteWorkflow(SagaWorkflow, cfg, OperationInput{Method: http.MethodPost, Data: map[string]any{"k": "v"}})
    if !env.IsWorkflowCompleted() || env.GetWorkflowError() == nil {
        t.Fatalf("expected workflow error but got nil")
    }
    if len(store.deletions) != 1 || store.deletions[0] != "api1:a1" {
        t.Fatalf("expected rollback of step1 only, got %+v", store.deletions)
    }
}

func Test_Saga_Fail_Step3_Rollback2Then1(t *testing.T) {
    var suite testsuite.WorkflowTestSuite
    env := suite.NewTestWorkflowEnvironment()
    env.SetTestTimeout(10 * time.Second)
    store := &mockStore{}
    srv := setupServer(t, defaultHandlers(t, store, map[string]bool{"api3": true}, map[string]time.Duration{}))
    defer srv.Close()

    cfg := newCfg(srv.URL)
    env.RegisterWorkflow(SagaWorkflow)
    registerActivities(env, cfg)

    env.ExecuteWorkflow(SagaWorkflow, cfg, OperationInput{Method: http.MethodPost, Data: map[string]any{"k": "v"}})
    if !env.IsWorkflowCompleted() || env.GetWorkflowError() == nil {
        t.Fatalf("expected workflow error but got nil")
    }
    if len(store.deletions) != 2 || store.deletions[0] != "api2:b2" || store.deletions[1] != "api1:a1" {
        t.Fatalf("expected rollback order [api2, api1], got %+v", store.deletions)
    }
}

func Test_Saga_Timeout_Rollback1(t *testing.T) {
    var suite testsuite.WorkflowTestSuite
    env := suite.NewTestWorkflowEnvironment()
    env.SetTestTimeout(10 * time.Second)
    store := &mockStore{}
    // Sleep on api2 longer than HTTP timeout to force activity timeout
    srv := setupServer(t, defaultHandlers(t, store, map[string]bool{}, map[string]time.Duration{"api2": 3 * time.Second}))
    defer srv.Close()

    cfg := newCfg(srv.URL)
    cfg.HTTPTimeoutSeconds = 1
    env.RegisterWorkflow(SagaWorkflow)
    registerActivities(env, cfg)

    env.ExecuteWorkflow(SagaWorkflow, cfg, OperationInput{Method: http.MethodPost, Data: map[string]any{"k": "v"}})
    if !env.IsWorkflowCompleted() || env.GetWorkflowError() == nil {
        t.Fatalf("expected workflow timeout/error")
    }
    if len(store.deletions) != 1 || store.deletions[0] != "api1:a1" {
        t.Fatalf("expected rollback of step1 only, got %+v", store.deletions)
    }
}


