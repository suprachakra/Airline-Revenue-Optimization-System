#!/bin/bash

# Wait for IAROS services to be ready
# Used in demo and development environments

set -e

echo "🚀 Waiting for IAROS services to start..."

# Function to check if a service is healthy
check_service() {
    local service_name=$1
    local url=$2
    local max_attempts=60
    local attempt=1

    echo "⏳ Checking $service_name..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            echo "✅ $service_name is ready!"
            return 0
        fi
        
        echo "   Attempt $attempt/$max_attempts - waiting..."
        sleep 5
        attempt=$((attempt + 1))
    done
    
    echo "❌ $service_name failed to start within timeout"
    return 1
}

# Function to check database connectivity
check_database() {
    local db_name=$1
    local connection_string=$2
    local max_attempts=30
    local attempt=1

    echo "⏳ Checking $db_name..."
    
    while [ $attempt -le $max_attempts ]; do
        if docker exec iaros_postgres_1 2>/dev/null pg_isready -U iaros_user -d iaros > /dev/null 2>&1; then
            echo "✅ $db_name is ready!"
            return 0
        fi
        
        echo "   Attempt $attempt/$max_attempts - waiting..."
        sleep 3
        attempt=$((attempt + 1))
    done
    
    echo "❌ $db_name failed to start within timeout"
    return 1
}

# Check core infrastructure
echo "📊 Checking infrastructure services..."

check_database "PostgreSQL" "postgresql://iaros_user:iaros_pass@localhost:5432/iaros"

if curl -s http://localhost:6379 > /dev/null 2>&1 || docker exec iaros_redis_1 2>/dev/null redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis is ready!"
else
    echo "❌ Redis is not responding"
    exit 1
fi

if curl -s http://localhost:27017 > /dev/null 2>&1 || docker exec iaros_mongo_1 2>/dev/null mongosh --eval "db.adminCommand('ping')" > /dev/null 2>&1; then
    echo "✅ MongoDB is ready!"
else
    echo "❌ MongoDB is not responding"
    exit 1
fi

# Check application services
echo "🎯 Checking application services..."

check_service "API Gateway" "http://localhost:8080/health"
check_service "Web Portal" "http://localhost:3000"
check_service "Pricing Service" "http://localhost:8081/health"
check_service "User Management" "http://localhost:8085/health"

# Check monitoring services
echo "📈 Checking monitoring services..."

check_service "Grafana" "http://localhost:3001/api/health"
check_service "Prometheus" "http://localhost:9090/-/healthy"

echo ""
echo "🎉 All IAROS services are ready!"
echo ""
echo "🌐 Access URLs:"
echo "   Web Portal:     http://localhost:3000"
echo "   API Gateway:    http://localhost:8080"
echo "   Admin Panel:    http://localhost:3000/admin"
echo "   Grafana:        http://localhost:3001 (admin/admin)"
echo "   Prometheus:     http://localhost:9090"
echo ""
echo "👤 Demo Credentials:"
echo "   Email:          demo@iaros.com"
echo "   Password:       DemoPass123!"
echo ""
echo "📚 Next steps:"
echo "   1. Visit http://localhost:3000 to explore the platform"
echo "   2. Check API documentation at http://localhost:8080/docs"
echo "   3. View monitoring at http://localhost:3001"
echo "   4. Run tests with: ./run-complete-testing.sh"
echo "" 