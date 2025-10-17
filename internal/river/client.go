package river

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type Client struct {
	riverClient *river.Client[pgx.Tx]
	workers     *river.Workers
	pgxpool     *pgxpool.Pool
}

// NewClient creates a new River client
func NewClient(ctx context.Context, databaseURL string, engine WorkflowEngine) (*Client, error) {
	// Create pgx connection pool
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	// Note: River tables should be created by running setup_river.sql
	// or using River's CLI migration tools before starting the client

	// Create River workers
	workers := river.NewWorkers()

	// Register workflow execution worker
	workflowWorker := NewWorkflowExecutionWorker(engine)
	river.AddWorker(workers, workflowWorker)

	// Create River client
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
			"workflow":         {MaxWorkers: 5},
		},
		Workers: workers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create river client: %w", err)
	}

	log.Println("✅ River client created successfully")

	return &Client{
		riverClient: riverClient,
		workers:     workers,
		pgxpool:     pool,
	}, nil
}

// Start starts the River client and workers
func (c *Client) Start(ctx context.Context) error {
	if err := c.riverClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start river client: %w", err)
	}

	log.Println("✅ River workers started")
	return nil
}

// Stop gracefully stops the River client
func (c *Client) Stop(ctx context.Context) error {
	if err := c.riverClient.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop river client: %w", err)
	}

	c.pgxpool.Close()
	log.Println("✅ River client stopped")
	return nil
}

// GetClient returns the underlying River client
func (c *Client) GetClient() *river.Client[pgx.Tx] {
	return c.riverClient
}

// GetPgxPool returns the pgx connection pool
func (c *Client) GetPgxPool() *pgxpool.Pool {
	return c.pgxpool
}
