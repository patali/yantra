# Yantra Server

## Database Migrations

### Quick Start (Recommended)

Run all migrations automatically:

```bash
./migrate.sh
```

This script will:
1. Install River CLI if needed
2. Run River migrations
3. Build the GORM migration binary
4. Run GORM migrations

### Manual Migration Steps

If you prefer to run migrations manually:

#### 1. Run River migrations

```bash
go install github.com/riverqueue/river/cmd/river@latest
river migrate-up --database-url "$DATABASE_URL"
```

#### 2. Run GORM migrations

```bash
# Make sure DATABASE_URL is set in .env
go run cmd/migrate/main.go
```

## Docker Setup

### Build and run with Docker Compose

```bash
docker-compose up --build
```

### Running Migrations

You have **two options** for running migrations:

#### Option 1: Migration API Endpoint (Recommended)

After deploying your application, trigger migrations via the API:

1. **Generate a secure API key:**
```bash
openssl rand -hex 32
```

2. **Set the key in your environment:**
```bash
MIGRATION_API_KEY=your-generated-key
```

3. **Deploy your application and use the migration endpoints:**

```bash
# Check migration status
curl -H "X-Migration-Key: your-secret-key" \
  http://localhost:3000/api/migration/status

# Run migrations
curl -X POST -H "X-Migration-Key: your-secret-key" \
  http://localhost:3000/api/migration/run
```

**Security Notes:**
- The endpoint is **disabled** if `MIGRATION_API_KEY` is not set
- Requires exact API key match in the `X-Migration-Key` header
- Safe to call multiple times (migrations are idempotent)

**Benefits:**
- ✅ Simple to use with any deployment platform
- ✅ No performance impact on server startup
- ✅ Manual control over when migrations run
- ✅ Perfect for Coolify deployments

#### Option 2: Local Script

For local development:

```bash
./migrate.sh
```

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