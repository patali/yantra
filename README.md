# Yantra Server

## Database Migrations

Migrations are integrated directly into the application and run automatically on startup. Both River (job queue) and GORM (application models) migrations are handled programmatically.

## Migration Methods

### Method 1: Automatic on Startup (Default)

Migrations run automatically when you start the server:

```bash
go run cmd/server/main.go
```

On startup, the application will:
1. Run River migrations (creates job queue tables)
2. Run GORM migrations (creates application tables)
3. Start the server

**Benefits:**
- ✅ Zero configuration needed
- ✅ Always up-to-date database schema
- ✅ Works in all environments (dev, staging, production)

### Method 2: Migration API Endpoint

For production deployments where you want explicit control over migrations:

1. **Generate a secure API key:**
```bash
openssl rand -hex 32
```

2. **Set the key in your environment:**
```bash
MIGRATION_API_KEY=your-generated-key
```

3. **Use the migration endpoints:**

```bash
# Check migration status
curl -H "X-Migration-Key: your-secret-key" \
  http://localhost:3000/api/migration/status

# Run migrations manually
curl -X POST -H "X-Migration-Key: your-secret-key" \
  http://localhost:3000/api/migration/run
```

**Security Notes:**
- The endpoint is **disabled** if `MIGRATION_API_KEY` is not set
- Requires exact API key match in the `X-Migration-Key` header
- Safe to call multiple times (migrations are idempotent)

**Benefits:**
- ✅ Explicit control over when migrations run
- ✅ Useful for CI/CD pipelines
- ✅ Can run migrations before deploying new code

## Docker Setup

### Build and run with Docker Compose

```bash
docker-compose up --build
```

Migrations will run automatically on container startup.

### Environment Variables

Copy `.env.docker` to `.env` and update the `DATABASE_URL` to match your database configuration.

For Coolify deployments, set `DATABASE_URL` to use your PostgreSQL container name:
```
DATABASE_URL="postgresql://postgres:password@<postgres-container-name>:5432/yantra"
```

**Required for API migrations:**
```bash
# Generate a secure key
openssl rand -hex 32

# Add to .env
MIGRATION_API_KEY=your-generated-key
```

### Coolify Deployment

1. **Set environment variables in Coolify:**
   - `DATABASE_URL` - PostgreSQL connection string
   - `MIGRATION_API_KEY` - Secure random key for migrations

2. **Deploy your application**

3. **Run migrations after deployment:**
   ```bash
   curl -X POST -H "X-Migration-Key: your-key" \
     https://your-app.com/api/migration/run
   ```

You can automate step 3 using Coolify's post-deployment webhooks or scripts.

## Testing

Yantra has a comprehensive test suite including unit tests, integration tests, and regression tests.

### Running Tests

```bash
# Run all tests
go test ./...

# Run unit tests only (executors and services)
go test ./src/executors/... ./src/services/...

# Run integration tests
go test ./src/workflows/... -tags=integration

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test ./src/executors/ -run TestConditional

# Verbose output
go test -v ./...
```

### Test Organization

#### Unit Tests
- **Executors**: Individual test files per node type in `src/executors/`
  - `conditional_test.go` - Conditional node tests
  - `delay_test.go` - Delay node tests
  - `transform_test.go` - Transform node tests
  - `loop_test.go` - Loop node tests
  - `email_test.go` - Email node tests
  - `http_test.go` - HTTP node tests
  - `slack_test.go` - Slack node tests
  - And more...
- **Services**: Service-level tests in `src/services/`
  - `auth_service_test.go` - Authentication tests
  - `scheduler_service_test.go` - Scheduler tests

#### Integration Tests
- **Location**: `src/workflows/integration_test.go`
- **Purpose**: End-to-end workflow execution tests
- **Test Data**: `src/workflows/testdata/`
  - `workflows/` - Workflow JSON definitions
  - `fixtures/` - Test input data

#### Test Fixtures
Pre-built workflow definitions for regression testing:
- `simple_transform.json` - Basic data transformation
- `conditional_loop.json` - Conditional branching with loops
- `data_aggregation.json` - Complex data processing pipeline
- `error_handling.json` - Error propagation tests

### Test Strategy

See [TEST_STRATEGY.md](./TEST_STRATEGY.md) for detailed information about:
- Test organization and structure
- Integration test framework
- Regression test workflows
- Performance benchmarks
- CI/CD integration guidelines

### Test Database Setup

Integration tests require a PostgreSQL database. Set the connection string:

```bash
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/yantra_test?sslmode=disable"
```

Or use the default test database (same as above).

### Writing Tests

All new features and bug fixes should include:
1. **Unit tests** for individual node executors
2. **Integration tests** for multi-node workflows
3. **Regression tests** for critical workflows

See `src/executors/test_helpers.go` for testing utilities.