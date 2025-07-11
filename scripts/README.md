# IAROS Utility Scripts

This directory contains comprehensive utility scripts for IAROS development, deployment, and operations. These scripts are designed to streamline common tasks and provide enterprise-grade automation capabilities.

## 📚 Script Overview

### 🛠️ Development & Setup Scripts

#### `setup-dev-environment.sh`
**Purpose**: Automated development environment setup  
**Features**: 
- Installs all required dependencies (Node.js, Go, Python, Docker)
- Sets up development databases (PostgreSQL, Redis, MongoDB, Elasticsearch)
- Configures IDE settings and extensions
- Creates environment variables and configuration files
- Validates installation with health checks

```bash
# Full development setup
./setup-dev-environment.sh

# Check what would be installed
./setup-dev-environment.sh --help
```

#### `wait-for-services.sh`
**Purpose**: Service health checking and wait functionality  
**Features**:
- Waits for services to be ready before proceeding
- Supports multiple service types (HTTP, database, message queue)
- Configurable timeout and retry intervals
- Used by other scripts to ensure dependencies are available

```bash
# Wait for all services
./wait-for-services.sh

# Wait for specific services with timeout
./wait-for-services.sh --timeout 300 --services "api-gateway,database"
```

### 🚀 Deployment & Operations Scripts

#### `deploy.sh`
**Purpose**: Enterprise deployment automation  
**Features**:
- Multi-environment deployment (staging, production)
- Blue-green and canary deployment strategies
- Rollback capabilities with specific version targeting
- Health checks and smoke tests
- Notifications (Slack, email) and logging

```bash
# Deploy to staging
./deploy.sh --environment staging --version v3.0.1

# Production deployment with safety checks
./deploy.sh --environment production --version v3.0.1

# Rollback to previous version
./deploy.sh --environment production --rollback

# Dry run to test deployment
./deploy.sh --environment production --version v3.0.1 --dry-run
```

#### `backup.sh`
**Purpose**: Comprehensive backup solution  
**Features**:
- Database backups (PostgreSQL, Redis, MongoDB)
- Configuration and certificate backups
- Incremental and full backup modes
- Encryption and compression options
- S3 upload and retention management
- Restore capabilities

```bash
# Full backup with all options
./backup.sh

# Database-only backup
./backup.sh --db-only

# Backup with S3 upload
./backup.sh --s3-upload

# Restore from backup
./backup.sh --restore backup_20240115_143022
```

### 📊 Monitoring & Observability Scripts

#### `monitoring-setup.sh`
**Purpose**: Complete monitoring stack deployment  
**Features**:
- Prometheus, Grafana, Jaeger, ELK stack installation
- Custom IAROS dashboards and alerts
- Service discovery configuration
- Ingress and SSL setup
- Health monitoring and verification

```bash
# Install complete monitoring stack
./monitoring-setup.sh

# Install specific components
./monitoring-setup.sh --prometheus-only
./monitoring-setup.sh --grafana-only

# Setup for staging environment
./monitoring-setup.sh --environment staging
```

#### `log-analysis.sh`
**Purpose**: Advanced log analysis and reporting  
**Features**:
- Error pattern detection and analysis
- Performance metrics extraction
- Security event monitoring
- Real-time log monitoring
- HTML report generation

```bash
# Analyze last 24 hours of logs
./log-analysis.sh --time-range 24h

# Focus on errors only
./log-analysis.sh --analysis-type errors

# Real-time monitoring
./log-analysis.sh --real-time
```

### 🔧 Testing & Performance Scripts

#### `performance-test.sh`
**Purpose**: Comprehensive performance testing  
**Features**:
- Load, stress, endurance, spike, and volume testing
- k6 and Apache Bench integration
- Custom IAROS test scenarios
- Performance metrics collection
- HTML report generation with charts

```bash
# Run load test
./performance-test.sh --load-test --users 100 --duration 10m

# Run all performance tests
./performance-test.sh --all-tests

# Stress test with high load
./performance-test.sh --stress-test --users 500
```

### 🧹 Maintenance & Cleanup Scripts

#### `cleanup.sh`
**Purpose**: System cleanup and maintenance  
**Features**:
- Log file cleanup with retention policies
- Docker container/image/volume cleanup
- Cache and temporary file removal
- Node.js, Go, and Python artifact cleanup
- Dry-run mode for safety

```bash
# Clean temporary files
./cleanup.sh --temp

# Clean everything (with confirmation)
./cleanup.sh --all

# See what would be cleaned
./cleanup.sh --all --dry-run

# Force cleanup without confirmation
./cleanup.sh --all --force
```

## 🔧 Usage Patterns

### Daily Development Workflow
```bash
# 1. Start development environment
./setup-dev-environment.sh

# 2. Wait for services to be ready
./wait-for-services.sh

# 3. Run tests and performance checks
./performance-test.sh --load-test

# 4. Deploy to staging
./deploy.sh --environment staging --version v3.0.1

# 5. Clean up development artifacts
./cleanup.sh --temp --cache
```

### Production Deployment Workflow
```bash
# 1. Create backup before deployment
./backup.sh --all

# 2. Deploy with health checks
./deploy.sh --environment production --version v3.0.1

# 3. Monitor deployment
./log-analysis.sh --real-time

# 4. Rollback if needed
./deploy.sh --environment production --rollback
```

### Monitoring & Maintenance Workflow
```bash
# 1. Setup monitoring stack
./monitoring-setup.sh

# 2. Analyze logs for issues
./log-analysis.sh --analysis-type security

# 3. Run performance tests
./performance-test.sh --endurance-test

# 4. Clean up system
./cleanup.sh --logs --cache
```

## 🎯 Script Configuration

### Environment Variables
Scripts use the following environment variables:

```bash
# Common Configuration
export IAROS_ENV=production
export BASE_URL=https://api.iaros.com
export LOG_LEVEL=info

# Database Configuration
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=iaros
export DB_USER=iaros_user
export DB_PASSWORD=iaros_pass

# Monitoring Configuration
export PROMETHEUS_URL=http://localhost:9090
export GRAFANA_URL=http://localhost:3000
export JAEGER_URL=http://localhost:16686

# Deployment Configuration
export KUBERNETES_CONTEXT=iaros-production
export HELM_CHART_PATH=./infrastructure/helm/iaros
export DOCKER_REGISTRY=registry.iaros.com

# Backup Configuration
export BACKUP_DIR=/opt/iaros/backups
export S3_BUCKET=iaros-backups
export S3_REGION=us-east-1
export RETENTION_DAYS=30

# Notification Configuration
export SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
export EMAIL_NOTIFICATION=ops@iaros.com
```

### Configuration Files
Scripts look for configuration in:
- `./config/` - Script-specific configuration
- `./infrastructure/config/` - Infrastructure configuration
- `~/.iaros/config` - User-specific configuration

## 🔒 Security Considerations

### Permissions
```bash
# Set proper permissions for scripts
chmod +x scripts/*.sh

# Restrict access to sensitive scripts
chmod 700 scripts/backup.sh
chmod 700 scripts/deploy.sh
```

### Secrets Management
- Use environment variables for sensitive data
- Store secrets in secure key management systems
- Never commit secrets to version control
- Use GPG encryption for backup files

### Access Control
- Implement role-based access for production scripts
- Use audit logging for deployment scripts
- Require approval for production deployments
- Implement break-glass procedures for emergency access

## 📋 Prerequisites

### System Requirements
- **Operating System**: Linux, macOS, or Windows (WSL)
- **Memory**: 8GB RAM minimum, 16GB recommended
- **Storage**: 50GB free space for development environment
- **Network**: Internet access for downloads and updates

### Required Tools
- **Docker**: Container runtime and orchestration
- **Kubernetes**: Container orchestration (kubectl, helm)
- **Node.js**: JavaScript runtime (v18+)
- **Go**: Programming language (v1.19+)
- **Python**: Programming language (v3.9+)
- **Git**: Version control system

### Optional Tools
- **k6**: Performance testing (auto-installed)
- **Apache Bench**: Load testing (auto-installed)
- **jq**: JSON processing
- **curl**: HTTP client
- **gpg**: Encryption for backups

## 🛠️ Troubleshooting

### Common Issues

#### Permission Denied
```bash
# Fix script permissions
chmod +x scripts/*.sh

# Check file ownership
ls -la scripts/
```

#### Service Not Ready
```bash
# Check service status
./wait-for-services.sh --verbose

# Check logs
./log-analysis.sh --real-time
```

#### Database Connection Issues
```bash
# Test database connectivity
./scripts/test-db-connection.sh

# Check database logs
./log-analysis.sh --analysis-type errors
```

#### Performance Issues
```bash
# Run performance diagnostics
./performance-test.sh --load-test --users 10

# Check resource usage
./monitoring-setup.sh --prometheus-only
```

### Getting Help
```bash
# Show help for any script
./script-name.sh --help

# Check script version
./script-name.sh --version

# Enable verbose output
./script-name.sh --verbose
```

## 📈 Performance Metrics

### Script Execution Times
- **setup-dev-environment.sh**: 5-15 minutes (first run)
- **deploy.sh**: 3-8 minutes (depending on environment)
- **backup.sh**: 2-10 minutes (depending on data size)
- **monitoring-setup.sh**: 10-20 minutes (complete stack)
- **performance-test.sh**: 5-60 minutes (depending on test type)

### Resource Usage
- **CPU**: Low to moderate usage during execution
- **Memory**: 1-4GB during intensive operations
- **Network**: High during downloads and deployments
- **Storage**: Varies by operation (backups can be large)

## 🔄 Continuous Improvement

### Feedback and Contributions
- Report issues through the project's issue tracker
- Contribute improvements via pull requests
- Share usage patterns and best practices
- Suggest new script features and capabilities

### Maintenance Schedule
- **Weekly**: Review and update dependencies
- **Monthly**: Security updates and patches
- **Quarterly**: Performance optimization and new features
- **Annually**: Major version updates and architecture reviews

## 📚 References

### Documentation
- [IAROS Architecture Documentation](../docs/architecture/)
- [Deployment Guide](../docs/deployment/)
- [Monitoring Guide](../docs/monitoring/)
- [Security Best Practices](../docs/security/)

### External Resources
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Docker Documentation](https://docs.docker.com/)
- [Helm Documentation](https://helm.sh/docs/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)

---

**Note**: These scripts are designed for enterprise use and follow industry best practices for security, reliability, and maintainability. Always test in a non-production environment before using in production. 