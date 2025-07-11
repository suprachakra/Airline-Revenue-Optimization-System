#!/bin/bash

# IAROS Development Environment Setup Script
# Automatically sets up complete development environment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
IAROS_VERSION="3.0.0"
NODE_VERSION="18"
GO_VERSION="1.19"
PYTHON_VERSION="3.9"

echo -e "${BLUE}🚀 IAROS Development Environment Setup v${IAROS_VERSION}${NC}"
echo "=================================="

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

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check system requirements
check_system_requirements() {
    print_info "Checking system requirements..."
    
    # Check OS
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
        print_status "Operating System: Linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macos"
        print_status "Operating System: macOS"
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        OS="windows"
        print_status "Operating System: Windows (WSL/Cygwin)"
    else
        print_error "Unsupported operating system: $OSTYPE"
        exit 1
    fi
    
    # Check available memory
    if command_exists free; then
        MEMORY_GB=$(free -g | awk 'NR==2{print $2}')
        if [ "$MEMORY_GB" -lt 8 ]; then
            print_warning "Low memory detected (${MEMORY_GB}GB). Recommended: 8GB+"
        else
            print_status "Memory: ${MEMORY_GB}GB"
        fi
    fi
    
    # Check disk space
    if command_exists df; then
        DISK_SPACE=$(df -h . | awk 'NR==2{print $4}')
        print_status "Available disk space: $DISK_SPACE"
    fi
}

# Function to install dependencies based on OS
install_system_dependencies() {
    print_info "Installing system dependencies..."
    
    case $OS in
        "linux")
            if command_exists apt-get; then
                sudo apt-get update
                sudo apt-get install -y curl wget git build-essential
                print_status "System dependencies installed (Ubuntu/Debian)"
            elif command_exists yum; then
                sudo yum update -y
                sudo yum install -y curl wget git gcc gcc-c++ make
                print_status "System dependencies installed (RHEL/CentOS)"
            fi
            ;;
        "macos")
            if ! command_exists brew; then
                print_info "Installing Homebrew..."
                /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
            fi
            brew install curl wget git
            print_status "System dependencies installed (macOS)"
            ;;
        "windows")
            print_info "Please ensure you have Git, curl, and wget installed"
            print_status "System dependencies assumed installed (Windows)"
            ;;
    esac
}

# Function to install Docker
install_docker() {
    if command_exists docker; then
        print_status "Docker already installed: $(docker --version)"
        return
    fi
    
    print_info "Installing Docker..."
    
    case $OS in
        "linux")
            curl -fsSL https://get.docker.com -o get-docker.sh
            sudo sh get-docker.sh
            sudo usermod -aG docker $USER
            rm get-docker.sh
            ;;
        "macos")
            print_info "Please install Docker Desktop for Mac from https://docker.com/products/docker-desktop"
            print_warning "Manual installation required - script will continue assuming Docker will be available"
            ;;
        "windows")
            print_info "Please install Docker Desktop for Windows from https://docker.com/products/docker-desktop"
            print_warning "Manual installation required - script will continue assuming Docker will be available"
            ;;
    esac
    
    print_status "Docker installation completed"
}

# Function to install Docker Compose
install_docker_compose() {
    if command_exists docker-compose; then
        print_status "Docker Compose already installed: $(docker-compose --version)"
        return
    fi
    
    print_info "Installing Docker Compose..."
    
    if [[ "$OS" == "linux" ]]; then
        sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        sudo chmod +x /usr/local/bin/docker-compose
        print_status "Docker Compose installed"
    else
        print_info "Docker Compose should be included with Docker Desktop"
    fi
}

# Function to install Node.js
install_nodejs() {
    if command_exists node; then
        NODE_CURRENT=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
        if [ "$NODE_CURRENT" -ge "$NODE_VERSION" ]; then
            print_status "Node.js already installed: $(node --version)"
            return
        fi
    fi
    
    print_info "Installing Node.js v${NODE_VERSION}..."
    
    # Install using Node Version Manager (nvm)
    if ! command_exists nvm; then
        curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
        export NVM_DIR="$HOME/.nvm"
        [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    fi
    
    nvm install $NODE_VERSION
    nvm use $NODE_VERSION
    nvm alias default $NODE_VERSION
    
    print_status "Node.js v${NODE_VERSION} installed"
    
    # Install global packages
    npm install -g yarn pm2 typescript ts-node
    print_status "Global npm packages installed"
}

# Function to install Go
install_go() {
    if command_exists go; then
        GO_CURRENT=$(go version | awk '{print $3}' | cut -d'o' -f2)
        print_status "Go already installed: $GO_CURRENT"
        return
    fi
    
    print_info "Installing Go v${GO_VERSION}..."
    
    case $OS in
        "linux")
            wget "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
            rm "go${GO_VERSION}.linux-amd64.tar.gz"
            ;;
        "macos")
            if command_exists brew; then
                brew install go
            else
                curl -O "https://go.dev/dl/go${GO_VERSION}.darwin-amd64.pkg"
                print_info "Please install the downloaded Go package manually"
            fi
            ;;
    esac
    
    # Add Go to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc
    export PATH=$PATH:/usr/local/go/bin
    
    print_status "Go v${GO_VERSION} installed"
}

# Function to install Python
install_python() {
    if command_exists python3; then
        PYTHON_CURRENT=$(python3 --version | cut -d' ' -f2 | cut -d'.' -f1,2)
        print_status "Python already installed: $(python3 --version)"
    else
        print_info "Installing Python v${PYTHON_VERSION}..."
        
        case $OS in
            "linux")
                sudo apt-get install -y python3 python3-pip python3-venv
                ;;
            "macos")
                brew install python@3.9
                ;;
        esac
        
        print_status "Python v${PYTHON_VERSION} installed"
    fi
    
    # Install pipenv for virtual environments
    pip3 install --user pipenv virtualenv
    print_status "Python package managers installed"
}

# Function to install Kubernetes tools
install_k8s_tools() {
    print_info "Installing Kubernetes tools..."
    
    # Install kubectl
    if ! command_exists kubectl; then
        case $OS in
            "linux")
                curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
                sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
                rm kubectl
                ;;
            "macos")
                brew install kubectl
                ;;
        esac
        print_status "kubectl installed"
    else
        print_status "kubectl already installed: $(kubectl version --client --short)"
    fi
    
    # Install Helm
    if ! command_exists helm; then
        curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
        print_status "Helm installed"
    else
        print_status "Helm already installed: $(helm version --short)"
    fi
    
    # Install Minikube for local development
    if ! command_exists minikube; then
        case $OS in
            "linux")
                curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
                sudo install minikube-linux-amd64 /usr/local/bin/minikube
                rm minikube-linux-amd64
                ;;
            "macos")
                brew install minikube
                ;;
        esac
        print_status "Minikube installed"
    else
        print_status "Minikube already installed: $(minikube version)"
    fi
}

# Function to setup development databases
setup_databases() {
    print_info "Setting up development databases..."
    
    # Create docker-compose for development databases
    cat > docker-compose.dev.yml << EOF
version: '3.8'
services:
  postgres:
    image: postgres:14
    container_name: iaros_postgres_dev
    environment:
      POSTGRES_DB: iaros
      POSTGRES_USER: iaros_user
      POSTGRES_PASSWORD: iaros_pass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./infrastructure/database/init-schemas.sql:/docker-entrypoint-initdb.d/init-schemas.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U iaros_user -d iaros"]
      interval: 30s
      timeout: 10s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: iaros_redis_dev
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 30s
      timeout: 10s
      retries: 5

  mongodb:
    image: mongo:6
    container_name: iaros_mongo_dev
    environment:
      MONGO_INITDB_ROOT_USERNAME: iaros_user
      MONGO_INITDB_ROOT_PASSWORD: iaros_pass
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 30s
      timeout: 10s
      retries: 5

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.5.0
    container_name: iaros_elasticsearch_dev
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    ports:
      - "9200:9200"
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data

volumes:
  postgres_data:
  redis_data:
  mongo_data:
  elasticsearch_data:
EOF

    # Start development databases
    docker-compose -f docker-compose.dev.yml up -d
    
    print_status "Development databases started"
}

# Function to clone and setup IAROS repositories
setup_iaros_repositories() {
    print_info "Setting up IAROS repositories..."
    
    # Create workspace directory
    mkdir -p ~/iaros-workspace
    cd ~/iaros-workspace
    
    # Clone main repository (if not already cloned)
    if [ ! -d "iaros" ]; then
        git clone https://github.com/iaros/iaros.git
        cd iaros
    else
        cd iaros
        git pull origin main
    fi
    
    print_status "IAROS repositories set up"
}

# Function to install project dependencies
install_project_dependencies() {
    print_info "Installing project dependencies..."
    
    # Go dependencies
    if [ -f "go.mod" ]; then
        go mod download
        go mod tidy
        print_status "Go dependencies installed"
    fi
    
    # Node.js dependencies for web portal
    if [ -d "frontend/web-portal" ]; then
        cd frontend/web-portal
        npm install
        cd ../..
        print_status "Web portal dependencies installed"
    fi
    
    # Node.js dependencies for mobile app
    if [ -d "frontend/mobile-app" ]; then
        cd frontend/mobile-app
        npm install
        cd ../..
        print_status "Mobile app dependencies installed"
    fi
    
    # Python dependencies
    if [ -f "requirements.txt" ]; then
        pip3 install -r requirements.txt
        print_status "Python dependencies installed"
    fi
}

# Function to setup environment variables
setup_environment() {
    print_info "Setting up environment variables..."
    
    # Create .env file for development
    cat > .env.development << EOF
# IAROS Development Environment Configuration

# Application
NODE_ENV=development
PORT=3000
API_PORT=8080

# Database URLs
DATABASE_URL=postgresql://iaros_user:iaros_pass@localhost:5432/iaros
REDIS_URL=redis://localhost:6379
MONGODB_URL=mongodb://iaros_user:iaros_pass@localhost:27017/iaros
ELASTICSEARCH_URL=http://localhost:9200

# API Keys (Development)
JWT_SECRET=dev_jwt_secret_key_change_in_production
API_KEY=dev_api_key

# External Services (Development)
STRIPE_SECRET_KEY=sk_test_your_stripe_key
SENDGRID_API_KEY=your_sendgrid_api_key

# Monitoring
SENTRY_DSN=your_sentry_dsn
PROMETHEUS_URL=http://localhost:9090

# Feature Flags
ENABLE_ANALYTICS=true
ENABLE_CACHING=true
ENABLE_MONITORING=true

# Debug
DEBUG=iaros:*
LOG_LEVEL=debug
EOF

    print_status "Environment variables configured"
}

# Function to setup IDE configurations
setup_ide_configs() {
    print_info "Setting up IDE configurations..."
    
    # VS Code settings
    mkdir -p .vscode
    cat > .vscode/settings.json << EOF
{
    "go.toolsManagement.autoUpdate": true,
    "go.useLanguageServer": true,
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "go.testFlags": ["-v"],
    "typescript.preferences.importModuleSpecifier": "relative",
    "eslint.workingDirectories": ["frontend/web-portal", "frontend/mobile-app"],
    "python.defaultInterpreterPath": "python3",
    "python.linting.enabled": true,
    "python.linting.pylintEnabled": true,
    "files.exclude": {
        "**/node_modules": true,
        "**/dist": true,
        "**/build": true,
        "**/.git": false
    }
}
EOF

    # VS Code extensions
    cat > .vscode/extensions.json << EOF
{
    "recommendations": [
        "golang.go",
        "ms-vscode.vscode-typescript-next",
        "ms-python.python",
        "ms-vscode.vscode-json",
        "redhat.vscode-yaml",
        "ms-kubernetes-tools.vscode-kubernetes-tools",
        "ms-vscode-remote.remote-containers",
        "bradlc.vscode-tailwindcss",
        "esbenp.prettier-vscode",
        "ms-vscode.vscode-eslint"
    ]
}
EOF

    print_status "IDE configurations set up"
}

# Function to run initial tests
run_initial_tests() {
    print_info "Running initial tests to verify setup..."
    
    # Wait for services to be ready
    ./scripts/wait-for-services.sh
    
    # Run basic health checks
    if command_exists go; then
        go version
        print_status "Go environment working"
    fi
    
    if command_exists node; then
        node --version
        npm --version
        print_status "Node.js environment working"
    fi
    
    if command_exists python3; then
        python3 --version
        print_status "Python environment working"
    fi
    
    if command_exists docker; then
        docker --version
        print_status "Docker environment working"
    fi
    
    print_status "Initial tests completed"
}

# Function to display final instructions
display_final_instructions() {
    echo ""
    echo -e "${GREEN}🎉 IAROS Development Environment Setup Complete!${NC}"
    echo "============================================="
    echo ""
    echo -e "${BLUE}📚 Next Steps:${NC}"
    echo "1. Restart your terminal to ensure all PATH changes take effect"
    echo "2. Run: source ~/.bashrc (or ~/.zshrc)"
    echo "3. Navigate to your project: cd ~/iaros-workspace/iaros"
    echo "4. Start development services: docker-compose -f docker-compose.dev.yml up -d"
    echo "5. Wait for services: ./scripts/wait-for-services.sh"
    echo "6. Start frontend: cd frontend/web-portal && npm start"
    echo "7. Start API Gateway: go run services/api_gateway/main.go"
    echo ""
    echo -e "${BLUE}🌐 Access URLs:${NC}"
    echo "- Web Portal: http://localhost:3000"
    echo "- API Gateway: http://localhost:8080"
    echo "- Database: postgresql://iaros_user:iaros_pass@localhost:5432/iaros"
    echo "- Redis: redis://localhost:6379"
    echo "- MongoDB: mongodb://iaros_user:iaros_pass@localhost:27017/iaros"
    echo ""
    echo -e "${BLUE}🛠️  Development Commands:${NC}"
    echo "- Test everything: ./run-complete-testing.sh"
    echo "- View logs: docker-compose -f docker-compose.dev.yml logs -f"
    echo "- Stop services: docker-compose -f docker-compose.dev.yml down"
    echo ""
    echo -e "${BLUE}📖 Documentation:${NC}"
    echo "- Project README: ./README.md"
    echo "- API Docs: http://localhost:8080/docs"
    echo "- Contributing: ./CONTRIBUTING.md"
    echo ""
    echo -e "${YELLOW}⚠️  Important Notes:${NC}"
    echo "- This setup is for development only"
    echo "- Default credentials are for development use"
    echo "- Change all passwords in production"
    echo "- Review security settings before deployment"
    echo ""
}

# Main execution flow
main() {
    # Check if running as root (not recommended)
    if [ "$EUID" -eq 0 ]; then
        print_warning "Running as root is not recommended"
        print_info "Please run this script as a regular user"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # Setup steps
    check_system_requirements
    install_system_dependencies
    install_docker
    install_docker_compose
    install_nodejs
    install_go
    install_python
    install_k8s_tools
    setup_databases
    setup_iaros_repositories
    install_project_dependencies
    setup_environment
    setup_ide_configs
    run_initial_tests
    display_final_instructions
    
    print_status "Development environment setup completed successfully!"
}

# Error handling
trap 'print_error "Setup failed at line $LINENO. Exit code: $?"' ERR

# Run main function
main "$@" 