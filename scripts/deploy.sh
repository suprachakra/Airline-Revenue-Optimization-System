#!/bin/bash

# IAROS Deployment Script
# Handles deployment to staging and production environments

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
DEPLOYMENT_LOG="deployment_${TIMESTAMP}.log"

# Default values
ENVIRONMENT=""
VERSION=""
DRY_RUN=false
ROLLBACK=false
ROLLBACK_VERSION=""
SKIP_TESTS=false
FORCE=false

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}" | tee -a "$DEPLOYMENT_LOG"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" | tee -a "$DEPLOYMENT_LOG"
}

print_error() {
    echo -e "${RED}❌ $1${NC}" | tee -a "$DEPLOYMENT_LOG"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}" | tee -a "$DEPLOYMENT_LOG"
}

# Function to display usage
usage() {
    cat << EOF
IAROS Deployment Script

Usage: $0 [OPTIONS]

Options:
    -e, --environment ENV     Target environment (staging|production)
    -v, --version VERSION     Version to deploy (e.g., v3.0.1)
    -d, --dry-run            Perform dry run without actual deployment
    -r, --rollback           Rollback to previous version
    -rv, --rollback-version   Specific version to rollback to
    -st, --skip-tests        Skip running tests before deployment
    -f, --force              Force deployment without confirmations
    -h, --help               Show this help message

Examples:
    $0 -e staging -v v3.0.1              # Deploy v3.0.1 to staging
    $0 -e production -v v3.0.1 -d        # Dry run for production
    $0 -e production -r                   # Rollback production to previous version
    $0 -e staging -rv v3.0.0              # Rollback staging to specific version

EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -r|--rollback)
                ROLLBACK=true
                shift
                ;;
            -rv|--rollback-version)
                ROLLBACK_VERSION="$2"
                ROLLBACK=true
                shift 2
                ;;
            -st|--skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Function to validate inputs
validate_inputs() {
    if [[ -z "$ENVIRONMENT" ]]; then
        print_error "Environment must be specified"
        usage
        exit 1
    fi

    if [[ "$ENVIRONMENT" != "staging" && "$ENVIRONMENT" != "production" ]]; then
        print_error "Environment must be 'staging' or 'production'"
        exit 1
    fi

    if [[ "$ROLLBACK" == false && -z "$VERSION" ]]; then
        print_error "Version must be specified for deployment"
        usage
        exit 1
    fi

    if [[ "$ENVIRONMENT" == "production" && "$FORCE" == false ]]; then
        print_warning "Deploying to production environment"
        read -p "Are you sure? (yes/no): " -r
        if [[ ! $REPLY == "yes" ]]; then
            print_info "Deployment cancelled"
            exit 0
        fi
    fi
}

# Function to load environment configuration
load_environment_config() {
    local env_file="$PROJECT_ROOT/infrastructure/config/${ENVIRONMENT}.env"
    
    if [[ -f "$env_file" ]]; then
        source "$env_file"
        print_status "Loaded environment configuration for $ENVIRONMENT"
    else
        print_warning "Environment configuration file not found: $env_file"
    fi

    # Set environment-specific variables
    case $ENVIRONMENT in
        staging)
            KUBE_CONTEXT="iaros-staging"
            NAMESPACE="iaros-staging"
            DOMAIN="staging.iaros.com"
            REGISTRY="registry.iaros.com/staging"
            ;;
        production)
            KUBE_CONTEXT="iaros-production"
            NAMESPACE="iaros-production"
            DOMAIN="iaros.com"
            REGISTRY="registry.iaros.com/production"
            ;;
    esac
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking deployment prerequisites..."

    # Check required tools
    local required_tools=("kubectl" "helm" "docker" "git")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            print_error "$tool is required but not installed"
            exit 1
        fi
    done

    # Check kubectl context
    if ! kubectl config get-contexts | grep -q "$KUBE_CONTEXT"; then
        print_error "Kubernetes context '$KUBE_CONTEXT' not found"
        exit 1
    fi

    # Switch to correct context
    kubectl config use-context "$KUBE_CONTEXT"
    print_status "Switched to Kubernetes context: $KUBE_CONTEXT"

    # Check namespace
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        print_info "Creating namespace: $NAMESPACE"
        kubectl create namespace "$NAMESPACE"
    fi

    print_status "Prerequisites check completed"
}

# Function to run pre-deployment tests
run_tests() {
    if [[ "$SKIP_TESTS" == true ]]; then
        print_warning "Skipping tests as requested"
        return
    fi

    print_info "Running pre-deployment tests..."

    # Run unit tests
    if [[ -f "$PROJECT_ROOT/run-complete-testing.sh" ]]; then
        cd "$PROJECT_ROOT"
        ./run-complete-testing.sh --ci
        print_status "All tests passed"
    else
        print_warning "Test script not found, skipping tests"
    fi
}

# Function to build and push Docker images
build_and_push_images() {
    print_info "Building and pushing Docker images..."

    local services=(
        "api-gateway"
        "pricing-service"
        "forecasting-service"
        "offer-service"
        "order-service"
        "distribution-service"
        "ancillary-service"
        "user-management-service"
        "network-planning-service"
        "procurement-service"
        "promotion-service"
    )

    for service in "${services[@]}"; do
        local service_dir="$PROJECT_ROOT/services/${service}"
        if [[ -d "$service_dir" && -f "$service_dir/Dockerfile" ]]; then
            print_info "Building $service..."
            
            local image_tag="${REGISTRY}/${service}:${VERSION}"
            
            if [[ "$DRY_RUN" == false ]]; then
                docker build -t "$image_tag" "$service_dir"
                docker push "$image_tag"
            fi
            
            print_status "Built and pushed $service"
        else
            print_warning "Skipping $service (no Dockerfile found)"
        fi
    done

    # Build frontend
    local frontend_dir="$PROJECT_ROOT/frontend/web-portal"
    if [[ -d "$frontend_dir" && -f "$frontend_dir/Dockerfile" ]]; then
        print_info "Building web-portal..."
        
        local image_tag="${REGISTRY}/web-portal:${VERSION}"
        
        if [[ "$DRY_RUN" == false ]]; then
            docker build -t "$image_tag" "$frontend_dir"
            docker push "$image_tag"
        fi
        
        print_status "Built and pushed web-portal"
    fi
}

# Function to deploy using Helm
deploy_with_helm() {
    print_info "Deploying IAROS using Helm..."

    local helm_chart="$PROJECT_ROOT/infrastructure/helm/iaros"
    local values_file="$PROJECT_ROOT/infrastructure/helm/values-${ENVIRONMENT}.yaml"

    if [[ ! -d "$helm_chart" ]]; then
        print_error "Helm chart not found: $helm_chart"
        exit 1
    fi

    if [[ ! -f "$values_file" ]]; then
        print_error "Values file not found: $values_file"
        exit 1
    fi

    local helm_args=(
        "upgrade"
        "--install"
        "iaros"
        "$helm_chart"
        "--namespace" "$NAMESPACE"
        "--values" "$values_file"
        "--set" "image.tag=$VERSION"
        "--set" "environment=$ENVIRONMENT"
        "--wait"
        "--timeout" "10m"
    )

    if [[ "$DRY_RUN" == true ]]; then
        helm_args+=("--dry-run")
    fi

    if [[ "$DRY_RUN" == false ]]; then
        # Create backup of current deployment
        create_deployment_backup
    fi

    print_info "Running Helm deployment..."
    helm "${helm_args[@]}"

    if [[ "$DRY_RUN" == false ]]; then
        print_status "Helm deployment completed"
    else
        print_status "Dry run completed successfully"
    fi
}

# Function to create deployment backup
create_deployment_backup() {
    print_info "Creating deployment backup..."

    local backup_dir="$PROJECT_ROOT/backups/deployments/${ENVIRONMENT}/${TIMESTAMP}"
    mkdir -p "$backup_dir"

    # Backup current Helm release
    helm get values iaros -n "$NAMESPACE" > "$backup_dir/helm-values.yaml"
    helm get manifest iaros -n "$NAMESPACE" > "$backup_dir/helm-manifest.yaml"

    # Backup current deployment configurations
    kubectl get deployments -n "$NAMESPACE" -o yaml > "$backup_dir/deployments.yaml"
    kubectl get services -n "$NAMESPACE" -o yaml > "$backup_dir/services.yaml"
    kubectl get configmaps -n "$NAMESPACE" -o yaml > "$backup_dir/configmaps.yaml"
    kubectl get secrets -n "$NAMESPACE" -o yaml > "$backup_dir/secrets.yaml"

    print_status "Deployment backup created: $backup_dir"
}

# Function to perform rollback
perform_rollback() {
    print_info "Performing rollback..."

    if [[ -n "$ROLLBACK_VERSION" ]]; then
        print_info "Rolling back to version: $ROLLBACK_VERSION"
        helm upgrade iaros "$PROJECT_ROOT/infrastructure/helm/iaros" \
            --namespace "$NAMESPACE" \
            --set "image.tag=$ROLLBACK_VERSION" \
            --wait --timeout 10m
    else
        print_info "Rolling back to previous version"
        helm rollback iaros -n "$NAMESPACE"
    fi

    print_status "Rollback completed"
}

# Function to run health checks
run_health_checks() {
    print_info "Running post-deployment health checks..."

    # Wait for deployments to be ready
    kubectl wait --for=condition=available deployment --all -n "$NAMESPACE" --timeout=600s

    # Check service health endpoints
    local services=(
        "api-gateway:8080/health"
        "pricing-service:8081/health"
        "forecasting-service:8082/health"
        "offer-service:8083/health"
        "order-service:8084/health"
        "user-management-service:8085/health"
    )

    for service_endpoint in "${services[@]}"; do
        local service_name=$(echo "$service_endpoint" | cut -d':' -f1)
        local endpoint=$(echo "$service_endpoint" | cut -d':' -f2)
        
        print_info "Checking health of $service_name..."
        
        # Port forward and check health
        kubectl port-forward "service/$service_name" 8080:8080 -n "$NAMESPACE" &
        local port_forward_pid=$!
        
        sleep 5
        
        if curl -f "http://localhost:8080/health" &> /dev/null; then
            print_status "$service_name is healthy"
        else
            print_error "$service_name health check failed"
            kill $port_forward_pid 2>/dev/null || true
            exit 1
        fi
        
        kill $port_forward_pid 2>/dev/null || true
    done

    print_status "All health checks passed"
}

# Function to run smoke tests
run_smoke_tests() {
    print_info "Running smoke tests..."

    local smoke_test_script="$PROJECT_ROOT/tests/smoke/smoke-tests.sh"
    
    if [[ -f "$smoke_test_script" ]]; then
        "$smoke_test_script" --environment "$ENVIRONMENT"
        print_status "Smoke tests passed"
    else
        print_warning "Smoke test script not found, skipping"
    fi
}

# Function to update monitoring and alerting
update_monitoring() {
    print_info "Updating monitoring configuration..."

    # Update Prometheus rules
    local prometheus_rules="$PROJECT_ROOT/infrastructure/observability/prometheus/rules-${ENVIRONMENT}.yaml"
    if [[ -f "$prometheus_rules" ]]; then
        kubectl apply -f "$prometheus_rules" -n "$NAMESPACE"
        print_status "Prometheus rules updated"
    fi

    # Update Grafana dashboards
    local grafana_dashboards="$PROJECT_ROOT/infrastructure/observability/grafana/dashboards/"
    if [[ -d "$grafana_dashboards" ]]; then
        kubectl create configmap grafana-dashboards \
            --from-file="$grafana_dashboards" \
            --namespace="$NAMESPACE" \
            --dry-run=client -o yaml | kubectl apply -f -
        print_status "Grafana dashboards updated"
    fi
}

# Function to send notifications
send_notifications() {
    local status=$1
    local message=$2

    print_info "Sending deployment notifications..."

    # Slack notification (if webhook configured)
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        local color="good"
        if [[ "$status" != "success" ]]; then
            color="danger"
        fi

        curl -X POST -H 'Content-type: application/json' \
            --data "{
                \"attachments\": [{
                    \"color\": \"$color\",
                    \"title\": \"IAROS Deployment $status\",
                    \"text\": \"$message\",
                    \"fields\": [
                        {\"title\": \"Environment\", \"value\": \"$ENVIRONMENT\", \"short\": true},
                        {\"title\": \"Version\", \"value\": \"$VERSION\", \"short\": true},
                        {\"title\": \"Timestamp\", \"value\": \"$TIMESTAMP\", \"short\": true}
                    ]
                }]
            }" \
            "$SLACK_WEBHOOK_URL"
    fi

    # Email notification (if configured)
    if [[ -n "${EMAIL_NOTIFICATION:-}" ]] && command -v sendmail &> /dev/null; then
        echo "Subject: IAROS Deployment $status - $ENVIRONMENT
        
Deployment Details:
- Environment: $ENVIRONMENT
- Version: $VERSION
- Status: $status
- Message: $message
- Timestamp: $TIMESTAMP

View logs: $DEPLOYMENT_LOG
" | sendmail "$EMAIL_NOTIFICATION"
    fi

    print_status "Notifications sent"
}

# Function to cleanup old deployments
cleanup_old_deployments() {
    print_info "Cleaning up old deployment artifacts..."

    # Keep only last 5 Helm releases
    helm history iaros -n "$NAMESPACE" --max 100 | tail -n +6 | awk '{print $1}' | while read revision; do
        if [[ -n "$revision" && "$revision" != "REVISION" ]]; then
            helm delete iaros --revision "$revision" -n "$NAMESPACE" 2>/dev/null || true
        fi
    done

    # Cleanup old Docker images (keep last 3 versions)
    # This would typically be done by a separate cleanup job

    print_status "Cleanup completed"
}

# Main deployment function
main_deploy() {
    print_info "Starting IAROS deployment to $ENVIRONMENT"
    print_info "Version: $VERSION"
    print_info "Timestamp: $TIMESTAMP"

    if [[ "$DRY_RUN" == true ]]; then
        print_warning "This is a DRY RUN - no actual changes will be made"
    fi

    # Deployment steps
    load_environment_config
    check_prerequisites
    
    if [[ "$ROLLBACK" == true ]]; then
        perform_rollback
    else
        run_tests
        build_and_push_images
        deploy_with_helm
    fi

    if [[ "$DRY_RUN" == false ]]; then
        run_health_checks
        run_smoke_tests
        update_monitoring
        cleanup_old_deployments
        
        send_notifications "success" "Deployment completed successfully"
        print_status "Deployment to $ENVIRONMENT completed successfully!"
    else
        print_status "Dry run completed successfully!"
    fi
}

# Error handling
handle_error() {
    local exit_code=$?
    print_error "Deployment failed at line $1 with exit code $exit_code"
    
    if [[ "$DRY_RUN" == false ]]; then
        send_notifications "failed" "Deployment failed with exit code $exit_code"
        
        # Offer rollback on failure
        if [[ "$ROLLBACK" == false ]]; then
            read -p "Deployment failed. Do you want to rollback? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                print_info "Initiating rollback..."
                ROLLBACK=true
                perform_rollback
            fi
        fi
    fi
    
    exit $exit_code
}

trap 'handle_error $LINENO' ERR

# Main execution
main() {
    # Initialize logging
    echo "IAROS Deployment Log - $TIMESTAMP" > "$DEPLOYMENT_LOG"
    
    # Parse arguments and validate
    parse_args "$@"
    validate_inputs
    
    # Run main deployment
    main_deploy
}

# Execute main function
main "$@" 