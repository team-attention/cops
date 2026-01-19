---
sidebar_position: 1
---

# Self-Hosted Deployment

Run C-Ops on your own infrastructure for full data control and customization.

## Overview

Self-hosting C-Ops requires deploying three components:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Your Infrastructure                          │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                      API Server                           │   │
│  │         (cops-api Docker container)                       │   │
│  └────────────────────────┬─────────────────────────────────┘   │
│                           │                                      │
│                           ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                       MongoDB                             │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Web Dashboard                          │   │
│  │         (cops-web Docker container)                       │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘

                    + Google OAuth Integration
```

## Prerequisites

Before starting, ensure you have:

- **Docker & Docker Compose** installed
- **Google Cloud Console** account for OAuth setup
- **Domain with HTTPS** (recommended for production)

## Step 1: Google OAuth Setup

C-Ops uses Google OAuth for authentication. You'll need to create OAuth credentials.

### Create OAuth Client ID

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Navigate to **APIs & Services** > **Credentials**
4. Click **Create Credentials** > **OAuth client ID**
5. Select **Web application** as the application type
6. Configure the following:
   - **Name**: `C-Ops`
   - **Authorized redirect URIs**:
     - `https://your-api-domain.com/auth/callback`
     - `http://localhost:8080/auth/callback` (for local testing)

### Configure OAuth Consent Screen

1. Go to **APIs & Services** > **OAuth consent screen**
2. Select **External** user type (or **Internal** for organization-only)
3. Fill in required fields:
   - **App name**: `C-Ops`
   - **User support email**: Your email
   - **Developer contact information**: Your email
4. Add scopes:
   - `email`
   - `profile`
   - `openid`
5. Add test users if in testing mode

### Save Credentials

Note down the following values:
- **Client ID**: `xxxxx.apps.googleusercontent.com`
- **Client Secret**: `GOCSPX-xxxxx`

## Step 2: Environment Configuration

Create a `.env` file with required configuration:

```bash
# MongoDB Configuration
MONGODB_URI=mongodb://mongodb:27017/cops
MONGODB_DATABASE=cops

# JWT Configuration (generate a secure random string)
JWT_SECRET_KEY=your-secure-random-string-at-least-32-chars

# Google OAuth Configuration
GOOGLE_CLIENT_ID=xxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxx

# Server Configuration
COPS_API_URL=https://your-api-domain.com
COPS_WEB_BASE_URL=https://your-dashboard-domain.com

# Optional: Log level
COPS_LOG_LEVEL=info
```

### Generate JWT Secret

Generate a secure JWT secret:

```bash
openssl rand -base64 32
```

## Step 3: Docker Compose Deployment

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  mongodb:
    image: mongo:7
    container_name: cops-mongodb
    volumes:
      - mongodb_data:/data/db
    environment:
      - MONGO_INITDB_DATABASE=cops
    networks:
      - cops-network
    restart: unless-stopped

  api:
    image: ghcr.io/team-attention/cops-api:latest
    container_name: cops-api
    ports:
      - "8080:8080"
    environment:
      - MONGODB_URI=${MONGODB_URI}
      - MONGODB_DATABASE=${MONGODB_DATABASE}
      - JWT_SECRET_KEY=${JWT_SECRET_KEY}
      - GOOGLE_CLIENT_ID=${GOOGLE_CLIENT_ID}
      - GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET}
      - COPS_WEB_BASE_URL=${COPS_WEB_BASE_URL}
      - COPS_LOG_LEVEL=${COPS_LOG_LEVEL:-info}
    depends_on:
      - mongodb
    networks:
      - cops-network
    restart: unless-stopped

  web:
    image: ghcr.io/team-attention/cops-web:latest
    container_name: cops-web
    ports:
      - "3000:3000"
    environment:
      - VITE_API_URL=${COPS_API_URL}
    depends_on:
      - api
    networks:
      - cops-network
    restart: unless-stopped

volumes:
  mongodb_data:

networks:
  cops-network:
    driver: bridge
```

### Start Services

```bash
# Load environment variables
export $(cat .env | xargs)

# Start all services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f api
```

## Step 4: Configure Reverse Proxy (Production)

For production deployments, use a reverse proxy with HTTPS.

### Nginx Example

```nginx
server {
    listen 443 ssl http2;
    server_name api.your-domain.com;

    ssl_certificate /etc/ssl/certs/your-cert.pem;
    ssl_certificate_key /etc/ssl/private/your-key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # gRPC support
        grpc_pass grpc://localhost:8080;
    }
}

server {
    listen 443 ssl http2;
    server_name dashboard.your-domain.com;

    ssl_certificate /etc/ssl/certs/your-cert.pem;
    ssl_certificate_key /etc/ssl/private/your-key.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

## Step 5: Configure CLI and Daemon

After deploying the server, configure your CLI and daemon to connect to your self-hosted instance.

### Set Environment Variable

Add to your shell profile (`~/.zshrc` or `~/.bashrc`):

```bash
export COPS_API_URL=https://your-api-domain.com
```

### Authenticate

```bash
# Reload shell or source profile
source ~/.zshrc

# Authenticate with your self-hosted instance
cops auth login
```

### Verify Connection

```bash
# Register a project
cops add .

# List projects to verify
cops list
```

## Troubleshooting

### Authentication Issues

**Symptom**: OAuth callback fails

**Solutions**:
1. Verify redirect URI in Google Console matches your API URL
2. Check `COPS_WEB_BASE_URL` environment variable
3. Ensure HTTPS is properly configured

### Connection Issues

**Symptom**: CLI cannot connect to API

**Solutions**:
1. Verify `COPS_API_URL` is set correctly:
   ```bash
   echo $COPS_API_URL
   ```
2. Test API health endpoint:
   ```bash
   curl https://your-api-domain.com/health/live
   ```
3. Check firewall rules allow traffic on required ports

### Database Issues

**Symptom**: API fails to start with MongoDB errors

**Solutions**:
1. Verify MongoDB is running:
   ```bash
   docker compose ps mongodb
   ```
2. Check MongoDB connection string format
3. Ensure database user has correct permissions

### gRPC/ConnectRPC Issues

**Symptom**: Daemon cannot send data

**Solutions**:
1. Ensure reverse proxy supports HTTP/2
2. Check gRPC-specific headers are passed through
3. Verify SSL certificates are valid

## Updating

To update your self-hosted deployment:

```bash
# Pull latest images
docker compose pull

# Restart with new images
docker compose up -d

# Verify health
docker compose ps
```

## Backup and Recovery

### MongoDB Backup

```bash
# Create backup
docker compose exec mongodb mongodump --out /data/backup

# Copy backup from container
docker cp cops-mongodb:/data/backup ./backup

# Restore from backup
docker compose exec mongodb mongorestore /data/backup
```

## Security Considerations

1. **Use HTTPS**: Always use TLS/SSL for production deployments
2. **Secure JWT Secret**: Use a strong, randomly generated secret
3. **MongoDB Authentication**: Enable authentication for MongoDB in production
4. **Network Isolation**: Use Docker networks to isolate services
5. **Regular Updates**: Keep Docker images and dependencies updated
