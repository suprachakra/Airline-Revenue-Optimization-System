#!/bin/bash
# start_all.sh - Enterprise-grade IAROS service launcher with comprehensive monitoring
# Author: IAROS Infrastructure Team
# Version: 2.0.0

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LOG_DIR="${PROJECT_ROOT}/logs"
CONFIG_DIR="${PROJECT_ROOT}/infrastructure/config"
COMPOSE_FILE="${PROJECT_ROOT}/infrastructure/ci-cd/docker-compose.yml"
PID_FILE="${LOG_DIR}/iaros_services.pid"
HEALTH_CHECK_TIMEOUT=300
STARTUP_TIMEOUT=600

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/startup.log"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/startup.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/startup.log"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/startup.log"
}

# Error handling
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        log_error "IAROS startup failed with exit code $exit_code"
        log_info "Cleaning up partially started services..."
        docker-compose -f "$COMPOSE_FILE" down --remove-orphans 2>/dev/null || true
    fi
    exit $exit_code
}

trap cleanup EXIT

# Validation functions
validate_environment() {
    log_info "Validating environment prerequisites..."
    
    # Check required commands
    local required_commands=("docker" "docker-compose" "curl" "jq")
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" &> /dev/null; then
            log_error "Required command '$cmd' not found. Please install it."
            exit 1
        fi
    done
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running. Please start Docker."
        exit 1
    fi
    
    # Check available resources
    local available_memory=$(free -m | awk 'NR==2{printf "%.0f", $7}')
    if [[ $available_memory -lt 4096 ]]; then
        log_warn "Available memory ($available_memory MB) is less than recommended minimum (4GB)"
    fi
    
    # Check disk space
    local available_disk=$(df "$PROJECT_ROOT" | awk 'NR==2 {print $4}')
    if [[ $available_disk -lt 10485760 ]]; then # 10GB in KB
        log_warn "Available disk space is less than recommended minimum (10GB)"
    fi
    
    log_success "Environment validation completed"
}

validate_configuration() {
    log_info "Validating configuration files..."
    
    # Check essential configuration files
    local config_files=(
        "$CONFIG_DIR/config.sample.yaml"
        "$CONFIG_DIR/dev.env.example"
        "$COMPOSE_FILE"
    )
    
    for config_file in "${config_files[@]}"; do
        if [[ ! -f "$config_file" ]]; then
            log_error "Required configuration file not found: $config_file"
            exit 1
        fi
    done
    
    # Validate docker-compose file
    if ! docker-compose -f "$COMPOSE_FILE" config &> /dev/null; then
        log_error "Invalid docker-compose configuration"
        exit 1
    fi
    
    log_success "Configuration validation completed"
}

# Service management functions
check_services_status() {
    log_info "Checking existing services status..."
    
    if docker-compose -f "$COMPOSE_FILE" ps --services --filter "status=running" | grep -q .; then
        log_warn "Some services are already running"
        docker-compose -f "$COMPOSE_FILE" ps
        read -p "Do you want to restart all services? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Stopping existing services..."
            docker-compose -f "$COMPOSE_FILE" down --remove-orphans
        else
            log_info "Skipping service restart"
            return 1
        fi
    fi
    return 0
}

start_services() {
    log_info "Starting IAROS services with docker-compose..."
    
    # Create necessary directories
    mkdir -p "$LOG_DIR"
    
    # Start services in detached mode
    if ! docker-compose -f "$COMPOSE_FILE" up -d --build; then
        log_error "Failed to start services"
        exit 1
    fi
    
    # Store process info
    echo "$$" > "$PID_FILE"
    echo "$(date)" >> "$PID_FILE"
    
    log_success "Services started successfully"
}

wait_for_services() {
    log_info "Waiting for services to become healthy..."
    
    local start_time=$(date +%s)
    local timeout=$HEALTH_CHECK_TIMEOUT
    
    # Define core services to check
    local core_services=(
        "api_gateway"
        "user_management_service"
        "pricing_service"
        "booking_service"
    )
    
    for service in "${core_services[@]}"; do
        log_info "Waiting for $service to be healthy..."
        
        local service_start_time=$(date +%s)
        while true; do
            local current_time=$(date +%s)
            local elapsed=$((current_time - service_start_time))
            
            if [[ $elapsed -gt $timeout ]]; then
                log_error "Service $service failed to become healthy within $timeout seconds"
                return 1
            fi
            
            # Check if service is healthy
            local health_status=$(docker-compose -f "$COMPOSE_FILE" ps --format "table {{.Name}}\t{{.Status}}" | grep "$service" | awk '{print $2}' || echo "unknown")
            
            if [[ "$health_status" == *"healthy"* ]] || [[ "$health_status" == *"running"* ]]; then
                log_success "Service $service is healthy"
                break
            fi
            
            echo -n "."
            sleep 5
        done
    done
    
    local total_time=$(($(date +%s) - start_time))
    log_success "All core services are healthy (took ${total_time}s)"
}

perform_health_checks() {
    log_info "Performing comprehensive health checks..."
    
    # API Gateway health check
    local api_url="http://localhost:8080/health"
    if curl -sf "$api_url" &> /dev/null; then
        log_success "API Gateway health check passed"
    else
        log_warn "API Gateway health check failed - service may still be starting"
    fi
    
    # Database connectivity check
    if docker-compose -f "$COMPOSE_FILE" exec -T postgres pg_isready &> /dev/null; then
        log_success "Database connectivity check passed"
    else
        log_warn "Database connectivity check failed"
    fi
    
    # Redis connectivity check
    if docker-compose -f "$COMPOSE_FILE" exec -T redis redis-cli ping | grep -q "PONG"; then
        log_success "Redis connectivity check passed"
    else
        log_warn "Redis connectivity check failed"
    fi
}

display_service_info() {
    log_info "IAROS Service Information:"
    echo "=================================="
    
    # Service status
    docker-compose -f "$COMPOSE_FILE" ps
    
    echo ""
    echo "Key URLs:"
    echo "  - API Gateway: http://localhost:8080"
    echo "  - Web Portal: http://localhost:3000"
    echo "  - Grafana: http://localhost:3001"
    echo "  - Swagger UI: http://localhost:8080/swagger"
    
    echo ""
    echo "Useful Commands:"
    echo "  - View logs: docker-compose -f $COMPOSE_FILE logs -f [service]"
    echo "  - Stop services: docker-compose -f $COMPOSE_FILE down"
    echo "  - Restart service: docker-compose -f $COMPOSE_FILE restart [service]"
    
    echo ""
    echo "Log files location: $LOG_DIR"
}

# Main execution
main() {
    log_info "Starting IAROS Airline Revenue Optimization System"
    log_info "=================================================="
    
    # Pre-startup validation
    validate_environment
    validate_configuration
    
    # Service management
    if ! check_services_status; then
        log_info "Exiting as requested by user"
        exit 0
    fi
    
    # Start services
    start_services
    
    # Wait for services to be ready
    wait_for_services
    
    # Perform health checks
    perform_health_checks
    
    # Display information
    display_service_info
    
    log_success "IAROS startup completed successfully!"
    log_info "System is ready for use. Monitor logs for any issues."
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
