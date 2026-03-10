# Phase 3 — Deployment Implementation Plan

## Overview

Phase 3 focuses on making Filo production-ready through containerization, GPU acceleration, and multi-tenant deployment capabilities. This phase transforms Filo from a development setup into a robust, scalable business solution.

## Current State (Post-Phase 2)

After Phase 2, Filo will have:
- ✅ Intelligent features (RAG, intent classification, workflows)
- ✅ Stable foundation from Phase 1
- ✅ Local-first privacy architecture
- ✅ Business-specific knowledge and workflows

## Phase 3 Goals

Transform Filo into a production-ready deployment platform with:
- **Containerization**: Complete Docker-based deployment
- **GPU Acceleration**: NVIDIA GPU support for faster inference
- **Security**: HTTPS and secure microphone access
- **Multi-tenancy**: Support for multiple business personas
- **Scalability**: Production-grade deployment patterns

## Implementation Components

### 1. Complete Dockerization

**Purpose**: Containerize all components for consistent, reproducible deployments

**Implementation**:
```dockerfile
# Dockerfile for Go server + Kokoro
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o filo-server ./server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/filo-server /usr/local/bin/filo-server
COPY --from=builder /app/models/ /app/models/
COPY --from=builder /app/templates/ /app/templates/

EXPOSE 8080
CMD ["filo-server"]
```

**Integration Points**:
- Create Dockerfile for Go server
- Containerize Kokoro TTS server
- Update docker-compose.yml for production
- Add health checks and monitoring

**Files to Create/Modify**:
- `Dockerfile` - Go server containerization
- `docker-compose.prod.yml` - Production deployment
- `scripts/build.sh` - Build automation
- Update existing `docker-compose.yml`

### 2. GPU Acceleration (NVIDIA CDI via WSL2)

**Purpose**: Enable GPU acceleration for faster STT/TTS inference

**Implementation**:
```yaml
# docker-compose.yml with GPU support
services:
  speaches:
    image: ghcr.io/ten-framework/speaches:latest
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    environment:
      - CUDA_VISIBLE_DEVICES=0
      - WHISPER_MODEL=medium
```

**Integration Points**:
- Configure NVIDIA Container Runtime
- Update Speaches for GPU inference
- Optimize model loading for GPU
- Add GPU health monitoring

**Files to Create/Modify**:
- `docker-compose.gpu.yml` - GPU-enabled deployment
- `scripts/setup_gpu.sh` - GPU setup automation
- Update Speaches configuration

### 3. Nginx + HTTPS Setup

**Purpose**: Enable secure microphone access and production web serving

**Implementation**:
```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name your-business.com;
    
    ssl_certificate /etc/ssl/certs/filo.crt;
    ssl_certificate_key /etc/ssl/private/filo.key;
    
    location / {
        proxy_pass http://filo-server:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
    
    location /ws {
        proxy_pass http://filo-server:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

**Integration Points**:
- Set up SSL/TLS certificates
- Configure WebSocket proxying
- Add security headers
- Enable HTTP/2 for better performance

**Files to Create**:
- `nginx/nginx.conf` - Nginx configuration
- `scripts/setup_ssl.sh` - SSL certificate setup
- `docker-compose.ssl.yml` - SSL-enabled deployment

### 4. Multi-tenant Persona Configuration

**Purpose**: Support multiple business personas on a single deployment

**Implementation**:
```go
type TenantConfig struct {
    ID           string            `json:"id"`
    BusinessName string            `json:"business_name"`
    Language     string            `json:"language"`
    Hours        BusinessHours     `json:"hours"`
    Personas     map[string]Persona `json:"personas"`
    Knowledge    string            `json:"knowledge_path"`
}

type MultiTenantManager struct {
    tenants map[string]*TenantConfig
    mutex   sync.RWMutex
}
```

**Integration Points**:
- Tenant-aware configuration loading
- Dynamic persona switching
- Isolated knowledge bases
- Tenant-specific workflows

**Files to Create**:
- `server/multitenant.go` - Multi-tenant management
- `config/tenants/` - Tenant configuration directory
- Update persona system for multi-tenancy

## Implementation Timeline

### Week 1: Containerization Foundation
- [ ] Create Dockerfile for Go server
- [ ] Containerize Kokoro TTS
- [ ] Update docker-compose for production
- [ ] Add health checks and monitoring

### Week 2: GPU Acceleration
- [ ] Set up NVIDIA Container Runtime
- [ ] Configure GPU-enabled Speaches
- [ ] Optimize models for GPU inference
- [ ] Test GPU performance improvements

### Week 3: Security & HTTPS
- [ ] Set up SSL/TLS certificates
- [ ] Configure Nginx reverse proxy
- [ ] Enable WebSocket SSL support
- [ ] Add security headers and hardening

### Week 4: Multi-tenancy
- [ ] Implement tenant management system
- [ ] Create dynamic persona switching
- [ ] Set up isolated knowledge bases
- [ ] Test multi-tenant scenarios

## Technical Architecture

### Production Deployment Flow
```
Internet (HTTPS)
  │
  ▼
Nginx (SSL Termination)
  │
  ▼
Docker Network
  ├── Filo Server (Go) :8080
  ├── Kokoro TTS (Python) :5000
  ├── Speaches STT (GPU) :8000
  ├── Redis (Cache) :6379
  ├── Qdrant (RAG) :6333
  └── n8n (Workflows) :5678
```

### Multi-tenant Architecture
```
Filo Server
├── Tenant Router
│   ├── Tenant A: Restaurant
│   │   ├── Persona: Friendly
│   │   ├── Knowledge: Menu, Hours
│   │   └── Workflows: Reservations
│   ├── Tenant B: Clinic
│   │   ├── Persona: Professional
│   │   ├── Knowledge: Services, Hours
│   │   └── Workflows: Appointments
│   └── Tenant C: Store
│       ├── Persona: Helpful
│       ├── Knowledge: Products, Hours
│       └── Workflows: Orders
```

## Dependencies and Requirements

### New Dependencies
- **NVIDIA Container Runtime**: For GPU acceleration
- **Nginx**: For SSL termination and reverse proxying
- **SSL Certificates**: For HTTPS (Let's Encrypt or custom)
- **Docker Compose**: For multi-service orchestration

### Infrastructure Requirements
- **GPU Support**: NVIDIA GPU with CUDA support
- **SSL Certificates**: Valid SSL/TLS certificates
- **Domain Name**: For HTTPS and professional appearance
- **Storage**: Persistent storage for models and data

## Success Criteria

### Functional Requirements
- [ ] Complete containerization with single `docker-compose up`
- [ ] GPU acceleration reduces inference time by >50%
- [ ] HTTPS enables microphone access in production
- [ ] Multi-tenant system supports 10+ concurrent tenants

### Performance Requirements
- [ ] Container startup time <30 seconds
- [ ] GPU inference latency <200ms for STT/TTS
- [ ] SSL termination adds <10ms overhead
- [ ] Multi-tenant isolation with no cross-contamination

### Security Requirements
- [ ] HTTPS enforced for all external connections
- [ ] Secure WebSocket connections (wss://)
- [ ] Container security best practices implemented
- [ ] Multi-tenant data isolation guaranteed

### Operational Requirements
- [ ] Health checks for all services
- [ ] Log aggregation and monitoring
- [ ] Backup and recovery procedures
- [ ] Zero-downtime deployment capability

## Risk Mitigation

### Technical Risks
- **GPU Compatibility**: Test across different NVIDIA GPUs
- **SSL Complexity**: Use automated certificate management (Let's Encrypt)
- **Container Security**: Follow security best practices and regular updates
- **Multi-tenant Complexity**: Implement thorough isolation testing

### Operational Risks
- **Deployment Complexity**: Create comprehensive deployment documentation
- **Resource Management**: Monitor and optimize resource usage
- **Backup Strategy**: Implement automated backup procedures
- **Monitoring**: Set up comprehensive monitoring and alerting

## Configuration Management

### Environment Variables
```env
# Core Configuration
FILAMENT_MODE=production
LOG_LEVEL=info
ENABLE_METRICS=true

# SSL Configuration
SSL_CERT_PATH=/etc/ssl/certs/filo.crt
SSL_KEY_PATH=/etc/ssl/private/filo.key
ENABLE_HTTPS=true

# GPU Configuration
ENABLE_GPU=true
GPU_DEVICE=0
CUDA_VISIBLE_DEVICES=0

# Multi-tenancy
ENABLE_MULTI_TENANT=true
TENANT_CONFIG_PATH=/app/config/tenants
DEFAULT_TENANT=default
```

### Deployment Scripts
```bash
# scripts/deploy.sh
#!/bin/bash
set -e

echo "Building Filo containers..."
docker-compose -f docker-compose.prod.yml build

echo "Setting up SSL certificates..."
./scripts/setup_ssl.sh

echo "Starting production deployment..."
docker-compose -f docker-compose.prod.yml up -d

echo "Verifying deployment..."
./scripts/health_check.sh
```

## Next Steps

1. **Complete Phase 2**: Ensure intelligent features are stable
2. **Start with Containerization**: Build foundation for all other components
3. **Add GPU Support**: Enable performance improvements
4. **Implement Security**: Set up HTTPS and secure access
5. **Add Multi-tenancy**: Enable business scalability

## Dependencies on Previous Phases

- **Phase 1 Foundation**: Stable core functionality
- **Phase 2 Intelligence**: RAG, intent classification, workflows
- **Infrastructure**: Docker setup and service orchestration
- **Configuration**: Environment variable and persona systems

This phase transforms Filo into a production-ready, scalable business solution suitable for deployment across multiple businesses while maintaining high performance and security standards.