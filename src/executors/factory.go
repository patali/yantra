package executors

import (
	"fmt"

	"gorm.io/gorm"
)

// ExecutorFactory provides a registry of executors
type ExecutorFactory struct {
	executors map[string]Executor
}

// NewExecutorFactory creates a new executor factory
func NewExecutorFactory(db *gorm.DB) *ExecutorFactory {
	factory := &ExecutorFactory{
		executors: make(map[string]Executor),
	}

	// Register executors
	factory.Register("json-array", NewJsonArrayTriggerExecutor())
	factory.Register("conditional", NewConditionalExecutor())
	factory.Register("transform", NewTransformExecutor())
	factory.Register("delay", NewDelayExecutor())
	factory.Register("email", NewEmailExecutor(db))
	factory.Register("http", NewHTTPExecutor())
	factory.Register("slack", NewSlackExecutor())
	factory.Register("loop", NewLoopExecutor())
	factory.Register("loop-accumulator", NewLoopAccumulatorExecutor())
	factory.Register("json_to_csv", NewJSONToCSVExecutor())
	factory.Register("json", NewJSONExecutor()) // JSON data node

	return factory
}

// SetEmailService sets the email service for the email executor
func (f *ExecutorFactory) SetEmailService(service EmailServiceInterface) {
	if emailExecutor, ok := f.executors["email"].(*EmailExecutor); ok {
		emailExecutor.SetEmailService(service)
	}
}

// Register registers an executor for a node type
func (f *ExecutorFactory) Register(nodeType string, executor Executor) {
	f.executors[nodeType] = executor
}

// GetExecutor returns an executor for a node type
func (f *ExecutorFactory) GetExecutor(nodeType string) (Executor, error) {
	executor, exists := f.executors[nodeType]
	if !exists {
		return nil, fmt.Errorf("no executor found for node type: %s", nodeType)
	}
	return executor, nil
}
