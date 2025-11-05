package executors

import (
	"context"
	"testing"
)

// Mock email service for testing
type MockEmailService struct {
	SentEmails []EmailOptions
	ShouldFail bool
	ErrorMsg   string
}

func (m *MockEmailService) SendEmail(ctx context.Context, accountID string, options EmailOptions) (*EmailResult, error) {
	if m.ShouldFail {
		return &EmailResult{
			Success: false,
			Error:   m.ErrorMsg,
		}, nil
	}

	// Track sent emails for verification
	m.SentEmails = append(m.SentEmails, options)

	return &EmailResult{
		Success:   true,
		MessageID: "mock-message-id",
	}, nil
}

// NewMockEmailService creates a new mock email service
func NewMockEmailService(shouldFail bool) *MockEmailService {
	return &MockEmailService{
		SentEmails: []EmailOptions{},
		ShouldFail: shouldFail,
		ErrorMsg:   "mock email error",
	}
}

// TestExecutionContextBuilder helps build test execution contexts
type TestExecutionContextBuilder struct {
	nodeID       string
	nodeConfig   map[string]interface{}
	input        interface{}
	workflowData map[string]interface{}
	executionID  string
	accountID    string
}

// NewTestExecutionContext creates a new builder
func NewTestExecutionContext() *TestExecutionContextBuilder {
	return &TestExecutionContextBuilder{
		nodeID:       "test-node",
		nodeConfig:   map[string]interface{}{},
		workflowData: map[string]interface{}{},
		executionID:  "test-execution",
		accountID:    "test-account",
	}
}

// WithNodeID sets the node ID
func (b *TestExecutionContextBuilder) WithNodeID(id string) *TestExecutionContextBuilder {
	b.nodeID = id
	return b
}

// WithConfig sets the node configuration
func (b *TestExecutionContextBuilder) WithConfig(config map[string]interface{}) *TestExecutionContextBuilder {
	b.nodeConfig = config
	return b
}

// WithConfigValue sets a single config value
func (b *TestExecutionContextBuilder) WithConfigValue(key string, value interface{}) *TestExecutionContextBuilder {
	b.nodeConfig[key] = value
	return b
}

// WithInput sets the input data
func (b *TestExecutionContextBuilder) WithInput(input interface{}) *TestExecutionContextBuilder {
	b.input = input
	return b
}

// WithWorkflowData sets the workflow data
func (b *TestExecutionContextBuilder) WithWorkflowData(data map[string]interface{}) *TestExecutionContextBuilder {
	b.workflowData = data
	return b
}

// WithExecutionID sets the execution ID
func (b *TestExecutionContextBuilder) WithExecutionID(id string) *TestExecutionContextBuilder {
	b.executionID = id
	return b
}

// WithAccountID sets the account ID
func (b *TestExecutionContextBuilder) WithAccountID(id string) *TestExecutionContextBuilder {
	b.accountID = id
	return b
}

// Build returns the execution context
func (b *TestExecutionContextBuilder) Build() ExecutionContext {
	return ExecutionContext{
		NodeID:       b.nodeID,
		NodeConfig:   b.nodeConfig,
		Input:        b.input,
		WorkflowData: b.workflowData,
		ExecutionID:  b.executionID,
		AccountID:    b.accountID,
	}
}

// AssertExecutionSuccess is a helper to assert successful execution
func AssertExecutionSuccess(t *testing.T, result *ExecutionResult, err error) {
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}
}

// AssertExecutionError is a helper to assert execution error
func AssertExecutionError(t *testing.T, result *ExecutionResult, err error, expectedError string) {
	if err != nil && expectedError != "" {
		// Some executors return errors directly
		return
	}
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.Success {
		t.Fatal("Expected failure, got success")
	}
	if expectedError != "" && result.Error != expectedError {
		t.Fatalf("Expected error containing '%s', got: %s", expectedError, result.Error)
	}
}

// GenerateTestArray creates a test array of specified size
func GenerateTestArray(size int) []interface{} {
	items := make([]interface{}, size)
	for i := 0; i < size; i++ {
		items[i] = map[string]interface{}{
			"id":    i,
			"value": i * 10,
		}
	}
	return items
}

// GenerateTestUsers creates a test array of user objects
func GenerateTestUsers(count int) []interface{} {
	users := make([]interface{}, count)
	names := []string{"John", "Jane", "Bob", "Alice", "Charlie"}
	for i := 0; i < count; i++ {
		users[i] = map[string]interface{}{
			"id":    i + 1,
			"name":  names[i%len(names)],
			"email": names[i%len(names)] + "@example.com",
			"age":   20 + (i % 50),
		}
	}
	return users
}
