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
	case NodeTypeJSONArray:
		return NewJsonArrayTriggerExecutor(), nil
	case NodeTypeConditional:
		return NewConditionalExecutor(), nil
	case NodeTypeTransform:
		return NewTransformExecutor(), nil
	case NodeTypeDelay:
		return NewDelayExecutor(), nil
	case NodeTypeSleep:
		return NewSleepExecutor(), nil
	case NodeTypeEmail:
		return NewEmailExecutor(f.db, f.emailService), nil
	case NodeTypeHTTP:
		return NewHTTPExecutor(f.httpClient), nil
	case NodeTypeSlack:
		return NewSlackExecutor(f.httpClient), nil
	case NodeTypeLoop:
		return NewLoopExecutor(), nil
	case NodeTypeLoopAccumulator:
		return NewLoopAccumulatorExecutor(), nil
	case NodeTypeJSONToCSV:
		return NewJSONToCSVExecutor(), nil
	case NodeTypeJSON:
		return NewJSONExecutor(), nil
	default:
		return nil, fmt.Errorf("no executor found for node type: %s", nodeType)
	}
}
