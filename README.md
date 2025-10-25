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