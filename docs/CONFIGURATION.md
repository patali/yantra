# Configuration Guide

This guide covers all configuration options for Yantra.

## Environment Variables

### Backend Configuration

#### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://user:pass@host:5432/db?sslmode=disable` |
| `JWT_SECRET` | JWT signing secret (min 32 chars) | `your-secure-secret-key-here` |

#### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server HTTP port | `3000` |
| `NODE_ENV` | Environment mode | `development` |
| `MIGRATION_API_KEY` | API key for manual migrations | (disabled) |

#### Email Configuration

Required for email node functionality:

| Variable | Description | Default |
|----------|-------------|---------|
| `SMTP_HOST` | SMTP server hostname | - |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USER` | SMTP username | - |
| `SMTP_PASS` | SMTP password | - |
| `SMTP_FROM` | Default sender email | - |

### Frontend Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_API_URL` | Backend API URL | `http://localhost:3000` |

## Database Configuration

### Connection String Format

```
postgresql://[user]:[password]@[host]:[port]/[database]?[parameters]
```

**Example (Local Development):**
```
postgresql://yantra:yantra_dev_password@localhost:5432/yantra?sslmode=disable
```

**Example (Docker):**
```
postgresql://yantra:yantra_dev_password@host.docker.internal:5432/yantra?sslmode=disable
```

**Example (Production):**
```
postgresql://yantra:secure_password@db.example.com:5432/yantra?sslmode=require
```

### SSL/TLS Options

- `sslmode=disable` - No SSL (development only)
- `sslmode=require` - Require SSL connection
- `sslmode=verify-ca` - Verify certificate authority
- `sslmode=verify-full` - Full certificate verification

## JWT Configuration

### Generating a Secure Secret

```bash
# Generate a random 32-byte secret
openssl rand -base64 32

# Or generate a hex secret
openssl rand -hex 32
```

### Token Expiration

Default JWT token expiration: **24 hours**

To customize, modify the backend configuration in `backend/src/services/auth_service.go`.

## Migration Configuration

### Automatic Migrations (Default)

Migrations run automatically on server startup. No configuration needed.

### Manual Migrations (Production)

For production environments where you want control over migrations:

1. **Generate a migration API key:**

```bash
openssl rand -hex 32
```

2. **Set the environment variable:**

```bash
MIGRATION_API_KEY=your-generated-key
```

3. **Run migrations via API:**

```bash
curl -X POST \
  -H "X-Migration-Key: your-generated-key" \
  http://localhost:3000/api/migration/run
```

## Email Node Configuration

### Gmail Example

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password
SMTP_FROM=noreply@yourdomain.com
```

**Note**: Gmail requires an [App Password](https://support.google.com/accounts/answer/185833) for SMTP access.

### SendGrid Example

```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASS=your-sendgrid-api-key
SMTP_FROM=noreply@yourdomain.com
```

### AWS SES Example

```bash
SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_USER=your-smtp-username
SMTP_PASS=your-smtp-password
SMTP_FROM=verified@yourdomain.com
```

## Docker Configuration

### Environment File (.env)

Create a `.env` file in the project root:

```bash
# Copy example
cp env.example .env

# Edit with your values
nano .env
```

### Docker Compose Override

For custom configurations, create `docker-compose.override.yml`:

```yaml
services:
  yantra-server:
    environment:
      LOG_LEVEL: debug
    ports:
      - "8080:3000"
  
  yantra-web:
    ports:
      - "8000:80"
```

## Resource Limits

### Workflow Execution Limits

These are hardcoded for system protection:

| Limit | Value |
|-------|-------|
| Max execution duration | 30 minutes |
| Max loop iterations | 10,000 |
| Max data size | 10MB |
| Max nested loop depth | Configurable |

To modify these limits, edit `backend/src/executors/base.go`.

## Development vs Production

### Development Configuration

```bash
# .env for development
DATABASE_URL=postgresql://yantra:yantra_dev_password@localhost:5432/yantra?sslmode=disable
JWT_SECRET=dev-secret-key-min-32-characters-long
NODE_ENV=development
PORT=3000

# Optional for development
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
```

### Production Configuration

```bash
# .env for production
DATABASE_URL=postgresql://yantra:${STRONG_PASSWORD}@prod-db.example.com:5432/yantra?sslmode=require
JWT_SECRET=${STRONG_JWT_SECRET}
NODE_ENV=production
PORT=3000
MIGRATION_API_KEY=${MIGRATION_KEY}

# Email configuration
SMTP_HOST=smtp.provider.com
SMTP_PORT=587
SMTP_USER=${SMTP_USERNAME}
SMTP_PASS=${SMTP_PASSWORD}
SMTP_FROM=noreply@yourdomain.com
```

## Configuration Best Practices

1. **Never commit secrets** - Use `.env` files (gitignored by default)
2. **Use strong passwords** - Generate with `openssl rand -base64 32`
3. **Enable SSL in production** - Use `sslmode=require` for database
4. **Rotate secrets regularly** - Especially JWT secrets and API keys
5. **Use environment-specific configs** - Different `.env` files per environment
6. **Secure email credentials** - Use app passwords or API keys
7. **Monitor configuration** - Log configuration loading (without secrets)

## Troubleshooting

### Database Connection Issues

**Problem**: `connection refused` error

**Solutions**:
- Verify PostgreSQL is running
- Check host and port in DATABASE_URL
- For Docker, use `host.docker.internal` (macOS/Windows)
- For Docker on Linux, use host IP or `network_mode: host`

### JWT Errors

**Problem**: Invalid token errors

**Solutions**:
- Ensure JWT_SECRET is at least 32 characters
- Verify secret is consistent across restarts
- Check token expiration time

### Email Sending Fails

**Problem**: SMTP errors

**Solutions**:
- Verify SMTP credentials
- Check firewall/network allows port 587
- Use app passwords (not account password)
- Test with `telnet smtp.server.com 587`

For more help, see [Getting Started](./GETTING_STARTED.md) or [Deployment Guide](./DEPLOYMENT.md).

