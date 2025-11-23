# Deployment Guide

This guide covers deploying Yantra to production environments.

## Pre-Deployment Checklist

- [ ] Set strong `JWT_SECRET` (min 32 characters)
- [ ] Configure production `DATABASE_URL` with SSL
- [ ] Set `MIGRATION_API_KEY` for controlled migrations
- [ ] Configure SMTP for email nodes
- [ ] Set up SSL/TLS certificates
- [ ] Configure reverse proxy (nginx/caddy)
- [ ] Set up monitoring and alerts
- [ ] Enable database backups
- [ ] Configure log aggregation
- [ ] Set proper `VITE_API_URL` for frontend

## Docker Deployment

### Production Docker Compose

Create a `docker-compose.prod.yml`:

```yaml
services:
  yantra-server:
    image: yantra-server:latest
    restart: unless-stopped
    environment:
      DATABASE_URL: ${DATABASE_URL}
      JWT_SECRET: ${JWT_SECRET}
      NODE_ENV: production
      MIGRATION_API_KEY: ${MIGRATION_API_KEY}
      SMTP_HOST: ${SMTP_HOST}
      SMTP_PORT: ${SMTP_PORT}
      SMTP_USER: ${SMTP_USER}
      SMTP_PASS: ${SMTP_PASS}
      SMTP_FROM: ${SMTP_FROM}
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "-O", "/dev/null", "http://localhost:3000/health"]
      interval: 30s
      timeout: 3s
      retries: 3

  yantra-web:
    image: yantra-web:latest
    restart: unless-stopped
    ports:
      - "80:80"
    depends_on:
      yantra-server:
        condition: service_healthy
```

### Build and Deploy

```bash
# Build images
docker-compose -f docker-compose.prod.yml build

# Run with production config
NODE_ENV=production docker-compose -f docker-compose.prod.yml up -d

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

## Kubernetes Deployment

### Backend Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: yantra-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: yantra-server
  template:
    metadata:
      labels:
        app: yantra-server
    spec:
      containers:
      - name: yantra-server
        image: yantra-server:latest
        ports:
        - containerPort: 3000
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: yantra-secrets
              key: database-url
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: yantra-secrets
              key: jwt-secret
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: yantra-server
spec:
  selector:
    app: yantra-server
  ports:
  - port: 3000
    targetPort: 3000
```

### Frontend Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: yantra-web
spec:
  replicas: 2
  selector:
    matchLabels:
      app: yantra-web
  template:
    metadata:
      labels:
        app: yantra-web
    spec:
      containers:
      - name: yantra-web
        image: yantra-web:latest
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: yantra-web
spec:
  selector:
    app: yantra-web
  ports:
  - port: 80
    targetPort: 80
```

### Ingress Configuration

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: yantra-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - yantra.example.com
    secretName: yantra-tls
  rules:
  - host: yantra.example.com
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: yantra-server
            port:
              number: 3000
      - path: /
        pathType: Prefix
        backend:
          service:
            name: yantra-web
            port:
              number: 80
```

## Reverse Proxy Setup

### Nginx Configuration

```nginx
upstream yantra_backend {
    server localhost:3000;
}

upstream yantra_frontend {
    server localhost:4700;
}

server {
    listen 80;
    server_name yantra.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yantra.example.com;

    ssl_certificate /etc/ssl/certs/yantra.crt;
    ssl_certificate_key /etc/ssl/private/yantra.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Backend API
    location /api/ {
        proxy_pass http://yantra_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Frontend
    location / {
        proxy_pass http://yantra_frontend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

### Caddy Configuration

```caddyfile
yantra.example.com {
    # Backend API
    handle /api/* {
        reverse_proxy localhost:3000
    }

    # Frontend
    handle {
        reverse_proxy localhost:4700
    }

    # Automatic HTTPS with Let's Encrypt
    tls admin@example.com
}
```

## Database Setup

### Production PostgreSQL

```bash
# Create database and user
sudo -u postgres psql

CREATE DATABASE yantra;
CREATE USER yantra WITH ENCRYPTED PASSWORD 'strong-password-here';
GRANT ALL PRIVILEGES ON DATABASE yantra TO yantra;

# Enable required extensions
\c yantra
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

### Connection Pooling

For production, use connection pooling (PgBouncer):

```ini
[databases]
yantra = host=localhost port=5432 dbname=yantra

[pgbouncer]
listen_addr = *
listen_port = 6432
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 20
```

Update `DATABASE_URL`:
```
postgresql://yantra:password@localhost:6432/yantra?sslmode=require
```

## Environment Variables

### Production .env Template

```bash
# Database
DATABASE_URL=postgresql://yantra:${DB_PASSWORD}@db.example.com:5432/yantra?sslmode=require

# Security
JWT_SECRET=${STRONG_JWT_SECRET}
MIGRATION_API_KEY=${MIGRATION_KEY}

# Server
NODE_ENV=production
PORT=3000

# Email (Required for email nodes)
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASS=${SENDGRID_API_KEY}
SMTP_FROM=noreply@example.com
```

### Generating Secrets

```bash
# JWT Secret
openssl rand -base64 48

# Migration API Key
openssl rand -hex 32
```

## Monitoring

### Health Checks

```bash
# Backend health
curl https://yantra.example.com/health

# Expected response
{"status":"healthy","timestamp":"2025-11-23T10:00:00Z"}
```

### Prometheus Metrics (Future Enhancement)

Add metrics endpoint for monitoring:
- Request latency
- Active executions
- Queue length
- Database connections
- Error rates

### Log Aggregation

**Using ELK Stack:**
```yaml
# Filebeat configuration
filebeat.inputs:
- type: container
  paths:
    - '/var/lib/docker/containers/*/*.log'
  processors:
    - add_docker_metadata: ~

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
```

**Using Loki:**
```yaml
# Promtail configuration
clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: yantra
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
```

## Backups

### Database Backups

```bash
# Daily backup script
#!/bin/bash
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
FILENAME="yantra_backup_${DATE}.sql.gz"

pg_dump -h db.example.com -U yantra yantra | gzip > ${BACKUP_DIR}/${FILENAME}

# Keep only last 30 days
find ${BACKUP_DIR} -name "yantra_backup_*.sql.gz" -mtime +30 -delete
```

**Automated with cron:**
```cron
# Daily at 2 AM
0 2 * * * /usr/local/bin/backup-yantra.sh
```

## SSL/TLS Certificates

### Let's Encrypt with Certbot

```bash
# Install certbot
sudo apt-get install certbot

# Get certificate
sudo certbot certonly --standalone -d yantra.example.com

# Auto-renewal
sudo certbot renew --dry-run
```

### Using with Nginx

```nginx
ssl_certificate /etc/letsencrypt/live/yantra.example.com/fullchain.pem;
ssl_certificate_key /etc/letsencrypt/live/yantra.example.com/privkey.pem;
```

## Scaling

### Horizontal Scaling

**Backend:**
- Multiple instances behind load balancer
- Shared PostgreSQL database
- River queue distributes work

**Database:**
- Primary-replica setup for read scaling
- Connection pooling (PgBouncer)

**Example Load Balancer Config:**
```nginx
upstream yantra_backends {
    least_conn;
    server backend1:3000;
    server backend2:3000;
    server backend3:3000;
}
```

## Troubleshooting

### Common Issues

**Problem**: High database connection count

**Solution**: Enable connection pooling with PgBouncer

**Problem**: Slow workflow execution

**Solution**:
- Check database query performance
- Add indexes on frequently queried fields
- Increase worker count

**Problem**: Email sending failures

**Solution**:
- Verify SMTP credentials
- Check outbox_messages table for errors
- Review outbox worker logs

### Health Check Endpoints

```bash
# Backend health
curl https://api.example.com/health

# Database connectivity
curl https://api.example.com/health/db
```

## Security Best Practices

1. Use strong, unique passwords (min 32 chars)
2. Enable SSL/TLS everywhere
3. Keep secrets in environment variables, never in code
4. Regularly rotate JWT secrets
5. Enable firewall rules
6. Use least-privilege database users
7. Enable audit logging
8. Keep dependencies updated
9. Use security headers (CORS, CSP, etc.)
10. Regular security audits

## Migration Process

1. Backup database
2. Set `MIGRATION_API_KEY`
3. Deploy new version
4. Run migrations via API
5. Verify application health
6. Monitor logs for errors

```bash
# Run migrations
curl -X POST \
  -H "X-Migration-Key: ${MIGRATION_API_KEY}" \
  https://api.example.com/api/migration/run
```

For more details, see [Configuration Guide](./CONFIGURATION.md).

