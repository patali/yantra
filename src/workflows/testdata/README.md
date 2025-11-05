# Test Workflow Fixtures

This directory contains test workflow definitions and fixtures for integration and regression testing.

## Directory Structure

```
testdata/
├── workflows/          # Workflow JSON definitions
│   ├── simple_transform.json
│   ├── conditional_loop.json
│   ├── data_aggregation.json
│   └── error_handling.json
└── fixtures/           # Test input data
    └── sample_users.json
```

## Workflow Test Cases

### 1. simple_transform.json
**Purpose**: Tests basic data transformation operations
**Node Types**: start, json, transform, end
**Test Coverage**:
- Field mapping
- Basic data flow
- Single transformation operation

**Expected Output**: Transformed object with mapped field names

---

### 2. conditional_loop.json
**Purpose**: Tests conditional branching combined with loop processing
**Node Types**: start, json, loop, conditional, transform (x2), end
**Test Coverage**:
- Array iteration
- Conditional logic evaluation
- Branch selection (true/false paths)
- Loop variable access

**Expected Output**: Processed array with conditional filtering

---

### 3. data_aggregation.json
**Purpose**: Tests complex data processing pipeline
**Node Types**: start, json-array, loop, transform, loop-accumulator, json_to_csv, end
**Test Coverage**:
- JSON array validation
- Loop iteration
- Data accumulation
- CSV conversion
- Multi-step transformation

**Expected Output**: CSV formatted string

---

### 4. error_handling.json
**Purpose**: Tests error detection and propagation
**Node Types**: start, json, conditional, transform, end
**Test Coverage**:
- Missing required fields
- Invalid configurations
- Error propagation through nodes
- Graceful failure handling

**Expected Output**: Error state with appropriate error message

---

## Test Fixtures

### sample_users.json
Sample user data for testing user processing workflows.

**Structure**:
```json
{
  "users": [
    {
      "id": number,
      "firstName": string,
      "lastName": string,
      "email": string,
      "age": number,
      "active": boolean,
      "department": string
    }
  ]
}
```

## Usage

### In Integration Tests

```go
// Load workflow definition
workflowDef := LoadWorkflowFromFile(t, "simple_transform.json")

// Load test fixtures
fixtures := LoadFixtureFromFile(t, "sample_users.json")

// Execute workflow
execution, err := ExecuteTestWorkflow(t, engineService, workflow, fixtures, 10*time.Second)
```

### Running Integration Tests

```bash
# Run all integration tests
go test ./src/workflows/... -tags=integration -v

# Run specific test
go test ./src/workflows/... -tags=integration -run TestSimpleTransformWorkflow -v

# With coverage
go test ./src/workflows/... -tags=integration -coverprofile=coverage.out
```

## Adding New Test Cases

1. **Create workflow JSON**: Add new workflow definition to `workflows/` directory
2. **Create fixtures** (if needed): Add input data to `fixtures/` directory
3. **Add test function**: Create test in `integration_test.go`
4. **Document**: Update this README with workflow description

### Workflow JSON Format

```json
{
  "name": "Workflow Name",
  "description": "What this workflow tests",
  "nodes": [
    {
      "id": "unique-id",
      "type": "node-type",
      "label": "Display Label",
      "position": {"x": 100, "y": 100},
      "config": {
        // Node-specific configuration
      }
    }
  ],
  "edges": [
    {
      "id": "edge-id",
      "source": "source-node-id",
      "target": "target-node-id",
      "label": "optional-label"
    }
  ]
}
```

## Regression Testing

These workflows serve as regression tests to ensure:
- Node types continue to work correctly
- Multi-node workflows execute properly
- Breaking changes are detected early
- Performance remains acceptable

Run before:
- Merging pull requests
- Releasing new versions
- Making changes to core execution engine
- Adding or modifying node types

## CI/CD Integration

Add to your CI pipeline:

```yaml
- name: Run Integration Tests
  run: |
    go test ./src/workflows/... -tags=integration -v
  env:
    TEST_DATABASE_URL: postgres://postgres:postgres@localhost:5432/yantra_test
```
