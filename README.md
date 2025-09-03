## Temporal Saga Workflow with External APIs (Go)

### What this provides

- Saga-based Temporal workflow orchestrating three external API calls with compensations
- Configurable end-to-end timeouts and activity-level timeouts
- REST API to trigger create, update, delete flows
- Docker Compose to run Worker, API
- Automated tests: success path, failure at step2/step3 (rollbacks), timeout

### Endpoints (API)

- POST `/create` → calls three APIs with Method=POST
- POST `/update` → calls three APIs with Method=PUT (requires `id1/id2/id3`)
- POST `/delete` → calls three APIs with Method=DELETE (requires `id1/id2/id3`)

All accept JSON body:

```json
{
  "workflow_id": "saga-123",
  "data": { "key": "value" },
  "id1": "resource-id-step1",
  "id2": "resource-id-step2",
  "id3": "resource-id-step3"
}
```

- For create, omit `id1/id2/id3`.
- Add `?wait=true` query to block for workflow result.

**Example - Difference between fire-and-forget vs wait-for-result:**

**Without `?wait=true` (Fire and forget):**

```bash
curl -X POST http://localhost:8080/create \
  -H "Content-Type: application/json" \
  -d '{"workflow_id": "test-1", "data": {"name": "test"}}'
```

**Response (immediate):**

```json
{
  "workflow_id": "test-1",
  "run_id": "abc123def456"
}
```

**With `?wait=true` (Wait for completion):**

```bash
curl -X POST "http://localhost:8080/create?wait=true" \
  -H "Content-Type: application/json" \
  -d '{"workflow_id": "test-2", "data": {"name": "test"}}'
```

**Response (after workflow completes):**

```json
{
  "workflow_id": "test-2",
  "run_id": "def456ghi789",
  "result": {
    "step1_id": "resource-1-created",
    "step2_id": "resource-2-created",
    "step3_id": "resource-3-created"
  }
}
```

**Use cases:**

- **Fire and forget**: When you want to start the workflow and don't need the result immediately (e.g., background processing)
- **Wait for result**: When you need the created resource IDs or want to ensure the workflow completed successfully before proceeding

### API Examples

#### cURL Examples

**Create (POST) - Fire and forget:**

```bash
curl -X POST http://localhost:8080/create \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "create-123",
    "data": {
      "name": "John Doe",
      "email": "john@example.com",
      "amount": 100
    }
  }'
```

**Create (POST) - Wait for result:**

```bash
curl -X POST "http://localhost:8080/create?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "create-456",
    "data": {
      "name": "Jane Smith",
      "email": "jane@example.com",
      "amount": 200
    }
  }'
```

**Update (PUT) - Wait for result:**

```bash
curl -X POST "http://localhost:8080/update?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "update-789",
    "data": {
      "name": "John Updated",
      "email": "john.updated@example.com",
      "amount": 150
    },
    "id1": "resource-1-id",
    "id2": "resource-2-id",
    "id3": "resource-3-id"
  }'
```

**Delete (DELETE) - Fire and forget:**

```bash
curl -X POST http://localhost:8080/delete \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "delete-101",
    "data": {},
    "id1": "resource-1-id",
    "id2": "resource-2-id",
    "id3": "resource-3-id"
  }'
```

#### Postman Examples

**Create Request:**

- Method: `POST`
- URL: `http://localhost:8080/create`
- Headers: `Content-Type: application/json`
- Body (raw JSON):

```json
{
  "workflow_id": "create-postman-1",
  "data": {
    "customer": "Acme Corp",
    "order": "ORD-001",
    "items": ["item1", "item2", "item3"]
  }
}
```

**Update Request:**

- Method: `POST`
- URL: `http://localhost:8080/update?wait=true`
- Headers: `Content-Type: application/json`
- Body (raw JSON):

```json
{
  "workflow_id": "update-postman-1",
  "data": {
    "customer": "Acme Corp Updated",
    "order": "ORD-001-REV",
    "items": ["item1", "item2", "item3", "item4"]
  },
  "id1": "customer-123",
  "id2": "order-456",
  "id3": "invoice-789"
}
```

**Delete Request:**

- Method: `POST`
- URL: `http://localhost:8080/delete`
- Headers: `Content-Type: application/json`
- Body (raw JSON):

```json
{
  "workflow_id": "delete-postman-1",
  "data": {},
  "id1": "customer-123",
  "id2": "order-456",
  "id3": "invoice-789"
}
```

### Expected Responses

**Fire and forget response:**

```json
{
  "workflow_id": "create-123",
  "run_id": "abc123def456"
}
```

**Wait for result response:**

```json
{
  "workflow_id": "create-456",
  "run_id": "def456ghi789",
  "result": {
    "step1_id": "resource-1-created",
    "step2_id": "resource-2-created",
    "step3_id": "resource-3-created"
  }
}
```

### Workflow logic (Saga)

- Workflow executes three activities sequentially (Step1, Step2, Step3)
- Activity inputs include: `base_url`, `method` (POST/PUT/DELETE), optional `resource_id`, and payload
- Rollback is registered and executed in reverse order only for create (POST). Update/Delete do not auto-rollback since they are idempotent or caller-controlled
- If any activity fails, previously completed POST steps are rollback using DELETE calls
- Timeouts and retries are applied via Temporal `ActivityOptions`

### HTTP mapping in activities

- POST → `BaseURL/create` with JSON payload; expects response with `id` (supports `id`, `_id`, or `ID` keys)
- PUT → `BaseURL/{id}` with JSON payload
- DELETE → `BaseURL/{id}`

### Configuration (env)

- `TEMPORAL_ADDRESS` (default `temporal:7233`)
- `TEMPORAL_NAMESPACE` (default `default`)
- `TEMPORAL_TASK_QUEUE` (default `saga-task-queue`)
- `TRANSACTION_TIMEOUT_SECONDS` (default `30`) – schedule-to-close for the overall workflow ops
- `HTTP_TIMEOUT_SECONDS` (default `10`) – per-activity start/heartbeat/schedule timeouts
- `API1_BASE_URL`, `API2_BASE_URL`, `API3_BASE_URL` – external endpoints base URLs (e.g., `https://crudcrud.com/api/<key>/api1`)

#### Add these envs directly either in docker-compose or in pkg/config/config.go under default values.

### Run locally

```bash
docker compose up --build
```

- Temporal UI: http://localhost:8088
- API: http://localhost:8080

### Tests

Run all tests:

```bash
go test ./...
```

Covers:

- Success across all three activities
- Failure at step 2 → rollbacks step 1
- Failure at step 3 → rollbacks step 2 then step 1
- Timeout on an activity → rollbacks prior success(es)

### Assumptions & decisions

- Rollbacks only for POST (create). For PUT/DELETE, callers control ids; automatic rollback would require domain-specific semantics
- External APIs accept the routes defined above; response `id` key may vary (`id`, `_id`, `ID`)
- Mock mode exists for activities but tests disable it to validate real HTTP behavior via `httptest`
- Activity retries use exponential backoff (3 attempts) and respect configured timeouts
