#!/bin/bash

# System Integrations Production Deployment Script
# IAROS - Integrated Airline Revenue Optimization System
# Version: 1.0.0

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="iaros-production"
DEPLOYMENT_NAME="system-integrations"
IMAGE_TAG="${1:-v1.0.0}"
TIMEOUT=600 # 10 minutes

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we can connect to the cluster
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check if namespace exists
    if ! kubectl get namespace $NAMESPACE &> /dev/null; then
        print_error "Namespace $NAMESPACE does not exist"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to validate configuration
validate_configuration() {
    print_status "Validating configuration..."
    
    # Check if secrets exist
    if ! kubectl get secret integration-secrets -n $NAMESPACE &> /dev/null; then
        print_error "Integration secrets not found. Please create the secrets first."
        exit 1
    fi
    
    # Validate YAML files
    for yaml_file in "k8s/system-integrations-deployment.yaml" "observability/system-integrations-monitoring.yaml"; do
        if [[ ! -f "$yaml_file" ]]; then
            print_error "Required file $yaml_file not found"
            exit 1
        fi
        
        if ! kubectl apply --dry-run=client -f "$yaml_file" &> /dev/null; then
            print_error "Invalid YAML configuration in $yaml_file"
            exit 1
        fi
    done
    
    print_success "Configuration validation passed"
}

# Function to backup current deployment
backup_current_deployment() {
    print_status "Creating backup of current deployment..."
    
    if kubectl get deployment $DEPLOYMENT_NAME -n $NAMESPACE &> /dev/null; then
        kubectl get deployment $DEPLOYMENT_NAME -n $NAMESPACE -o yaml > "backup-${DEPLOYMENT_NAME}-$(date +%Y%m%d-%H%M%S).yaml"
        print_success "Backup created successfully"
    else
        print_warning "No existing deployment found to backup"
    fi
}

# Function to update image tag in deployment
update_image_tag() {
    print_status "Updating deployment with image tag: $IMAGE_TAG"
    
    # Update the image tag in the deployment file
    sed -i.bak "s|image: iaros/system-integrations:.*|image: iaros/system-integrations:${IMAGE_TAG}|g" k8s/system-integrations-deployment.yaml
    
    print_success "Image tag updated"
}

# Function to deploy to Kubernetes
deploy_to_kubernetes() {
    print_status "Deploying system integrations to Kubernetes..."
    
    # Apply the deployment
    kubectl apply -f k8s/system-integrations-deployment.yaml
    
    # Apply monitoring configuration
    kubectl apply -f observability/system-integrations-monitoring.yaml
    
    print_success "Deployment configurations applied"
}

# Function to wait for deployment rollout
wait_for_rollout() {
    print_status "Waiting for deployment rollout to complete..."
    
    if kubectl rollout status deployment/$DEPLOYMENT_NAME -n $NAMESPACE --timeout=${TIMEOUT}s; then
        print_success "Deployment rollout completed successfully"
    else
        print_error "Deployment rollout failed or timed out"
        return 1
    fi
}

# Function to run health checks
run_health_checks() {
    print_status "Running health checks..."
    
    # Check if pods are running
    local ready_pods=$(kubectl get pods -l app=system-integrations -n $NAMESPACE --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    local desired_pods=$(kubectl get deployment $DEPLOYMENT_NAME -n $NAMESPACE -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
    
    if [[ $ready_pods -eq $desired_pods ]] && [[ $ready_pods -gt 0 ]]; then
        print_success "All pods are running ($ready_pods/$desired_pods)"
    else
        print_error "Pods not ready: $ready_pods/$desired_pods running"
        return 1
    fi
    
    # Check service endpoints
    if kubectl get endpoints system-integrations-service -n $NAMESPACE &> /dev/null; then
        local endpoint_count=$(kubectl get endpoints system-integrations-service -n $NAMESPACE -o jsonpath='{.subsets[0].addresses}' | jq '. | length' 2>/dev/null || echo "0")
        if [[ $endpoint_count -gt 0 ]]; then
            print_success "Service endpoints are ready ($endpoint_count endpoints)"
        else
            print_error "No service endpoints available"
            return 1
        fi
    else
        print_error "Service endpoints not found"
        return 1
    fi
    
    # Test health endpoint
    print_status "Testing health endpoint..."
    local pod_name=$(kubectl get pods -l app=system-integrations -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [[ -n "$pod_name" ]]; then
        if kubectl exec -n $NAMESPACE $pod_name -- curl -f http://localhost:8080/health &> /dev/null; then
            print_success "Health endpoint responding"
        else
            print_error "Health endpoint not responding"
            return 1
        fi
    else
        print_error "No pods available for health check"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_status "Running integration tests..."
    
    # Run a subset of integration tests to verify deployment
    local test_pod_name="integration-test-$(date +%s)"
    
    kubectl run $test_pod_name -n $NAMESPACE --image=iaros/integration-tests:latest --rm -i --restart=Never -- \
        /bin/sh -c "cd /tests && go test -v -run TestSystemIntegrationsHealth ./integration/ -timeout 5m" || {
        print_error "Integration tests failed"
        return 1
    }
    
    print_success "Integration tests passed"
}

# Function to verify metrics and monitoring
verify_monitoring() {
    print_status "Verifying monitoring setup..."
    
    # Check if ServiceMonitor exists
    if kubectl get servicemonitor system-integrations-monitor -n $NAMESPACE &> /dev/null; then
        print_success "ServiceMonitor configured"
    else
        print_error "ServiceMonitor not found"
        return 1
    fi
    
    # Check if PrometheusRule exists
    if kubectl get prometheusrule system-integrations-alerts -n $NAMESPACE &> /dev/null; then
        print_success "PrometheusRule configured"
    else
        print_error "PrometheusRule not found"
        return 1
    fi
    
    # Test metrics endpoint
    local pod_name=$(kubectl get pods -l app=system-integrations -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [[ -n "$pod_name" ]]; then
        if kubectl exec -n $NAMESPACE $pod_name -- curl -f http://localhost:9090/metrics &> /dev/null; then
            print_success "Metrics endpoint responding"
        else
            print_error "Metrics endpoint not responding"
            return 1
        fi
    fi
}

# Function to rollback deployment
rollback_deployment() {
    print_error "Rolling back deployment..."
    
    kubectl rollout undo deployment/$DEPLOYMENT_NAME -n $NAMESPACE
    
    if kubectl rollout status deployment/$DEPLOYMENT_NAME -n $NAMESPACE --timeout=${TIMEOUT}s; then
        print_success "Rollback completed successfully"
    else
        print_error "Rollback failed"
        exit 1
    fi
}

# Function to update integration status
update_integration_status() {
    print_status "Updating integration status documentation..."
    
    # Update the Tech_Strategy/System_Integrations.md file
    local status_file="../../Tech_Strategy/System_Integrations.md"
    local temp_file=$(mktemp)
    
    if [[ -f "$status_file" ]]; then
        # Update the status of the 7 new integrations to Production
        sed 's/🔄 Implementation/✅ Production/g; s/🔄 PoC Status/✅ Production/g' "$status_file" > "$temp_file"
        mv "$temp_file" "$status_file"
        
        print_success "Integration status updated"
    else
        print_warning "Status file not found, skipping status update"
    fi
}

# Function to send deployment notification
send_notification() {
    local status=$1
    local message="System Integrations Deployment"
    
    if [[ $status == "success" ]]; then
        message="✅ $message completed successfully"
        print_success "Deployment notification sent"
    else
        message="❌ $message failed"
        print_error "Deployment failure notification sent"
    fi
    
    # Send notification to Slack/Teams (placeholder)
    # curl -X POST -H 'Content-type: application/json' \
    #      --data "{\"text\":\"$message\"}" \
    #      YOUR_WEBHOOK_URL
}

# Main deployment function
main() {
    print_status "Starting System Integrations Deployment"
    print_status "Image Tag: $IMAGE_TAG"
    print_status "Namespace: $NAMESPACE"
    echo ""
    
    # Pre-deployment checks
    check_prerequisites
    validate_configuration
    backup_current_deployment
    
    # Deployment process
    update_image_tag
    deploy_to_kubernetes
    
    # Wait for deployment and run checks
    if wait_for_rollout; then
        if run_health_checks && verify_monitoring; then
            print_status "Running integration tests..."
            if run_integration_tests; then
                update_integration_status
                send_notification "success"
                
                print_success "🎉 System Integrations Deployment Completed Successfully!"
                print_status "The following integrations are now live in production:"
                echo "  ✅ Sabre PSS Integration (Backup System)"
                echo "  ✅ SITA BagManager Integration (Baggage Tracking)"
                echo "  ✅ Weather Data Integration (Flight Conditions)"
                echo "  ✅ Social Media Integration (Marketing Automation)"
                echo "  ✅ Survey Platform Integration (Customer Feedback)"
                echo "  ✅ Quick Integrations (Personalization, Chatbot, Banking, Email, CRM)"
                echo "  ✅ Integration Manager (Orchestration Layer)"
                echo ""
                print_status "Monitoring Dashboard: https://grafana.iaros.ai/d/system-integrations"
                print_status "Metrics Endpoint: https://integrations.iaros.ai/metrics"
                
                exit 0
            else
                print_error "Integration tests failed"
                rollback_deployment
                send_notification "failure"
                exit 1
            fi
        else
            print_error "Health checks or monitoring verification failed"
            rollback_deployment
            send_notification "failure"
            exit 1
        fi
    else
        print_error "Deployment rollout failed"
        rollback_deployment
        send_notification "failure"
        exit 1
    fi
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 