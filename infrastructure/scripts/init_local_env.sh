#!/bin/bash
# init_local_env.sh - Enterprise IAROS Local Development Environment Setup
# Author: IAROS Infrastructure Team
# Version: 2.0.0

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LOG_DIR="${PROJECT_ROOT}/logs"
VENV_DIR="${PROJECT_ROOT}/venv"
CONFIG_DIR="${PROJECT_ROOT}/infrastructure/config"
REQUIREMENTS_FILE="${PROJECT_ROOT}/requirements.txt"
GO_VERSION="1.21"
NODE_VERSION="18"
PYTHON_VERSION="3.9"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/init.log"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/init.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/init.log"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/init.log"
}

# Error handling
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        log_error "Environment initialization failed with exit code $exit_code"
        log_info "Please check the logs and try again"
    fi
    exit $exit_code
}

trap cleanup EXIT

# Utility functions
check_command() {
    if command -v "$1" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

get_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macos"
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        echo "windows"
    else
        echo "unknown"
    fi
}

# Validation functions
validate_system_requirements() {
    log_info "Validating system requirements..."
    
    local os=$(get_os)
    log_info "Detected OS: $os"
    
    # Check minimum system requirements
    local total_memory_kb=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || echo "8388608")
    local total_memory_gb=$((total_memory_kb / 1024 / 1024))
    
    if [[ $total_memory_gb -lt 8 ]]; then
        log_warn "System has ${total_memory_gb}GB RAM. Recommended minimum is 8GB."
    fi
    
    # Check available disk space
    local available_space=$(df "$PROJECT_ROOT" | awk 'NR==2 {print $4}')
    local available_gb=$((available_space / 1024 / 1024))
    
    if [[ $available_gb -lt 20 ]]; then
        log_warn "Available disk space: ${available_gb}GB. Recommended minimum is 20GB."
    fi
    
    log_success "System requirements validation completed"
}

# Language installation functions
install_python() {
    log_info "Setting up Python environment..."
    
    if check_command python3; then
        local current_version=$(python3 --version | cut -d' ' -f2 | cut -d'.' -f1,2)
        log_info "Found Python $current_version"
        
        if [[ "$current_version" < "$PYTHON_VERSION" ]]; then
            log_warn "Python $current_version found, but $PYTHON_VERSION+ recommended"
        fi
    else
        log_error "Python 3 not found. Please install Python $PYTHON_VERSION or later."
        return 1
    fi
    
    # Create virtual environment
    if [[ ! -d "$VENV_DIR" ]]; then
        log_info "Creating Python virtual environment..."
        python3 -m venv "$VENV_DIR"
    fi
    
    # Activate virtual environment
    source "$VENV_DIR/bin/activate"
    
    # Upgrade pip
    log_info "Upgrading pip..."
    pip install --upgrade pip setuptools wheel
    
    # Install requirements if file exists
    if [[ -f "$REQUIREMENTS_FILE" ]]; then
        log_info "Installing Python dependencies..."
        pip install -r "$REQUIREMENTS_FILE"
    else
        log_info "Creating basic requirements.txt..."
        cat > "$REQUIREMENTS_FILE" << EOF
# Core IAROS Python Dependencies
fastapi==0.104.1
uvicorn[standard]==0.24.0
pydantic==2.5.0
sqlalchemy==2.0.23
alembic==1.13.0
redis==5.0.1
celery==5.3.4
pytest==7.4.3
pytest-asyncio==0.21.1
black==23.11.0
flake8==6.1.0
mypy==1.7.1
pandas==2.1.3
numpy==1.25.2
scikit-learn==1.3.2
matplotlib==3.8.2
psycopg2-binary==2.9.9
requests==2.31.0
aiohttp==3.9.1
pyjwt==2.8.0
bcrypt==4.1.1
prometheus-client==0.19.0
structlog==23.2.0
EOF
        pip install -r "$REQUIREMENTS_FILE"
    fi
    
    log_success "Python environment setup completed"
}

install_go() {
    log_info "Setting up Go environment..."
    
    if check_command go; then
        local current_version=$(go version | cut -d' ' -f3 | sed 's/go//')
        log_info "Found Go $current_version"
        
        if [[ "$current_version" < "$GO_VERSION" ]]; then
            log_warn "Go $current_version found, but $GO_VERSION+ recommended"
        fi
    else
        log_warn "Go not found. Please install Go $GO_VERSION or later."
        log_info "Visit: https://golang.org/doc/install"
        return 1
    fi
    
    # Set up Go workspace
    export GOPATH="$PROJECT_ROOT/go"
    export PATH="$PATH:$GOPATH/bin"
    
    # Create Go directories
    mkdir -p "$GOPATH"/{bin,src,pkg}
    
    # Install common Go tools
    log_info "Installing Go development tools..."
    go install golang.org/x/tools/cmd/goimports@latest
    go install golang.org/x/lint/golint@latest
    go install github.com/securecodewarrior/sast-scan/cmd/sast-scan@latest
    
    log_success "Go environment setup completed"
}

install_node() {
    log_info "Setting up Node.js environment..."
    
    if check_command node; then
        local current_version=$(node --version | sed 's/v//' | cut -d'.' -f1)
        log_info "Found Node.js v$current_version"
        
        if [[ $current_version -lt $NODE_VERSION ]]; then
            log_warn "Node.js v$current_version found, but v$NODE_VERSION+ recommended"
        fi
    else
        log_warn "Node.js not found. Please install Node.js v$NODE_VERSION or later."
        log_info "Visit: https://nodejs.org/"
        return 1
    fi
    
    # Install global packages
    log_info "Installing Node.js global packages..."
    npm install -g npm@latest
    npm install -g yarn
    npm install -g @angular/cli
    npm install -g create-react-app
    npm install -g eslint
    npm install -g prettier
    
    # Install frontend dependencies
    if [[ -f "$PROJECT_ROOT/frontend/web-portal/package.json" ]]; then
        log_info "Installing frontend dependencies..."
        cd "$PROJECT_ROOT/frontend/web-portal"
        npm install
        cd "$PROJECT_ROOT"
    fi
    
    if [[ -f "$PROJECT_ROOT/frontend/mobile-app/package.json" ]]; then
        log_info "Installing mobile app dependencies..."
        cd "$PROJECT_ROOT/frontend/mobile-app"
        npm install
        cd "$PROJECT_ROOT"
    fi
    
    log_success "Node.js environment setup completed"
}

# Database and infrastructure setup
setup_databases() {
    log_info "Setting up database environment..."
    
    # Check if Docker is available
    if check_command docker; then
        log_info "Setting up development databases with Docker..."
        
        # Create docker-compose for development databases
        cat > "$PROJECT_ROOT/docker-compose.dev.yml" << EOF
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: iaros_dev
      POSTGRES_USER: iaros_user
      POSTGRES_PASSWORD: iaros_pass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./infrastructure/database:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U iaros_user -d iaros_dev"]
      interval: 30s
      timeout: 10s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 30s
      timeout: 10s
      retries: 5

  mongodb:
    image: mongo:7
    environment:
      MONGO_INITDB_ROOT_USERNAME: iaros_user
      MONGO_INITDB_ROOT_PASSWORD: iaros_pass
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db

volumes:
  postgres_data:
  redis_data:
  mongodb_data:
EOF
        
        # Start development databases
        docker-compose -f "$PROJECT_ROOT/docker-compose.dev.yml" up -d
        
        # Wait for databases to be ready
        log_info "Waiting for databases to be ready..."
        sleep 10
        
        log_success "Development databases setup completed"
    else
        log_warn "Docker not found. Database setup skipped."
        log_info "Please install Docker to set up development databases automatically."
    fi
}

# Configuration setup
setup_configuration() {
    log_info "Setting up configuration files..."
    
    # Create logs directory
    mkdir -p "$LOG_DIR"
    
    # Copy sample configurations
    if [[ -f "$CONFIG_DIR/config.sample.yaml" ]] && [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
        cp "$CONFIG_DIR/config.sample.yaml" "$CONFIG_DIR/config.yaml"
        log_info "Created config.yaml from sample"
    fi
    
    if [[ -f "$CONFIG_DIR/dev.env.example" ]] && [[ ! -f "$CONFIG_DIR/dev.env" ]]; then
        cp "$CONFIG_DIR/dev.env.example" "$CONFIG_DIR/dev.env"
        log_info "Created dev.env from example"
    fi
    
    # Create basic environment file
    cat > "$PROJECT_ROOT/.env" << EOF
# IAROS Development Environment
NODE_ENV=development
LOG_LEVEL=debug
DATABASE_URL=postgresql://iaros_user:iaros_pass@localhost:5432/iaros_dev
REDIS_URL=redis://localhost:6379
MONGODB_URL=mongodb://iaros_user:iaros_pass@localhost:27017
API_PORT=8080
WEB_PORT=3000
JWT_SECRET=dev_secret_key_change_in_production
ENCRYPTION_KEY=dev_encryption_key_change_in_production
EOF
    
    log_success "Configuration setup completed"
}

# Development tools setup
setup_development_tools() {
    log_info "Setting up development tools..."
    
    # Install pre-commit hooks if available
    if check_command pre-commit; then
        log_info "Setting up pre-commit hooks..."
        cat > "$PROJECT_ROOT/.pre-commit-config.yaml" << EOF
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
      - id: check-json
  - repo: https://github.com/psf/black
    rev: 23.11.0
    hooks:
      - id: black
        language_version: python3
  - repo: https://github.com/pycqa/flake8
    rev: 6.1.0
    hooks:
      - id: flake8
EOF
        pre-commit install
    fi
    
    # Create VS Code settings
    mkdir -p "$PROJECT_ROOT/.vscode"
    cat > "$PROJECT_ROOT/.vscode/settings.json" << EOF
{
    "python.defaultInterpreterPath": "./venv/bin/python",
    "python.linting.enabled": true,
    "python.linting.flake8Enabled": true,
    "python.formatting.provider": "black",
    "editor.formatOnSave": true,
    "go.gopath": "./go",
    "go.formatTool": "goimports",
    "typescript.preferences.includePackageJsonAutoImports": "on",
    "eslint.workingDirectories": ["frontend/web-portal", "frontend/mobile-app"]
}
EOF
    
    log_success "Development tools setup completed"
}

# Test environment setup
setup_test_environment() {
    log_info "Setting up test environment..."
    
    # Create test configuration
    cat > "$PROJECT_ROOT/pytest.ini" << EOF
[tool:pytest]
testpaths = tests
python_files = test_*.py *_test.py
python_classes = Test*
python_functions = test_*
addopts = 
    --verbose
    --tb=short
    --cov=.
    --cov-report=html
    --cov-report=term-missing
markers =
    unit: Unit tests
    integration: Integration tests
    e2e: End-to-end tests
    slow: Slow running tests
EOF
    
    # Create Makefile for common tasks
    cat > "$PROJECT_ROOT/Makefile" << EOF
.PHONY: help install test lint format clean

help:
	@echo "Available commands:"
	@echo "  install     Install dependencies"
	@echo "  test        Run tests"
	@echo "  lint        Run linting"
	@echo "  format      Format code"
	@echo "  clean       Clean build artifacts"

install:
	./infrastructure/scripts/init_local_env.sh

test:
	pytest tests/

lint:
	flake8 .
	golint ./...
	eslint frontend/

format:
	black .
	gofmt -w .
	prettier --write frontend/

clean:
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete
	rm -rf .coverage htmlcov/
EOF
    
    log_success "Test environment setup completed"
}

# Display setup information
display_setup_info() {
    log_info "IAROS Development Environment Setup Complete!"
    echo "=============================================="
    
    echo ""
    echo "Environment Information:"
    echo "  - Project Root: $PROJECT_ROOT"
    echo "  - Python Virtual Environment: $VENV_DIR"
    echo "  - Go Workspace: $GOPATH"
    echo "  - Logs Directory: $LOG_DIR"
    
    echo ""
    echo "To activate the Python environment:"
    echo "  source $VENV_DIR/bin/activate"
    
    echo ""
    echo "To start development databases:"
    echo "  docker-compose -f docker-compose.dev.yml up -d"
    
    echo ""
    echo "To start the application:"
    echo "  ./infrastructure/scripts/start_all.sh"
    
    echo ""
    echo "Useful commands:"
    echo "  - Run tests: make test"
    echo "  - Lint code: make lint"
    echo "  - Format code: make format"
    echo "  - Clean artifacts: make clean"
}

# Main execution
main() {
    log_info "Initializing IAROS Local Development Environment"
    log_info "================================================"
    
    # Create logs directory first
    mkdir -p "$LOG_DIR"
    
    # System validation
    validate_system_requirements
    
    # Language environments
    install_python
    install_go || log_warn "Go setup incomplete - some services may not work"
    install_node || log_warn "Node.js setup incomplete - frontend may not work"
    
    # Infrastructure setup
    setup_databases
    setup_configuration
    
    # Development tools
    setup_development_tools
    setup_test_environment
    
    # Display information
    display_setup_info
    
    log_success "Local development environment initialization completed!"
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
