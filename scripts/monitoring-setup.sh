#!/bin/bash

# IAROS Monitoring Setup Script
# Configures comprehensive monitoring stack: Prometheus, Grafana, Jaeger, ELK

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
MONITORING_NAMESPACE="iaros-monitoring"
ENVIRONMENT="${ENVIRONMENT:-production}"

# Component flags
INSTALL_PROMETHEUS=true
INSTALL_GRAFANA=true
INSTALL_JAEGER=true
INSTALL_ELK=true
INSTALL_ALERTMANAGER=true
SETUP_DASHBOARDS=true
SETUP_ALERTS=true

# Function to print status messages
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Function to display usage
usage() {
    cat << EOF
IAROS Monitoring Setup Script

Usage: $0 [OPTIONS]

Options:
    --prometheus-only       Install only Prometheus
    --grafana-only          Install only Grafana
    --jaeger-only           Install only Jaeger
    --elk-only              Install only ELK stack
    --no-dashboards         Skip dashboard setup
    --no-alerts             Skip alert setup
    --namespace NAME        Kubernetes namespace (default: iaros-monitoring)
    --environment ENV       Environment (staging|production)
    -h, --help              Show this help message

Examples:
    $0                              # Full monitoring stack
    $0 --prometheus-only            # Prometheus only
    $0 --environment staging        # Setup for staging environment

EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --prometheus-only)
                INSTALL_GRAFANA=false
                INSTALL_JAEGER=false
                INSTALL_ELK=false
                shift
                ;;
            --grafana-only)
                INSTALL_PROMETHEUS=false
                INSTALL_JAEGER=false
                INSTALL_ELK=false
                shift
                ;;
            --jaeger-only)
                INSTALL_PROMETHEUS=false
                INSTALL_GRAFANA=false
                INSTALL_ELK=false
                shift
                ;;
            --elk-only)
                INSTALL_PROMETHEUS=false
                INSTALL_GRAFANA=false
                INSTALL_JAEGER=false
                shift
                ;;
            --no-dashboards)
                SETUP_DASHBOARDS=false
                shift
                ;;
            --no-alerts)
                SETUP_ALERTS=false
                shift
                ;;
            --namespace)
                MONITORING_NAMESPACE="$2"
                shift 2
                ;;
            --environment)
                ENVIRONMENT="$2"
                shift 2
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

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."

    # Check required tools
    local required_tools=("kubectl" "helm")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            print_error "$tool is required but not installed"
            exit 1
        fi
    done

    # Check Kubernetes connection
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    # Create monitoring namespace
    kubectl create namespace "$MONITORING_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Add Helm repositories
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo add grafana https://grafana.github.io/helm-charts
    helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
    helm repo add elastic https://helm.elastic.co
    helm repo update

    print_status "Prerequisites check completed"
}

# Function to install Prometheus
install_prometheus() {
    if [[ "$INSTALL_PROMETHEUS" != true ]]; then
        return
    fi

    print_info "Installing Prometheus..."

    # Create Prometheus values file
    cat > /tmp/prometheus-values.yaml << EOF
server:
  persistentVolume:
    enabled: true
    size: 20Gi
    storageClass: fast-ssd
  retention: "30d"
  resources:
    requests:
      cpu: 500m
      memory: 2Gi
    limits:
      cpu: 2000m
      memory: 4Gi

alertmanager:
  enabled: true
  persistentVolume:
    enabled: true
    size: 5Gi
    storageClass: fast-ssd
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi

pushgateway:
  enabled: true
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi

nodeExporter:
  enabled: true
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi

kubeStateMetrics:
  enabled: true
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi

serverFiles:
  prometheus.yml:
    global:
      scrape_interval: 15s
      evaluation_interval: 15s
      external_labels:
        cluster: iaros-${ENVIRONMENT}
        environment: ${ENVIRONMENT}
    
    rule_files:
      - /etc/prometheus/rules/*.yml
    
    scrape_configs:
      - job_name: 'prometheus'
        static_configs:
          - targets: ['localhost:9090']
      
      - job_name: 'iaros-api-gateway'
        kubernetes_sd_configs:
          - role: endpoints
            namespaces:
              names:
                - iaros-${ENVIRONMENT}
        relabel_configs:
          - source_labels: [__meta_kubernetes_service_name]
            action: keep
            regex: api-gateway
          - source_labels: [__meta_kubernetes_endpoint_port_name]
            action: keep
            regex: metrics
      
      - job_name: 'iaros-services'
        kubernetes_sd_configs:
          - role: endpoints
            namespaces:
              names:
                - iaros-${ENVIRONMENT}
        relabel_configs:
          - source_labels: [__meta_kubernetes_service_label_app_kubernetes_io_name]
            action: keep
            regex: (pricing|forecasting|offer|order|distribution|ancillary|user-management|network-planning|procurement|promotion)-service
          - source_labels: [__meta_kubernetes_endpoint_port_name]
            action: keep
            regex: metrics
      
      - job_name: 'iaros-databases'
        static_configs:
          - targets: ['postgres-exporter:9187', 'redis-exporter:9121', 'mongodb-exporter:9216']
      
      - job_name: 'kubernetes-pods'
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
            action: keep
            regex: true
          - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
            action: replace
            target_label: __metrics_path__
            regex: (.+)
          - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
            action: replace
            regex: ([^:]+)(?::\d+)?;(\d+)
            replacement: \${1}:\${2}
            target_label: __address__
EOF

    # Install Prometheus using Helm
    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace "$MONITORING_NAMESPACE" \
        --values /tmp/prometheus-values.yaml \
        --wait --timeout 10m

    print_status "Prometheus installed successfully"
}

# Function to install Grafana
install_grafana() {
    if [[ "$INSTALL_GRAFANA" != true ]]; then
        return
    fi

    print_info "Installing Grafana..."

    # Create Grafana values file
    cat > /tmp/grafana-values.yaml << EOF
adminUser: admin
adminPassword: iaros-grafana-admin-${ENVIRONMENT}

persistence:
  enabled: true
  size: 10Gi
  storageClassName: fast-ssd

resources:
  requests:
    cpu: 250m
    memory: 512Mi
  limits:
    cpu: 1000m
    memory: 1Gi

service:
  type: ClusterIP
  port: 3000

ingress:
  enabled: true
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - grafana-${ENVIRONMENT}.iaros.com
  tls:
    - secretName: grafana-tls
      hosts:
        - grafana-${ENVIRONMENT}.iaros.com

datasources:
  datasources.yaml:
    apiVersion: 1
    datasources:
      - name: Prometheus
        type: prometheus
        url: http://prometheus-server:80
        access: proxy
        isDefault: true
        editable: true
      - name: Jaeger
        type: jaeger
        url: http://jaeger-query:16686
        access: proxy
        editable: true
      - name: Elasticsearch
        type: elasticsearch
        url: http://elasticsearch:9200
        access: proxy
        database: "logstash-*"
        editable: true

dashboardProviders:
  dashboardproviders.yaml:
    apiVersion: 1
    providers:
      - name: 'default'
        orgId: 1
        folder: ''
        type: file
        disableDeletion: false
        editable: true
        options:
          path: /var/lib/grafana/dashboards/default
      - name: 'iaros'
        orgId: 1
        folder: 'IAROS'
        type: file
        disableDeletion: false
        editable: true
        options:
          path: /var/lib/grafana/dashboards/iaros

dashboards:
  default:
    kubernetes-cluster:
      gnetId: 7249
      revision: 1
      datasource: Prometheus
    kubernetes-pods:
      gnetId: 6417
      revision: 1
      datasource: Prometheus
    node-exporter:
      gnetId: 1860
      revision: 23
      datasource: Prometheus

grafana.ini:
  server:
    root_url: https://grafana-${ENVIRONMENT}.iaros.com
  security:
    admin_user: admin
    admin_password: iaros-grafana-admin-${ENVIRONMENT}
  auth:
    disable_login_form: false
  analytics:
    reporting_enabled: false
    check_for_updates: false
  users:
    allow_sign_up: false
    auto_assign_org_role: Viewer
EOF

    # Install Grafana using Helm
    helm upgrade --install grafana grafana/grafana \
        --namespace "$MONITORING_NAMESPACE" \
        --values /tmp/grafana-values.yaml \
        --wait --timeout 10m

    print_status "Grafana installed successfully"
}

# Function to install Jaeger
install_jaeger() {
    if [[ "$INSTALL_JAEGER" != true ]]; then
        return
    fi

    print_info "Installing Jaeger..."

    # Create Jaeger values file
    cat > /tmp/jaeger-values.yaml << EOF
provisionDataStore:
  cassandra: false
  elasticsearch: true

storage:
  type: elasticsearch
  elasticsearch:
    host: elasticsearch
    port: 9200
    scheme: http
    user: ""
    password: ""

agent:
  enabled: true
  daemonset:
    useHostPort: true
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

collector:
  enabled: true
  replicaCount: 2
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 1Gi
  service:
    type: ClusterIP
    grpc:
      port: 14250
    http:
      port: 14268

query:
  enabled: true
  replicaCount: 2
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    type: ClusterIP
    port: 16686
  ingress:
    enabled: true
    annotations:
      kubernetes.io/ingress.class: nginx
      cert-manager.io/cluster-issuer: letsencrypt-prod
    hosts:
      - jaeger-${ENVIRONMENT}.iaros.com
    tls:
      - secretName: jaeger-tls
        hosts:
          - jaeger-${ENVIRONMENT}.iaros.com

hotrod:
  enabled: false
EOF

    # Install Jaeger using Helm
    helm upgrade --install jaeger jaegertracing/jaeger \
        --namespace "$MONITORING_NAMESPACE" \
        --values /tmp/jaeger-values.yaml \
        --wait --timeout 10m

    print_status "Jaeger installed successfully"
}

# Function to install ELK stack
install_elk() {
    if [[ "$INSTALL_ELK" != true ]]; then
        return
    fi

    print_info "Installing ELK stack..."

    # Install Elasticsearch
    cat > /tmp/elasticsearch-values.yaml << EOF
clusterName: "iaros-elasticsearch"
nodeGroup: "master"

roles:
  master: "true"
  ingest: "true"
  data: "true"
  remote_cluster_client: "true"
  ml: "false"

replicas: 3
minimumMasterNodes: 2

esMajorVersion: "8"

clusterHealthCheckParams: "wait_for_status=yellow&timeout=1s"

protocol: http
httpPort: 9200
transportPort: 9300

resources:
  requests:
    cpu: "500m"
    memory: "2Gi"
  limits:
    cpu: "2000m"
    memory: "4Gi"

volumeClaimTemplate:
  accessModes: ["ReadWriteOnce"]
  storageClassName: "fast-ssd"
  resources:
    requests:
      storage: 50Gi

persistence:
  enabled: true

ingress:
  enabled: true
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: elasticsearch-${ENVIRONMENT}.iaros.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: elasticsearch-tls
      hosts:
        - elasticsearch-${ENVIRONMENT}.iaros.com

esConfig:
  elasticsearch.yml: |
    cluster.name: "iaros-elasticsearch"
    network.host: 0.0.0.0
    discovery.seed_hosts: "elasticsearch-master-headless"
    cluster.initial_master_nodes: "elasticsearch-master-0,elasticsearch-master-1,elasticsearch-master-2"
    bootstrap.memory_lock: false
    xpack.security.enabled: false
    xpack.security.transport.ssl.enabled: false
    xpack.security.http.ssl.enabled: false
EOF

    helm upgrade --install elasticsearch elastic/elasticsearch \
        --namespace "$MONITORING_NAMESPACE" \
        --values /tmp/elasticsearch-values.yaml \
        --wait --timeout 15m

    # Install Kibana
    cat > /tmp/kibana-values.yaml << EOF
elasticsearchHosts: "http://elasticsearch-master:9200"

replicas: 2

resources:
  requests:
    cpu: "250m"
    memory: "1Gi"
  limits:
    cpu: "1000m"
    memory: "2Gi"

service:
  type: ClusterIP
  port: 5601

ingress:
  enabled: true
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: kibana-${ENVIRONMENT}.iaros.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: kibana-tls
      hosts:
        - kibana-${ENVIRONMENT}.iaros.com

kibanaConfig:
  kibana.yml: |
    server.host: 0.0.0.0
    server.shutdownTimeout: 5s
    elasticsearch.hosts: ["http://elasticsearch-master:9200"]
    monitoring.ui.container.elasticsearch.enabled: true
    xpack.security.enabled: false
EOF

    helm upgrade --install kibana elastic/kibana \
        --namespace "$MONITORING_NAMESPACE" \
        --values /tmp/kibana-values.yaml \
        --wait --timeout 10m

    # Install Logstash
    cat > /tmp/logstash-values.yaml << EOF
replicas: 2

resources:
  requests:
    cpu: "250m"
    memory: "1Gi"
  limits:
    cpu: "1000m"
    memory: "2Gi"

persistence:
  enabled: true
  size: 10Gi

logstashConfig:
  logstash.yml: |
    http.host: 0.0.0.0
    xpack.monitoring.elasticsearch.hosts: ["http://elasticsearch-master:9200"]
    path.config: /usr/share/logstash/pipeline

logstashPipeline:
  pipeline.conf: |
    input {
      beats {
        port => 5044
      }
      http {
        port => 8080
      }
    }
    filter {
      if [fields][service] {
        mutate {
          add_field => { "service_name" => "%{[fields][service]}" }
        }
      }
      if [kubernetes] {
        mutate {
          add_field => { "k8s_namespace" => "%{[kubernetes][namespace]}" }
          add_field => { "k8s_pod" => "%{[kubernetes][pod][name]}" }
          add_field => { "k8s_container" => "%{[kubernetes][container][name]}" }
        }
      }
      grok {
        match => { "message" => "%{TIMESTAMP_ISO8601:timestamp} %{LOGLEVEL:level} %{GREEDYDATA:msg}" }
      }
      date {
        match => [ "timestamp", "ISO8601" ]
      }
    }
    output {
      elasticsearch {
        hosts => ["elasticsearch-master:9200"]
        index => "logstash-%{+YYYY.MM.dd}"
      }
    }

service:
  type: ClusterIP
  ports:
    - name: beats
      port: 5044
      protocol: TCP
      targetPort: 5044
    - name: http
      port: 8080
      protocol: TCP
      targetPort: 8080
EOF

    helm upgrade --install logstash elastic/logstash \
        --namespace "$MONITORING_NAMESPACE" \
        --values /tmp/logstash-values.yaml \
        --wait --timeout 10m

    print_status "ELK stack installed successfully"
}

# Function to setup custom dashboards
setup_dashboards() {
    if [[ "$SETUP_DASHBOARDS" != true ]]; then
        return
    fi

    print_info "Setting up custom dashboards..."

    # Create IAROS-specific Grafana dashboards
    local dashboard_dir="$PROJECT_ROOT/infrastructure/observability/dashboards"
    mkdir -p "$dashboard_dir"

    # IAROS Overview Dashboard
    cat > "$dashboard_dir/iaros-overview.json" << 'EOF'
{
  "dashboard": {
    "id": null,
    "title": "IAROS Overview",
    "tags": ["iaros", "overview"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Service Health",
        "type": "stat",
        "targets": [
          {
            "expr": "up{job=~\"iaros-.*\"}"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0}
      },
      {
        "id": 2,
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0}
      },
      {
        "id": 3,
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8}
      },
      {
        "id": 4,
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m])"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8}
      }
    ],
    "time": {"from": "now-1h", "to": "now"},
    "refresh": "5s"
  }
}
EOF

    # Revenue Optimization Dashboard
    cat > "$dashboard_dir/revenue-optimization.json" << 'EOF'
{
  "dashboard": {
    "id": null,
    "title": "Revenue Optimization",
    "tags": ["iaros", "revenue", "pricing"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Pricing Decisions/min",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(pricing_decisions_total[1m])"
          }
        ],
        "gridPos": {"h": 8, "w": 6, "x": 0, "y": 0}
      },
      {
        "id": 2,
        "title": "Revenue per Hour",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(revenue_total[1h])"
          }
        ],
        "gridPos": {"h": 8, "w": 6, "x": 6, "y": 0}
      },
      {
        "id": 3,
        "title": "Load Factor",
        "type": "gauge",
        "targets": [
          {
            "expr": "avg(load_factor)"
          }
        ],
        "gridPos": {"h": 8, "w": 6, "x": 12, "y": 0}
      },
      {
        "id": 4,
        "title": "Ancillary Revenue",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(ancillary_revenue_total[1h])"
          }
        ],
        "gridPos": {"h": 8, "w": 6, "x": 18, "y": 0}
      }
    ],
    "time": {"from": "now-24h", "to": "now"},
    "refresh": "30s"
  }
}
EOF

    # Create ConfigMap for dashboards
    kubectl create configmap iaros-dashboards \
        --from-file="$dashboard_dir" \
        --namespace "$MONITORING_NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -

    print_status "Custom dashboards configured"
}

# Function to setup alerting rules
setup_alerts() {
    if [[ "$SETUP_ALERTS" != true ]]; then
        return
    fi

    print_info "Setting up alerting rules..."

    # Create alerting rules
    cat > /tmp/iaros-alerts.yaml << EOF
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: iaros-alerts
  namespace: ${MONITORING_NAMESPACE}
  labels:
    app: prometheus
    release: prometheus
spec:
  groups:
  - name: iaros.rules
    rules:
    - alert: ServiceDown
      expr: up{job=~"iaros-.*"} == 0
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "IAROS service {{ \$labels.job }} is down"
        description: "{{ \$labels.job }} has been down for more than 1 minute"
    
    - alert: HighErrorRate
      expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "High error rate detected"
        description: "Error rate is {{ \$value }} errors per second"
    
    - alert: HighLatency
      expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High latency detected"
        description: "95th percentile latency is {{ \$value }} seconds"
    
    - alert: LowRevenue
      expr: rate(revenue_total[1h]) < 10000
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Revenue below threshold"
        description: "Hourly revenue is {{ \$value }}, below 10,000 threshold"
    
    - alert: DatabaseConnectionFailure
      expr: database_connections_failed_total > 5
      for: 1m
      labels:
        severity: critical
      annotations:
        summary: "Database connection failures"
        description: "{{ \$value }} database connection failures detected"
    
    - alert: MemoryUsageHigh
      expr: (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes > 0.9
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High memory usage"
        description: "Memory usage is {{ \$value | humanizePercentage }}"
    
    - alert: DiskSpaceLow
      expr: (node_filesystem_size_bytes{fstype!="tmpfs"} - node_filesystem_free_bytes{fstype!="tmpfs"}) / node_filesystem_size_bytes{fstype!="tmpfs"} > 0.8
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Low disk space"
        description: "Disk usage is {{ \$value | humanizePercentage }}"
EOF

    kubectl apply -f /tmp/iaros-alerts.yaml

    print_status "Alerting rules configured"
}

# Function to setup Alertmanager configuration
setup_alertmanager() {
    if [[ "$INSTALL_ALERTMANAGER" != true ]]; then
        return
    fi

    print_info "Configuring Alertmanager..."

    # Create Alertmanager configuration
    cat > /tmp/alertmanager-config.yaml << EOF
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@iaros.com'
  smtp_auth_username: 'alerts@iaros.com'
  smtp_auth_password: 'password'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'
  routes:
  - match:
      severity: critical
    receiver: 'critical-alerts'
  - match:
      severity: warning
    receiver: 'warning-alerts'

receivers:
- name: 'web.hook'
  webhook_configs:
  - url: 'http://alertmanager-webhook:8080/webhook'

- name: 'critical-alerts'
  email_configs:
  - to: 'ops-team@iaros.com'
    subject: 'CRITICAL: {{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
    body: |
      {{ range .Alerts }}
      Alert: {{ .Annotations.summary }}
      Description: {{ .Annotations.description }}
      Labels: {{ range .Labels.SortedPairs }}{{ .Name }}: {{ .Value }}{{ end }}
      {{ end }}
  slack_configs:
  - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
    channel: '#ops-alerts'
    title: 'CRITICAL Alert'
    text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'

- name: 'warning-alerts'
  email_configs:
  - to: 'dev-team@iaros.com'
    subject: 'WARNING: {{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
    body: |
      {{ range .Alerts }}
      Alert: {{ .Annotations.summary }}
      Description: {{ .Annotations.description }}
      Labels: {{ range .Labels.SortedPairs }}{{ .Name }}: {{ .Value }}{{ end }}
      {{ end }}

inhibit_rules:
- source_match:
    severity: 'critical'
  target_match:
    severity: 'warning'
  equal: ['alertname', 'dev', 'instance']
EOF

    # Create ConfigMap for Alertmanager
    kubectl create configmap alertmanager-config \
        --from-file=alertmanager.yml=/tmp/alertmanager-config.yaml \
        --namespace "$MONITORING_NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -

    print_status "Alertmanager configured"
}

# Function to create monitoring ingress
create_monitoring_ingress() {
    print_info "Creating monitoring ingress..."

    cat > /tmp/monitoring-ingress.yaml << EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: monitoring-ingress
  namespace: ${MONITORING_NAMESPACE}
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/auth-type: basic
    nginx.ingress.kubernetes.io/auth-secret: monitoring-auth
    nginx.ingress.kubernetes.io/auth-realm: 'Authentication Required'
spec:
  tls:
  - hosts:
    - monitoring-${ENVIRONMENT}.iaros.com
    secretName: monitoring-tls
  rules:
  - host: monitoring-${ENVIRONMENT}.iaros.com
    http:
      paths:
      - path: /prometheus
        pathType: Prefix
        backend:
          service:
            name: prometheus-server
            port:
              number: 80
      - path: /grafana
        pathType: Prefix
        backend:
          service:
            name: grafana
            port:
              number: 3000
      - path: /jaeger
        pathType: Prefix
        backend:
          service:
            name: jaeger-query
            port:
              number: 16686
      - path: /kibana
        pathType: Prefix
        backend:
          service:
            name: kibana-kibana
            port:
              number: 5601
EOF

    kubectl apply -f /tmp/monitoring-ingress.yaml

    print_status "Monitoring ingress created"
}

# Function to run health checks
run_health_checks() {
    print_info "Running monitoring stack health checks..."

    # Check Prometheus
    if [[ "$INSTALL_PROMETHEUS" == true ]]; then
        kubectl wait --for=condition=ready pod -l app=prometheus -n "$MONITORING_NAMESPACE" --timeout=300s
        print_status "Prometheus is healthy"
    fi

    # Check Grafana
    if [[ "$INSTALL_GRAFANA" == true ]]; then
        kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=grafana -n "$MONITORING_NAMESPACE" --timeout=300s
        print_status "Grafana is healthy"
    fi

    # Check Jaeger
    if [[ "$INSTALL_JAEGER" == true ]]; then
        kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=jaeger -n "$MONITORING_NAMESPACE" --timeout=300s
        print_status "Jaeger is healthy"
    fi

    # Check Elasticsearch
    if [[ "$INSTALL_ELK" == true ]]; then
        kubectl wait --for=condition=ready pod -l app=elasticsearch-master -n "$MONITORING_NAMESPACE" --timeout=600s
        print_status "ELK stack is healthy"
    fi

    print_status "All monitoring components are healthy"
}

# Function to display access information
display_access_info() {
    print_info "Monitoring stack access information:"
    echo ""
    echo "🌐 Web Interfaces:"
    
    if [[ "$INSTALL_PROMETHEUS" == true ]]; then
        echo "  - Prometheus: https://monitoring-${ENVIRONMENT}.iaros.com/prometheus"
    fi
    
    if [[ "$INSTALL_GRAFANA" == true ]]; then
        echo "  - Grafana: https://monitoring-${ENVIRONMENT}.iaros.com/grafana"
        echo "    Username: admin"
        echo "    Password: iaros-grafana-admin-${ENVIRONMENT}"
    fi
    
    if [[ "$INSTALL_JAEGER" == true ]]; then
        echo "  - Jaeger: https://monitoring-${ENVIRONMENT}.iaros.com/jaeger"
    fi
    
    if [[ "$INSTALL_ELK" == true ]]; then
        echo "  - Kibana: https://monitoring-${ENVIRONMENT}.iaros.com/kibana"
    fi
    
    echo ""
    echo "🔧 Port Forwarding Commands:"
    echo "  kubectl port-forward svc/prometheus-server 9090:80 -n $MONITORING_NAMESPACE"
    echo "  kubectl port-forward svc/grafana 3000:3000 -n $MONITORING_NAMESPACE"
    echo "  kubectl port-forward svc/jaeger-query 16686:16686 -n $MONITORING_NAMESPACE"
    echo "  kubectl port-forward svc/kibana-kibana 5601:5601 -n $MONITORING_NAMESPACE"
    echo ""
    echo "📊 Monitoring Endpoints:"
    echo "  - Metrics: /metrics"
    echo "  - Health: /health"
    echo "  - Ready: /ready"
    echo ""
}

# Main function
main() {
    echo "📊 IAROS Monitoring Setup"
    echo "========================="
    
    parse_args "$@"
    check_prerequisites
    
    print_info "Setting up monitoring stack for environment: $ENVIRONMENT"
    
    # Install components
    install_prometheus
    install_grafana
    install_jaeger
    install_elk
    
    # Setup configurations
    setup_dashboards
    setup_alerts
    setup_alertmanager
    create_monitoring_ingress
    
    # Verify installation
    run_health_checks
    
    # Display access information
    display_access_info
    
    print_status "IAROS monitoring stack setup completed successfully!"
}

# Error handling
trap 'print_error "Monitoring setup failed at line $LINENO"' ERR

# Execute main function
main "$@" 