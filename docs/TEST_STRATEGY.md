# Yantra Test Strategy & Reorganization Plan

## Overview
This document outlines the testing strategy for the Yantra workflow automation server, including unit tests, integration tests, and regression testing approach.

## Current State

### Existing Test Organization
- **Unit Tests for Executors**: All executor tests currently in `src/executors/executors_test.go` (867 lines)
- **JSON Executor Tests**: Separated in `src/executors/json_test.go` (170 lines)
- **Service Tests**: Auth and Scheduler services in `src/services/`
- **Coverage**: 11/13 node types (85% coverage)
- **Missing**: Integration tests for multi-node workflows

### Test Framework
- Go's built-in `testing` package
- `github.com/stretchr/testify` for assertions
- PostgreSQL for database integration tests

## Test Reorganization Plan

### 1. Separate Unit Tests by Node Type

Create individual test files for each executor type to improve maintainability:

```
src/executors/
├── conditional_test.go       # ConditionalExecutor tests
├── delay_test.go             # DelayExecutor tests
├── email_test.go             # EmailExecutor tests
├── http_test.go              # HTTPExecutor tests
├── json_test.go              # JSONExecutor tests (already exists)
├── json_array_test.go        # JsonArrayTriggerExecutor tests
├── json_to_csv_test.go       # JSONToCSVExecutor tests
├── loop_test.go              # LoopExecutor tests
├── loop_accumulator_test.go  # LoopAccumulatorExecutor tests
├── slack_test.go             # SlackExecutor tests
├── transform_test.go         # TransformExecutor tests
└── test_helpers.go           # Shared test utilities and mocks
```

**Benefits:**
- Easier to locate specific node type tests
- Parallel test execution optimization
- Reduced merge conflicts
- Clearer test ownership

### 2. Integration Tests

Create integration tests for end-to-end workflow execution:

```
src/workflows/
├── integration_test.go           # Integration test framework
├── testdata/
│   ├── workflows/                # Test workflow definitions
│   │   ├── simple_transform.json
│   │   ├── conditional_loop.json
│   │   ├── http_transform.json
│   │   ├── error_handling.json
│   │   └── complex_pipeline.json
│   └── fixtures/                 # Test input/output data
│       ├── sample_users.json
│       ├── expected_results.json
│       └── api_responses.json
└── workflow_executor_test.go     # Core workflow execution tests
```

**Integration Test Scenarios:**

#### Implemented Workflows ✅
1. **Simple Transform Pipeline** (`simple_transform.json`)
   - JSON → Transform → Output
   - Tests: Data transformation, field mapping

2. **Conditional Loop** (`conditional_loop.json`)
   - JSON Array → Loop → Conditional → Transform
   - Tests: Branch selection, iteration, data flow

3. **Data Aggregation** (`data_aggregation.json`)
   - JSON → Transform → Loop → Accumulator
   - Tests: Iteration, accumulation, data collection

4. **Error Handling** (`error_handling.json`)
   - Invalid Input → Node Failure → Error Propagation
   - Tests: Graceful degradation, error messages

#### Workflows Not Yet Implemented
5. **HTTP Integration Pipeline**
   - HTTP Request → Transform → Conditional → Email/Slack
   - Tests: External API calls, error handling
   - Status: Planned

6. **Complex CSV Pipeline**
   - JSON → Transform → Loop → JSON to CSV
   - Tests: Complex data manipulation, CSV export
   - Status: Planned

#### Edge Cases
7. **Large Data Processing**
   - 1000+ item array → Loop with limits
   - Tests: Performance, memory usage, limits

8. **Concurrent Execution**
   - Multiple workflows running simultaneously
   - Tests: Isolation, resource management

### 3. Regression Test Workflows

Create a comprehensive suite of real-world workflow examples:

```
src/workflows/testdata/regression/
├── README.md                          # Documentation
├── ecommerce_order_processing.json    # E-commerce workflow
├── user_onboarding.json               # User automation
├── data_aggregation.json              # Data processing
├── notification_system.json           # Multi-channel notifications
├── scheduled_reports.json             # Cron-based workflows
└── webhook_handlers.json              # Webhook processing
```

**Regression Test Coverage:**

| Workflow | Node Types Used | Purpose |
|----------|----------------|---------|
| E-commerce Order | HTTP, Conditional, Email, Slack | Order processing with notifications |
| User Onboarding | JSON, Transform, Loop, Email | Batch user setup |
| Data Aggregation | HTTP, Loop, Transform, JSON-to-CSV | API data collection |
| Notification System | Conditional, Email, Slack, Delay | Smart notification routing |
| Scheduled Reports | JSON, Transform, Loop, JSON-to-CSV | Periodic data reports |
| Webhook Handlers | JSON, Conditional, HTTP, Transform | Webhook processing |

### 4. Test Utilities and Mocks

Create shared testing infrastructure:

```go
// src/executors/test_helpers.go

// Mock services
type MockEmailService struct {
    SentEmails []EmailOptions
    ShouldFail bool
}

type MockHTTPClient struct {
    Responses map[string]*http.Response
    Requests  []*http.Request
}

// Test data generators
func GenerateTestExecutionContext(nodeType string, config map[string]interface{}) ExecutionContext
func GenerateTestWorkflow(nodes []TestNode) *Workflow
func GenerateTestArray(size int, schema map[string]interface{}) []interface{}

// Assertion helpers
func AssertExecutionSuccess(t *testing.T, result *ExecutionResult)
func AssertExecutionError(t *testing.T, result *ExecutionResult, expectedError string)
func AssertOutputContains(t *testing.T, result *ExecutionResult, key string, value interface{})
```

### 5. Performance Benchmarks

Add benchmark tests for critical operations:

```go
// src/executors/benchmarks_test.go

func BenchmarkTransformExecutor_SimpleMap(b *testing.B)
func BenchmarkLoopExecutor_1000Items(b *testing.B)
func BenchmarkConditionalExecutor_ComplexExpression(b *testing.B)
func BenchmarkJSONToCSVExecutor_LargeDataset(b *testing.B)
```

## Implementation Phases

### Phase 1: Reorganize Unit Tests (Priority: High)
- [x] Analyze current test structure
- [x] Create `test_helpers.go` with shared utilities
- [x] Split `executors_test.go` into individual files:
  - [x] `conditional_test.go`
  - [x] `delay_test.go`
  - [x] `transform_test.go`
  - [x] `loop_test.go`
  - [x] `loop_accumulator_test.go`
  - [x] `json_array_test.go`
  - [x] `json_to_csv_test.go`
  - [x] `http_test.go`
  - [x] `email_test.go`
  - [x] `slack_test.go`
- [x] Verify all tests still pass
- [x] Remove original `executors_test.go`

### Phase 2: Create Integration Test Framework (Priority: High)
- [x] Design integration test structure
- [x] Create `src/workflows/integration_test.go`
- [x] Set up test database and cleanup utilities
- [x] Implement workflow execution test harness
- [x] Add basic integration tests (4 workflows)

### Phase 3: Add Regression Test Workflows (Priority: Medium)
- [ ] Design 6 real-world workflow scenarios
- [ ] Create JSON workflow definitions
- [ ] Create test fixtures and expected outputs
- [ ] Implement regression test runner
- [ ] Document regression test usage

### Phase 4: Performance & Benchmarks (Priority: Low)
- [ ] Add benchmark tests for each executor
- [ ] Add performance regression detection
- [ ] Document performance baselines

## Test Execution

### Running Tests

```bash
# All tests
go test ./...

# Unit tests only (executors)
go test ./src/executors/...

# Integration tests only
go test ./src/workflows/... -tags=integration

# Specific node type
go test ./src/executors/ -run TestConditional

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. ./src/executors/...

# Verbose output
go test -v ./...
```

### CI/CD Integration

Recommended GitHub Actions workflow:

```yaml
name: Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test ./src/executors/... ./src/services/...

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test ./src/workflows/... -tags=integration

  regression-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test ./src/workflows/... -run Regression
```

## Success Metrics

### Coverage Goals
- **Unit Test Coverage**: >90% for all executors
- **Integration Test Coverage**: >80% for workflow engine
- **Regression Test Coverage**: 100% of major node type combinations

### Quality Metrics
- All tests pass consistently
- No flaky tests (>99% reliability)
- Test execution time <2 minutes for unit tests
- Integration tests complete in <5 minutes

## Test Maintenance

### Guidelines
1. **Every new node type** must include:
   - Unit tests (happy path, error cases, edge cases)
   - Integration test example
   - Regression workflow (if applicable)

2. **Every bug fix** must include:
   - Regression test reproducing the bug
   - Verification test ensuring fix works

3. **Every feature change** must include:
   - Updated tests for affected nodes
   - Integration test for new workflow patterns

### Code Review Checklist
- [ ] All new code has test coverage
- [ ] Tests are clear and well-documented
- [ ] No hardcoded test data (use fixtures)
- [ ] Tests clean up after themselves
- [ ] No real external API calls (use mocks)
- [ ] Tests run in isolation (no dependencies)

## References

- Go Testing Documentation: https://golang.org/pkg/testing/
- Testify Library: https://github.com/stretchr/testify
- Go Test Patterns: https://golang.org/doc/tutorial/add-a-test

---

**Last Updated**: 2025-11-05
**Status**: Phase 1 & 2 Complete ✅ | Phase 3 & 4 Planned
