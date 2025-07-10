# IAROS Kubernetes Deployment Templates

This directory contains standardized Kubernetes deployment templates for IAROS services.

## Templates Available

### 1. service-deployment-template.yaml
**Purpose**: Standard template for backend microservices  
**Use for**: All Go-based microservices (pricing, forecasting, offer, etc.)

**Variables to replace**:
- `{{SERVICE_NAME}}` - Name of the service (e.g., pricing-service)
- `{{SERVICE_TIER}}` - Service tier: `gateway`, `backend`, `frontend`
- `{{CPU_LIMIT}}` - CPU limit (e.g., "500m", "1000m")
- `{{MEMORY_LIMIT}}` - Memory limit (e.g., "512Mi", "1Gi")
- `{{CPU_REQUEST}}` - CPU request (e.g., "250m", "500m")
- `{{MEMORY_REQUEST}}` - Memory request (e.g., "256Mi", "512Mi")

## Resource Guidelines

### Service Tier Resource Allocations

**Gateway Services** (API Gateway):
```yaml
resources:
  limits:
    cpu: "500m"
    memory: "512Mi"
  requests:
    cpu: "250m"
    memory: "256Mi"
```

**Backend Services** (Standard):
```yaml
resources:
  limits:
    cpu: "500m"
    memory: "512Mi"
  requests:
    cpu: "250m"
    memory: "256Mi"
```

**Backend Services** (High-Performance - Forecasting, Pricing):
```yaml
resources:
  limits:
    cpu: "600m"
    memory: "1Gi"
  requests:
    cpu: "300m"
    memory: "512Mi"
```

**Frontend Services** (Web Portal):
```yaml
resources:
  limits:
    cpu: "200m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
```

## Usage Instructions

### 1. Create New Service Deployment

```bash
# Copy template
cp infrastructure/k8s/templates/service-deployment-template.yaml \
   infrastructure/k8s/new-service-deployment.yaml

# Replace variables
sed -i 's/{{SERVICE_NAME}}/new-service/g' infrastructure/k8s/new-service-deployment.yaml
sed -i 's/{{SERVICE_TIER}}/backend/g' infrastructure/k8s/new-service-deployment.yaml
sed -i 's/{{CPU_LIMIT}}/500m/g' infrastructure/k8s/new-service-deployment.yaml
sed -i 's/{{MEMORY_LIMIT}}/512Mi/g' infrastructure/k8s/new-service-deployment.yaml
sed -i 's/{{CPU_REQUEST}}/250m/g' infrastructure/k8s/new-service-deployment.yaml
sed -i 's/{{MEMORY_REQUEST}}/256Mi/g' infrastructure/k8s/new-service-deployment.yaml
```

### 2. Deploy Service

```bash
kubectl apply -f infrastructure/k8s/new-service-deployment.yaml
```

### 3. Verify Deployment

```bash
kubectl get deployments -n iaros-prod
kubectl get services -n iaros-prod
kubectl get pods -n iaros-prod -l app=new-service
```

## Security Features

All templates include:
- **Non-root execution**: Services run as user 1000
- **Read-only filesystem**: Root filesystem is read-only
- **No privilege escalation**: Security hardening
- **Dropped capabilities**: All Linux capabilities dropped
- **Resource limits**: CPU and memory limits enforced

## Monitoring

All services include:
- **Health checks**: Readiness and liveness probes
- **Prometheus metrics**: ServiceMonitor for metrics collection
- **Structured logging**: JSON logging for centralized log aggregation

## Best Practices

1. **Always use templates** for new services
2. **Test locally** before applying to production
3. **Follow naming conventions**: `{service-name}-{component}`
4. **Use appropriate resource limits** based on service requirements
5. **Include monitoring** and health checks
6. **Apply security context** for all containers 