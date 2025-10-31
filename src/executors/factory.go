package executors

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// ExecutorFactory provides a stateless factory for creating executors
type ExecutorFactory struct {
	db           *gorm.DB
	emailService EmailServiceInterface
	httpClient   *http.Client
}

// NewExecutorFactory creates a new executor factory with required dependencies
func NewExecutorFactory(db *gorm.DB, emailService EmailServiceInterface) *ExecutorFactory {
	// Create a shared HTTP client with connection pooling
	// This client is reused across all executor instances to prevent resource leaks
	transport := &http.Transport{
		MaxIdleConns:        100,              // Maximum idle connections across all hosts
		MaxIdleConnsPerHost: 10,               // Maximum idle connections per host
		MaxConnsPerHost:     100,              // Maximum connections per host
		IdleConnTimeout:     90 * time.Second, // How long idle connections stay open
		TLSHandshakeTimeout: 10 * time.Second, // TLS handshake timeout
		DisableCompression:  false,            // Enable compression
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	return &ExecutorFactory{
		db:           db,
		emailService: emailService,
		httpClient:   httpClient,
	}
}

// GetExecutor creates a new executor instance for a node type
// Each call returns a new instance, making the factory stateless
// Shared resources like HTTP clients are passed to executors to prevent leaks
func (f *ExecutorFactory) GetExecutor(nodeType string) (Executor, error) {
	switch nodeType {
	case "json-array":
		return NewJsonArrayTriggerExecutor(), nil
	case "conditional":
		return NewConditionalExecutor(), nil
	case "transform":
		return NewTransformExecutor(), nil
	case "delay":
		return NewDelayExecutor(), nil
	case "email":
		return NewEmailExecutor(f.db, f.emailService), nil
	case "http":
		return NewHTTPExecutor(f.httpClient), nil
	case "slack":
		return NewSlackExecutor(f.httpClient), nil
	case "loop":
		return NewLoopExecutor(), nil
	case "loop-accumulator":
		return NewLoopAccumulatorExecutor(), nil
	case "json_to_csv":
		return NewJSONToCSVExecutor(), nil
	case "json":
		return NewJSONExecutor(), nil
	default:
		return nil, fmt.Errorf("no executor found for node type: %s", nodeType)
	}
}
