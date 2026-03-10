#!/bin/bash
# SSL Certificate Setup Script for Filo AI Receptionist
# Generates self-signed certificates for development/testing
# For production, replace with certificates from a trusted CA

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SSL_DIR="$PROJECT_ROOT/nginx/ssl"
N8N_SSL_DIR="$PROJECT_ROOT/n8n/ssl"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Check if OpenSSL is installed
check_openssl() {
    if ! command -v openssl &> /dev/null; then
        error "OpenSSL is not installed. Please install OpenSSL first."
        exit 1
    fi
    log "OpenSSL is available"
}

# Create SSL directories
create_directories() {
    log "Creating SSL directories..."
    mkdir -p "$SSL_DIR"
    mkdir -p "$N8N_SSL_DIR"
    success "SSL directories created"
}

# Generate SSL certificate
generate_certificate() {
    local domain="${1:-localhost}"
    local country="${2:-PH}"
    local state="${3:-Manila}"
    local city="${4:-Manila}"
    local org="${5:-Filo}"
    
    log "Generating SSL certificate for $domain..."
    
    # Check if certificate already exists
    if [[ -f "$SSL_DIR/filo.crt" && -f "$SSL_DIR/filo.key" ]]; then
        warning "SSL certificate already exists, skipping generation"
        return 0
    fi
    
    # Generate private key
    log "Generating private key..."
    openssl genrsa -out "$SSL_DIR/filo.key" 2048 2>/dev/null
    
    # Generate certificate signing request
    log "Generating certificate signing request..."
    openssl req -new -key "$SSL_DIR/filo.key" -out "$SSL_DIR/filo.csr" \
        -subj "/C=$country/ST=$state/L=$city/O=$org/CN=$domain" 2>/dev/null
    
    # Generate self-signed certificate
    log "Generating self-signed certificate..."
    openssl x509 -req -days 365 -in "$SSL_DIR/filo.csr" -signkey "$SSL_DIR/filo.key" \
        -out "$SSL_DIR/filo.crt" 2>/dev/null
    
    # Set proper permissions
    chmod 600 "$SSL_DIR/filo.key"
    chmod 644 "$SSL_DIR/filo.crt"
    
    # Clean up CSR
    rm -f "$SSL_DIR/filo.csr"
    
    success "SSL certificate generated successfully"
}

# Setup n8n SSL
setup_n8n_ssl() {
    log "Setting up n8n SSL certificates..."
    
    if [[ -f "$SSL_DIR/filo.crt" && -f "$SSL_DIR/filo.key" ]]; then
        cp "$SSL_DIR/filo.crt" "$N8N_SSL_DIR/n8n.crt"
        cp "$SSL_DIR/filo.key" "$N8N_SSL_DIR/n8n.key"
        chmod 600 "$N8N_SSL_DIR/n8n.key"
        chmod 644 "$N8N_SSL_DIR/n8n.crt"
        success "n8n SSL certificates configured"
    else
        error "SSL certificates not found, cannot configure n8n SSL"
        return 1
    fi
}

# Verify certificate
verify_certificate() {
    local cert_file="$1"
    
    if [[ ! -f "$cert_file" ]]; then
        error "Certificate file not found: $cert_file"
        return 1
    fi
    
    log "Verifying certificate..."
    openssl x509 -in "$cert_file" -text -noout | grep -E "(Subject:|Not Before|Not After)" || true
    success "Certificate verification completed"
}

# Show certificate information
show_certificate_info() {
    log "Certificate Information:"
    echo "=================================="
    
    if [[ -f "$SSL_DIR/filo.crt" ]]; then
        echo "Main Certificate (nginx):"
        openssl x509 -in "$SSL_DIR/filo.crt" -text -noout | grep -E "(Subject:|Issuer:|Not Before|Not After)" | sed 's/^/  /'
        echo
    fi
    
    if [[ -f "$N8N_SSL_DIR/n8n.crt" ]]; then
        echo "n8n Certificate:"
        openssl x509 -in "$N8N_SSL_DIR/n8n.crt" -text -noout | grep -E "(Subject:|Issuer:|Not Before|Not After)" | sed 's/^/  /'
        echo
    fi
    
    echo "Certificate files:"
    echo "  - Private Key: $SSL_DIR/filo.key"
    echo "  - Certificate: $SSL_DIR/filo.crt"
    echo "  - n8n Key:     $N8N_SSL_DIR/n8n.key"
    echo "  - n8n Cert:    $N8N_SSL_DIR/n8n.crt"
    echo "=================================="
}

# Import certificate to system (Linux/macOS)
import_certificate() {
    local cert_file="$1"
    
    if [[ ! -f "$cert_file" ]]; then
        error "Certificate file not found for import: $cert_file"
        return 1
    fi
    
    log "Importing certificate to system trust store..."
    
    # Linux (Ubuntu/Debian)
    if [[ -f "/etc/ssl/certs" ]]; then
        sudo cp "$cert_file" /usr/local/share/ca-certificates/filo.crt
        sudo update-ca-certificates
        success "Certificate imported to Linux system"
    
    # macOS
    elif [[ "$(uname -s)" == "Darwin" ]]; then
        sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$cert_file"
        success "Certificate imported to macOS system"
    
    # Windows (WSL)
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        warning "Windows certificate import not implemented in this script"
        warning "Please manually import $cert_file to Windows Certificate Manager"
    else
        warning "Unknown system, skipping certificate import"
    fi
}

# Create certificate for specific domain
create_domain_certificate() {
    local domain="$1"
    
    log "Creating certificate for domain: $domain"
    
    # Generate certificate for the specific domain
    generate_certificate "$domain"
    
    # Verify the certificate
    verify_certificate "$SSL_DIR/filo.crt"
    
    # Show certificate info
    show_certificate_info
    
    # Import to system
    import_certificate "$SSL_DIR/filo.crt"
    
    success "Domain certificate setup completed for $domain"
}

# Main function
main() {
    case "${1:-help}" in
        "generate"|"create")
            check_openssl
            create_directories
            generate_certificate "${2:-localhost}"
            setup_n8n_ssl
            verify_certificate "$SSL_DIR/filo.crt"
            show_certificate_info
            import_certificate "$SSL_DIR/filo.crt"
            ;;
        "domain")
            if [[ -z "${2:-}" ]]; then
                error "Domain name required. Usage: $0 domain example.com"
                exit 1
            fi
            check_openssl
            create_directories
            create_domain_certificate "$2"
            ;;
        "info")
            show_certificate_info
            ;;
        "import")
            if [[ -z "${2:-}" ]]; then
                error "Certificate file path required. Usage: $0 import /path/to/cert.crt"
                exit 1
            fi
            import_certificate "$2"
            ;;
        "help"|"-h"|"--help")
            cat << EOF
SSL Certificate Setup Script for Filo AI Receptionist

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    generate [domain]     Generate self-signed certificate (default: localhost)
    domain <domain>       Create certificate for specific domain
    info                  Show certificate information
    import <cert_file>    Import certificate to system trust store
    help                  Show this help message

Examples:
    $0 generate                    # Generate certificate for localhost
    $0 generate example.com        # Generate certificate for example.com
    $0 domain mybusiness.com       # Create certificate for mybusiness.com
    $0 info                        # Show certificate information
    $0 import /path/to/cert.crt    # Import certificate to system

Notes:
    - For production use, replace self-signed certificates with CA-signed certificates
    - Certificates are valid for 365 days
    - Private keys are generated with 2048-bit RSA
    - Certificates are stored in nginx/ssl/ and n8n/ssl/ directories

EOF
            ;;
        *)
            error "Unknown command: $1"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"