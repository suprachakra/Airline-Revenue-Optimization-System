# IAROS Shared Libraries and SDKs

This directory contains shared libraries and SDKs that are used across all IAROS services. These libraries provide common functionality, reduce code duplication, and ensure consistent implementation patterns across the platform.

## 📁 Library Structure

```
common/libraries/
├── README.md
├── go/
│   ├── iaros-core/          # Core Go library
│   ├── iaros-db/           # Database utilities
│   ├── iaros-cache/        # Redis/caching utilities
│   ├── iaros-auth/         # Authentication utilities
│   ├── iaros-monitoring/   # Monitoring and metrics
│   └── iaros-config/       # Configuration management
├── python/
│   ├── iaros-analytics/    # Analytics and ML utilities
│   ├── iaros-data/         # Data processing utilities
│   └── iaros-validation/   # Data validation
└── javascript/
    ├── iaros-ui/           # UI components library
    ├── iaros-api/          # API client library
    └── iaros-utils/        # Common utilities
```

## 🚀 Core Libraries

### 1. IAROS Core Library (Go)

**Purpose**: Core functionality shared across all Go services

**Features**:
- HTTP client with circuit breaker
- Error handling patterns
- Logging utilities
- Request/response models
- Service discovery client

**Installation**:
```bash
go get github.com/iaros/common/libraries/go/iaros-core
```

**Usage Example**:
```go
package main

import (
    "github.com/iaros/common/libraries/go/iaros-core/client"
    "github.com/iaros/common/libraries/go/iaros-core/logging"
)

func main() {
    // Initialize logger
    logger := logging.NewIAROSLogger("service-name")
    
    // Create HTTP client with circuit breaker
    httpClient := client.NewHTTPClient(client.Config{
        Timeout: 30 * time.Second,
        Retries: 3,
        CircuitBreaker: true,
    })
    
    // Make request
    resp, err := httpClient.Get("https://api.example.com/data")
    if err != nil {
        logger.Error("Request failed", "error", err)
        return
    }
    
    logger.Info("Request successful", "status", resp.StatusCode)
}
```

### 2. IAROS Database Library (Go)

**Purpose**: Database connection management and utilities

**Features**:
- Connection pooling
- Transaction management
- Query builders
- Migration utilities
- Health checks

**Installation**:
```bash
go get github.com/iaros/common/libraries/go/iaros-db
```

**Usage Example**:
```go
package main

import (
    "github.com/iaros/common/libraries/go/iaros-db/postgres"
    "github.com/iaros/common/libraries/go/iaros-db/redis"
)

func main() {
    // Initialize PostgreSQL connection
    pgConfig := postgres.Config{
        Host:     "localhost",
        Port:     5432,
        Database: "iaros",
        Username: "iaros_user",
        Password: "secret",
        MaxConns: 50,
    }
    
    db, err := postgres.NewConnection(pgConfig)
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // Initialize Redis connection
    redisConfig := redis.Config{
        Host:     "localhost",
        Port:     6379,
        Database: 0,
        Password: "",
    }
    
    cache, err := redis.NewConnection(redisConfig)
    if err != nil {
        panic(err)
    }
    defer cache.Close()
}
```

### 3. IAROS Authentication Library (Go)

**Purpose**: Authentication and authorization utilities

**Features**:
- JWT token management
- OAuth2 integration
- RBAC utilities
- Session management
- API key validation

**Installation**:
```bash
go get github.com/iaros/common/libraries/go/iaros-auth
```

**Usage Example**:
```go
package main

import (
    "github.com/iaros/common/libraries/go/iaros-auth/jwt"
    "github.com/iaros/common/libraries/go/iaros-auth/rbac"
)

func main() {
    // Initialize JWT manager
    jwtManager := jwt.NewManager(jwt.Config{
        SecretKey: "your-secret-key",
        Issuer:    "iaros",
        Expiry:    24 * time.Hour,
    })
    
    // Create token
    claims := jwt.Claims{
        UserID: "user123",
        Email:  "user@example.com",
        Roles:  []string{"admin", "user"},
    }
    
    token, err := jwtManager.GenerateToken(claims)
    if err != nil {
        panic(err)
    }
    
    // Validate token
    validClaims, err := jwtManager.ValidateToken(token)
    if err != nil {
        panic(err)
    }
    
    // Check permissions
    rbacManager := rbac.NewManager()
    canAccess := rbacManager.HasPermission(validClaims.Roles, "pricing:read")
    
    fmt.Printf("User can access pricing data: %v\n", canAccess)
}
```

### 4. IAROS Monitoring Library (Go)

**Purpose**: Monitoring, metrics, and observability

**Features**:
- Prometheus metrics integration
- Distributed tracing
- Health checks
- Performance monitoring
- Alert management

**Installation**:
```bash
go get github.com/iaros/common/libraries/go/iaros-monitoring
```

**Usage Example**:
```go
package main

import (
    "github.com/iaros/common/libraries/go/iaros-monitoring/metrics"
    "github.com/iaros/common/libraries/go/iaros-monitoring/tracing"
)

func main() {
    // Initialize metrics
    metricsConfig := metrics.Config{
        ServiceName: "pricing-service",
        Namespace:   "iaros",
        Prometheus:  true,
    }
    
    metricsManager := metrics.NewManager(metricsConfig)
    
    // Create custom metrics
    requestCounter := metricsManager.NewCounter("requests_total", "Total HTTP requests")
    requestDuration := metricsManager.NewHistogram("request_duration_seconds", "HTTP request duration")
    
    // Initialize tracing
    tracingConfig := tracing.Config{
        ServiceName: "pricing-service",
        JaegerURL:   "http://localhost:14268/api/traces",
    }
    
    tracer := tracing.NewTracer(tracingConfig)
    
    // Example usage in HTTP handler
    http.HandleFunc("/api/price", func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        span := tracer.StartSpan("calculate_price")
        defer span.Finish()
        
        // Your business logic here
        
        requestCounter.Inc()
        requestDuration.Observe(time.Since(start).Seconds())
    })
}
```

## 🐍 Python Libraries

### 1. IAROS Analytics Library (Python)

**Purpose**: Machine learning and analytics utilities

**Features**:
- Model training pipelines
- Feature engineering
- Data preprocessing
- Model evaluation
- Deployment utilities

**Installation**:
```bash
pip install iaros-analytics
```

**Usage Example**:
```python
from iaros_analytics import ModelTrainer, FeatureEngineer
from iaros_analytics.models import DemandForecastModel

# Initialize feature engineer
fe = FeatureEngineer()

# Load and preprocess data
data = fe.load_data("s3://iaros-data/training/demand.csv")
features = fe.extract_features(data, [
    'seasonality', 'trend', 'holidays', 'weather'
])

# Train model
trainer = ModelTrainer(DemandForecastModel)
model = trainer.train(features, validation_split=0.2)

# Evaluate model
metrics = trainer.evaluate(model, test_data)
print(f"Model MAPE: {metrics['mape']:.2f}%")

# Deploy model
trainer.deploy(model, "demand-forecast-v1")
```

### 2. IAROS Data Processing Library (Python)

**Purpose**: Data pipeline and processing utilities

**Features**:
- ETL pipelines
- Data validation
- Schema management
- Data quality checks
- Batch processing

**Installation**:
```bash
pip install iaros-data
```

**Usage Example**:
```python
from iaros_data import Pipeline, DataValidator, SchemaManager

# Define data pipeline
pipeline = Pipeline("booking-data-pipeline")

# Add pipeline stages
pipeline.add_stage("extract", source="s3://iaros-raw/bookings/")
pipeline.add_stage("transform", transformer="booking_transformer")
pipeline.add_stage("validate", validator="booking_validator")
pipeline.add_stage("load", destination="postgresql://iaros-db/bookings")

# Run pipeline
results = pipeline.run()

# Validate data quality
validator = DataValidator()
quality_report = validator.validate(results, [
    "completeness", "uniqueness", "consistency"
])

print(f"Data quality score: {quality_report['score']:.2f}")
```

## 🌐 JavaScript Libraries

### 1. IAROS UI Components (React)

**Purpose**: Reusable UI components for web applications

**Features**:
- Common UI components
- Theming system
- Responsive design
- Accessibility support
- TypeScript support

**Installation**:
```bash
npm install @iaros/ui-components
```

**Usage Example**:
```jsx
import React from 'react';
import { 
  DataTable, 
  PricingChart, 
  BookingForm,
  IAROSThemeProvider 
} from '@iaros/ui-components';

function PricingDashboard() {
  return (
    <IAROSThemeProvider theme="airline">
      <div className="pricing-dashboard">
        <h1>Pricing Dashboard</h1>
        
        <PricingChart 
          data={pricingData}
          timeRange="7d"
          showCompetitor={true}
        />
        
        <DataTable 
          data={routeData}
          columns={[
            { key: 'origin', label: 'Origin' },
            { key: 'destination', label: 'Destination' },
            { key: 'price', label: 'Price' },
            { key: 'demand', label: 'Demand' }
          ]}
          sortable
          filterable
        />
        
        <BookingForm 
          onSubmit={handleBooking}
          ancillaryServices={ancillaryData}
        />
      </div>
    </IAROSThemeProvider>
  );
}
```

### 2. IAROS API Client (JavaScript)

**Purpose**: JavaScript/TypeScript client for IAROS APIs

**Features**:
- Auto-generated from OpenAPI specs
- Request/response interceptors
- Error handling
- Caching support
- TypeScript definitions

**Installation**:
```bash
npm install @iaros/api-client
```

**Usage Example**:
```javascript
import { IAROSClient } from '@iaros/api-client';

// Initialize client
const client = new IAROSClient({
  baseURL: 'https://api.iaros.com',
  apiKey: 'your-api-key',
  timeout: 30000
});

// Use pricing service
async function getPricing() {
  try {
    const response = await client.pricing.calculatePrice({
      origin: 'NYC',
      destination: 'LON',
      departureDate: '2024-06-15',
      passengers: 2,
      class: 'economy'
    });
    
    console.log('Price calculation:', response.data);
  } catch (error) {
    console.error('Pricing error:', error);
  }
}

// Use forecasting service
async function getForecast() {
  try {
    const response = await client.forecasting.getDemandForecast({
      route: 'NYC-LON',
      period: '30d',
      granularity: 'daily'
    });
    
    console.log('Demand forecast:', response.data);
  } catch (error) {
    console.error('Forecasting error:', error);
  }
}
```

## 📋 Library Versioning & Compatibility

### Version Management

All libraries follow semantic versioning (semver):
- `MAJOR.MINOR.PATCH`
- Major: Breaking changes
- Minor: New features, backward compatible
- Patch: Bug fixes, backward compatible

### Compatibility Matrix

| Library | Go Version | Python Version | Node.js Version | Status |
|---------|------------|----------------|-----------------|---------|
| iaros-core | 1.18+ | - | - | ✅ Stable |
| iaros-db | 1.18+ | - | - | ✅ Stable |
| iaros-auth | 1.18+ | - | - | ✅ Stable |
| iaros-monitoring | 1.18+ | - | - | ✅ Stable |
| iaros-analytics | - | 3.8+ | - | ✅ Stable |
| iaros-data | - | 3.8+ | - | ✅ Stable |
| iaros-ui | - | - | 16+ | ✅ Stable |
| iaros-api | - | - | 16+ | ✅ Stable |

### Dependency Management

```yaml
# go.mod example
module github.com/iaros/pricing-service

go 1.19

require (
    github.com/iaros/common/libraries/go/iaros-core v1.2.3
    github.com/iaros/common/libraries/go/iaros-db v1.1.0
    github.com/iaros/common/libraries/go/iaros-auth v1.0.5
    github.com/iaros/common/libraries/go/iaros-monitoring v1.0.2
)
```

```json
// package.json example
{
  "dependencies": {
    "@iaros/ui-components": "^2.1.0",
    "@iaros/api-client": "^1.3.2"
  }
}
```

```txt
# requirements.txt example
iaros-analytics==1.4.0
iaros-data==1.2.1
```

## 🔧 Development Guidelines

### Library Development Standards

1. **Code Quality**
   - 100% test coverage
   - Comprehensive documentation
   - Consistent code style
   - Type safety (where applicable)

2. **API Design**
   - RESTful design principles
   - Consistent error handling
   - Backward compatibility
   - Clear naming conventions

3. **Documentation**
   - API documentation (OpenAPI/Swagger)
   - Usage examples
   - Migration guides
   - Changelog maintenance

### Contributing to Libraries

1. **Fork the library repository**
2. **Create a feature branch**
3. **Implement changes with tests**
4. **Update documentation**
5. **Submit pull request**
6. **Code review process**
7. **Merge and release**

### Testing Requirements

```bash
# Go libraries
go test -v -cover ./...
go test -race ./...

# Python libraries
pytest tests/ --cov=iaros_analytics --cov-report=html

# JavaScript libraries
npm test
npm run test:coverage
```

## 📊 Usage Statistics

### Library Adoption

| Library | Services Using | Monthly Downloads | Last Update |
|---------|----------------|-------------------|-------------|
| iaros-core | 12/16 services | 15,000+ | Jan 2025 |
| iaros-db | 10/16 services | 8,500+ | Jan 2025 |
| iaros-auth | 16/16 services | 12,000+ | Jan 2025 |
| iaros-monitoring | 16/16 services | 10,000+ | Jan 2025 |
| iaros-analytics | 3/16 services | 2,000+ | Jan 2025 |
| iaros-ui | 2/2 frontends | 500+ | Jan 2025 |

### Performance Metrics

- **Library load time**: <100ms average
- **Memory overhead**: <50MB per service
- **API response time**: <10ms average
- **Error rate**: <0.1%

## 📞 Support & Maintenance

### Getting Help

1. **Documentation**: Check library-specific README files
2. **Examples**: Review example implementations
3. **Issues**: Create GitHub issues for bugs
4. **Discussions**: Use GitHub discussions for questions
5. **Slack**: #libraries-support channel

### Maintenance Schedule

- **Security updates**: Immediate
- **Bug fixes**: Weekly releases
- **Feature updates**: Monthly releases
- **Major versions**: Quarterly releases

### Contact Information

- **Library Team**: libraries@iaros.com
- **Emergency Issues**: +1-800-IAROS-LIB
- **Slack Channel**: #libraries-support
- **GitHub**: https://github.com/iaros/libraries

---

**Last Updated**: January 2025  
**Version**: 2.0  
**Next Review**: April 2025
