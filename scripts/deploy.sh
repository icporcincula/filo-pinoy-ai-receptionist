#!/bin/bash
# Production deployment script for Filo AI Receptionist
# Usage: ./scripts/deploy.sh [environment]

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENVIRONMENT="${1:-prod}"
COMPOSE_FILE="docker-compose.prod.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if Docker/Podman is installed
    if command -v docker-compose &> /dev/null; then
        DOCKER_CMD="docker-compose"
    elif command -v podman-compose &> /dev/null; then
        DOCKER_CMD="podman-compose"
    else
        error "Neither docker-compose nor podman-compose found"
        exit 1
    fi
    
    # Check if NVIDIA GPU is available (for GPU acceleration)
    if command -v nvidia-smi &> /dev/null; then
        log "NVIDIA GPU detected"
        GPU_AVAILABLE=true
    else
        warning "NVIDIA GPU not detected - GPU acceleration will be disabled"
        GPU_AVAILABLE=false
    fi
    
    success "Prerequisites check completed"
}

# Setup SSL certificates
setup_ssl() {
    log "Setting up SSL certificates..."
    
    SSL_DIR="$PROJECT_ROOT/nginx/ssl"
    mkdir -p "$SSL_DIR"
    
    # Check if certificates already exist
    if [[ -f "$SSL_DIR/filo.crt" && -f "$SSL_DIR/filo.key" ]]; then
        warning "SSL certificates already exist, skipping generation"
        return 0
    fi
    
    # Generate self-signed certificates for development/testing
    log "Generating self-signed SSL certificates..."
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "$SSL_DIR/filo.key" \
        -out "$SSL_DIR/filo.crt" \
        -subj "/C=PH/ST=Manila/L=Manila/O=Filo/CN=localhost" \
        2>/dev/null
    
    # Set proper permissions
    chmod 600 "$SSL_DIR/filo.key"
    chmod 644 "$SSL_DIR/filo.crt"
    
    success "SSL certificates generated"
}

# Setup n8n SSL certificates
setup_n8n_ssl() {
    log "Setting up n8n SSL certificates..."
    
    N8N_SSL_DIR="$PROJECT_ROOT/n8n/ssl"
    mkdir -p "$N8N_SSL_DIR"
    
    # Copy certificates for n8n
    if [[ -f "$PROJECT_ROOT/nginx/ssl/filo.crt" && -f "$PROJECT_ROOT/nginx/ssl/filo.key" ]]; then
        cp "$PROJECT_ROOT/nginx/ssl/filo.crt" "$N8N_SSL_DIR/n8n.crt"
        cp "$PROJECT_ROOT/nginx/ssl/filo.key" "$N8N_SSL_DIR/n8n.key"
        chmod 600 "$N8N_SSL_DIR/n8n.key"
        chmod 644 "$N8N_SSL_DIR/n8n.crt"
        success "n8n SSL certificates configured"
    else
        warning "SSL certificates not found, n8n SSL will be disabled"
    fi
}

# Create environment file
setup_environment() {
    log "Setting up environment configuration..."
    
    ENV_FILE="$PROJECT_ROOT/.env"
    
    # Create .env file if it doesn't exist
    if [[ ! -f "$ENV_FILE" ]]; then
        log "Creating .env file from template..."
        if [[ -f "$PROJECT_ROOT/.env.example" ]]; then
            cp "$PROJECT_ROOT/.env.example" "$ENV_FILE"
        else
            warning ".env.example not found, creating basic .env file"
            cat > "$ENV_FILE" << EOF
# Filo AI Receptionist Environment Configuration

# Core Services
PORT=8080
SPEACHES_URL=http://speaches:8000
OLLAMA_URL=http://host.docker.internal:11434
KOKORO_URL=http://host.docker.internal:5000
REDIS_ADDR=redis:6379

# LLM Configuration
LLM_MODEL=llama3.1:8b
SYSTEM_PROMPT=You are a helpful, concise voice assistant for a Philippine business. Keep responses short and conversational.

# VAD Configuration
TEN_VAD_MODE=2

# Turn Detection
TURN_DETECTION_BASE_URL=http://localhost:8000/v1
TURN_DETECTION_API_KEY=your-turn-detection-api-key
TURN_DETECTION_MODEL=turn-detection-model
TURN_DETECTION_TEMPERATURE=0.1
TURN_DETECTION_TOP_P=0.1
TURN_DETECTION_THRESHOLD_MS=50

# Qdrant (RAG)
QDRANT_URL=http://qdrant:6333
QDRANT_API_KEY=your-qdrant-api-key
QDRANT_COLLECTION_NAME=knowledge_base

# n8n (Workflows)
N8N_URL=http://n8n:5678
N8N_USER=admin
N8N_PASSWORD=your-n8n-password
N8N_HOST=localhost

# Business Configuration
BUSINESS_NAME=Your Business Name
BUSINESS_HOURS=9:00-18:00
BUSINESS_LANGUAGE=en
PERSONALITY_FRIENDLY=true
PERSONALITY_FORMAL=false

# Google Calendar
GOOGLE_CALENDAR_ID=your-calendar-id
GOOGLE_CREDENTIALS_PATH=/app/config/google-credentials.json
EOF
        fi
    fi
    
    success "Environment configuration completed"
}

# Build and deploy
deploy() {
    log "Starting deployment with $DOCKER_CMD..."
    
    cd "$PROJECT_ROOT"
    
    # Pull latest images
    log "Pulling latest images..."
    $DOCKER_CMD -f "$COMPOSE_FILE" pull
    
    # Build Go server
    log "Building Go server..."
    $DOCKER_CMD -f "$COMPOSE_FILE" build go-server
    
    # Start services
    log "Starting services..."
    $DOCKER_CMD -f "$COMPOSE_FILE" up -d
    
    # Wait for services to be ready
    log "Waiting for services to be ready..."
    sleep 10
    
    # Check service health
    check_health
}

# Check service health
check_health() {
    log "Checking service health..."
    
    local services_healthy=true
    
    # Check Nginx
    if curl -s -f http://localhost/health > /dev/null 2>&1; then
        success "Nginx is healthy"
    else
        error "Nginx health check failed"
        services_healthy=false
    fi
    
    # Check Go server
    if curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
        success "Go server is healthy"
    else
        error "Go server health check failed"
        services_healthy=false
    fi
    
    # Check Redis
    if $DOCKER_CMD -f "$COMPOSE_FILE" exec redis redis-cli ping > /dev/null 2>&1; then
        success "Redis is healthy"
    else
        error "Redis health check failed"
        services_healthy=false
    fi
    
    # Check Qdrant
    if curl -s -f http://localhost:6333/healthz > /dev/null 2>&1; then
        success "Qdrant is healthy"
    else
        error "Qdrant health check failed"
        services_healthy=false
    fi
    
    # Check Speaches
    if curl -s -f http://localhost:8000/health > /dev/null 2>&1; then
        success "Speaches is healthy"
    else
        error "Speaches health check failed"
        services_healthy=false
    fi
    
    # Check n8n
    if curl -s -f http://localhost:5678/healthz > /dev/null 2>&1; then
        success "n8n is healthy"
    else
        error "n8n health check failed"
        services_healthy=false
    fi
    
    if [[ "$services_healthy" == "true" ]]; then
        success "All services are healthy!"
        log "Filo AI Receptionist is now running at:"
        log "  - Web Interface: https://localhost"
        log "  - Go Server: http://localhost:8080"
        log "  - n8n Workflows: https://localhost:5678"
        log "  - Qdrant: http://localhost:6333"
    else
        error "Some services are not healthy. Check logs with: $DOCKER_CMD -f $COMPOSE_FILE logs"
        exit 1
    fi
}

# Show logs
show_logs() {
    log "Showing service logs (press Ctrl+C to exit)..."
    $DOCKER_CMD -f "$COMPOSE_FILE" logs -f
}

# Cleanup
cleanup() {
    log "Cleaning up old images and containers..."
    $DOCKER_CMD -f "$COMPOSE_FILE" down
    docker image prune -f 2>/dev/null || podman image prune -f 2>/dev/null || true
    success "Cleanup completed"
}

# Show help
show_help() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

Commands:
    deploy      Deploy Filo AI Receptionist (default)
    logs        Show service logs
    cleanup     Stop services and cleanup
    help        Show this help message

Options:
    -h, --help  Show this help message

Examples:
    $0                  # Deploy with default settings
    $0 deploy           # Explicitly deploy
    $0 logs             # Show logs
    $0 cleanup          # Stop and cleanup

Environment:
    Set environment variables in .env file before deployment.
    Use .env.example as a template.

EOF
}

# Main execution
main() {
    case "${1:-deploy}" in
        "deploy")
            check_prerequisites
            setup_environment
            setup_ssl
            setup_n8n_ssl
            deploy
            ;;
        "logs")
            cd "$PROJECT_ROOT"
            show_logs
            ;;
        "cleanup")
            cd "$PROJECT_ROOT"
            cleanup
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"