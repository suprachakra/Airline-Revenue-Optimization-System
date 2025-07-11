#!/bin/bash

# IAROS Performance Testing Script
# Comprehensive performance testing including load, stress, and endurance testing

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
TEST_RESULTS_DIR="$PROJECT_ROOT/test-results/performance/$TIMESTAMP"

# Test configuration
BASE_URL="${BASE_URL:-https://api.iaros.com}"
CONCURRENT_USERS="${CONCURRENT_USERS:-100}"
TEST_DURATION="${TEST_DURATION:-5m}"
RAMP_UP_TIME="${RAMP_UP_TIME:-1m}"
TEST_TYPE="${TEST_TYPE:-load}"

# Test scenarios
RUN_LOAD_TEST=false
RUN_STRESS_TEST=false
RUN_ENDURANCE_TEST=false
RUN_SPIKE_TEST=false
RUN_VOLUME_TEST=false

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
IAROS Performance Testing Script

Usage: $0 [OPTIONS]

Options:
    --load-test             Run load testing
    --stress-test           Run stress testing  
    --endurance-test        Run endurance testing
    --spike-test            Run spike testing
    --volume-test           Run volume testing
    --all-tests             Run all test types
    --base-url URL          Base URL for testing (default: https://api.iaros.com)
    --users COUNT           Number of concurrent users (default: 100)
    --duration TIME         Test duration (default: 5m)
    --ramp-up TIME          Ramp up time (default: 1m)
    --results-dir DIR       Results directory
    -h, --help              Show this help message

Examples:
    $0 --load-test --users 50 --duration 10m
    $0 --stress-test --users 500
    $0 --all-tests --base-url https://staging.iaros.com

EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --load-test)
                RUN_LOAD_TEST=true
                shift
                ;;
            --stress-test)
                RUN_STRESS_TEST=true
                shift
                ;;
            --endurance-test)
                RUN_ENDURANCE_TEST=true
                shift
                ;;
            --spike-test)
                RUN_SPIKE_TEST=true
                shift
                ;;
            --volume-test)
                RUN_VOLUME_TEST=true
                shift
                ;;
            --all-tests)
                RUN_LOAD_TEST=true
                RUN_STRESS_TEST=true
                RUN_ENDURANCE_TEST=true
                RUN_SPIKE_TEST=true
                RUN_VOLUME_TEST=true
                shift
                ;;
            --base-url)
                BASE_URL="$2"
                shift 2
                ;;
            --users)
                CONCURRENT_USERS="$2"
                shift 2
                ;;
            --duration)
                TEST_DURATION="$2"
                shift 2
                ;;
            --ramp-up)
                RAMP_UP_TIME="$2"
                shift 2
                ;;
            --results-dir)
                TEST_RESULTS_DIR="$2"
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
    local required_tools=("curl" "jq" "bc")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            print_error "$tool is required but not installed"
            exit 1
        fi
    done

    # Check for k6 (performance testing tool)
    if ! command -v k6 &> /dev/null; then
        print_info "Installing k6 performance testing tool..."
        case "$OSTYPE" in
            linux*)
                sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
                echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
                sudo apt-get update
                sudo apt-get install k6
                ;;
            darwin*)
                brew install k6
                ;;
            *)
                print_error "Unsupported OS for automatic k6 installation"
                print_info "Please install k6 manually from https://k6.io/docs/getting-started/installation/"
                exit 1
                ;;
        esac
    fi

    # Check for Apache Bench (ab)
    if ! command -v ab &> /dev/null; then
        print_info "Installing Apache Bench..."
        case "$OSTYPE" in
            linux*)
                sudo apt-get update
                sudo apt-get install apache2-utils
                ;;
            darwin*)
                brew install httpd
                ;;
        esac
    fi

    # Create results directory
    mkdir -p "$TEST_RESULTS_DIR"
    
    print_status "Prerequisites check completed"
}

# Function to check service health
check_service_health() {
    print_info "Checking service health before testing..."

    local health_endpoints=(
        "$BASE_URL/health"
        "$BASE_URL/api/v1/health"
        "$BASE_URL/pricing/health"
        "$BASE_URL/forecasting/health"
        "$BASE_URL/offers/health"
        "$BASE_URL/orders/health"
    )

    for endpoint in "${health_endpoints[@]}"; do
        if curl -f -s "$endpoint" > /dev/null 2>&1; then
            print_status "✓ $endpoint is healthy"
        else
            print_warning "⚠ $endpoint is not responding"
        fi
    done
}

# Function to create k6 test script
create_k6_script() {
    local test_name=$1
    local script_file="$TEST_RESULTS_DIR/${test_name}_test.js"

    cat > "$script_file" << EOF
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const httpReqFailed = new Rate('http_req_failed');
const httpReqDuration = new Trend('http_req_duration', true);
const apiCallsCount = new Counter('api_calls_total');

// Test configuration
export let options = {
    stages: [
        { duration: '${RAMP_UP_TIME}', target: ${CONCURRENT_USERS} },
        { duration: '${TEST_DURATION}', target: ${CONCURRENT_USERS} },
        { duration: '${RAMP_UP_TIME}', target: 0 },
    ],
    thresholds: {
        http_req_duration: ['p(95)<2000'], // 95% of requests should be below 2s
        http_req_failed: ['rate<0.1'],     // Error rate should be below 10%
    },
};

// Test data
const testData = {
    flights: [
        { origin: 'JFK', destination: 'LAX', date: '2024-06-01' },
        { origin: 'LAX', destination: 'CHI', date: '2024-06-02' },
        { origin: 'CHI', destination: 'MIA', date: '2024-06-03' },
    ],
    customers: [
        { id: 1, type: 'business', loyalty: 'gold' },
        { id: 2, type: 'leisure', loyalty: 'silver' },
        { id: 3, type: 'business', loyalty: 'platinum' },
    ],
};

// Test scenarios
export default function () {
    let responses = [];
    
    // Scenario 1: Flight Search
    responses.push(searchFlights());
    
    // Scenario 2: Price Calculation
    responses.push(calculatePricing());
    
    // Scenario 3: Offer Generation
    responses.push(generateOffers());
    
    // Scenario 4: Booking Creation
    responses.push(createBooking());
    
    // Scenario 5: Ancillary Services
    responses.push(getAncillaryServices());
    
    // Check responses
    for (let response of responses) {
        if (response) {
            check(response, {
                'status is 200': (r) => r.status === 200,
                'response time < 2000ms': (r) => r.timings.duration < 2000,
                'response has body': (r) => r.body.length > 0,
            });
            
            httpReqFailed.add(response.status !== 200);
            httpReqDuration.add(response.timings.duration);
            apiCallsCount.add(1);
        }
    }
    
    sleep(1);
}

function searchFlights() {
    const flight = testData.flights[Math.floor(Math.random() * testData.flights.length)];
    const params = {
        origin: flight.origin,
        destination: flight.destination,
        departure_date: flight.date,
        passengers: Math.floor(Math.random() * 4) + 1,
    };
    
    return http.get('${BASE_URL}/api/v1/flights/search', { params });
}

function calculatePricing() {
    const payload = {
        flight_id: 'FL' + Math.floor(Math.random() * 1000),
        booking_class: ['Economy', 'Business', 'First'][Math.floor(Math.random() * 3)],
        passengers: Math.floor(Math.random() * 4) + 1,
        advance_purchase: Math.floor(Math.random() * 60) + 1,
    };
    
    return http.post('${BASE_URL}/api/v1/pricing/calculate', JSON.stringify(payload), {
        headers: { 'Content-Type': 'application/json' },
    });
}

function generateOffers() {
    const customer = testData.customers[Math.floor(Math.random() * testData.customers.length)];
    const payload = {
        customer_id: customer.id,
        customer_type: customer.type,
        loyalty_tier: customer.loyalty,
        search_criteria: {
            origin: 'JFK',
            destination: 'LAX',
            date: '2024-06-01',
        },
    };
    
    return http.post('${BASE_URL}/api/v1/offers/generate', JSON.stringify(payload), {
        headers: { 'Content-Type': 'application/json' },
    });
}

function createBooking() {
    const payload = {
        flight_id: 'FL' + Math.floor(Math.random() * 1000),
        customer_id: Math.floor(Math.random() * 1000),
        passengers: [
            {
                first_name: 'John',
                last_name: 'Doe',
                email: 'john.doe@example.com',
                phone: '+1234567890',
            },
        ],
        payment: {
            method: 'credit_card',
            card_number: '4111111111111111',
            expiry: '12/25',
            cvv: '123',
        },
    };
    
    return http.post('${BASE_URL}/api/v1/bookings', JSON.stringify(payload), {
        headers: { 'Content-Type': 'application/json' },
    });
}

function getAncillaryServices() {
    const bookingId = 'BK' + Math.floor(Math.random() * 1000);
    return http.get(\`${BASE_URL}/api/v1/bookings/\${bookingId}/ancillary\`);
}
EOF

    echo "$script_file"
}

# Function to run load test
run_load_test() {
    if [[ "$RUN_LOAD_TEST" != true ]]; then
        return
    fi

    print_info "Running load test..."
    
    local script_file=$(create_k6_script "load")
    local results_file="$TEST_RESULTS_DIR/load_test_results.json"
    
    # Configure load test options
    export K6_OPTIONS='{
        "stages": [
            { "duration": "'$RAMP_UP_TIME'", "target": '$CONCURRENT_USERS' },
            { "duration": "'$TEST_DURATION'", "target": '$CONCURRENT_USERS' },
            { "duration": "'$RAMP_UP_TIME'", "target": 0 }
        ],
        "thresholds": {
            "http_req_duration": ["p(95)<2000"],
            "http_req_failed": ["rate<0.1"]
        }
    }'
    
    # Run k6 test
    k6 run --out json="$results_file" "$script_file"
    
    # Generate load test report
    generate_test_report "load" "$results_file"
    
    print_status "Load test completed"
}

# Function to run stress test
run_stress_test() {
    if [[ "$RUN_STRESS_TEST" != true ]]; then
        return
    fi

    print_info "Running stress test..."
    
    local script_file=$(create_k6_script "stress")
    local results_file="$TEST_RESULTS_DIR/stress_test_results.json"
    
    # Configure stress test with increasing load
    local stress_users=$((CONCURRENT_USERS * 5))
    
    export K6_OPTIONS='{
        "stages": [
            { "duration": "1m", "target": '$CONCURRENT_USERS' },
            { "duration": "2m", "target": '$((CONCURRENT_USERS * 2))' },
            { "duration": "3m", "target": '$((CONCURRENT_USERS * 3))' },
            { "duration": "3m", "target": '$stress_users' },
            { "duration": "2m", "target": '$((CONCURRENT_USERS * 2))' },
            { "duration": "1m", "target": 0 }
        ],
        "thresholds": {
            "http_req_duration": ["p(95)<5000"],
            "http_req_failed": ["rate<0.2"]
        }
    }'
    
    # Run k6 test
    k6 run --out json="$results_file" "$script_file"
    
    # Generate stress test report
    generate_test_report "stress" "$results_file"
    
    print_status "Stress test completed"
}

# Function to run endurance test
run_endurance_test() {
    if [[ "$RUN_ENDURANCE_TEST" != true ]]; then
        return
    fi

    print_info "Running endurance test..."
    
    local script_file=$(create_k6_script "endurance")
    local results_file="$TEST_RESULTS_DIR/endurance_test_results.json"
    
    # Configure endurance test for longer duration
    local endurance_users=$((CONCURRENT_USERS / 2))
    
    export K6_OPTIONS='{
        "stages": [
            { "duration": "5m", "target": '$endurance_users' },
            { "duration": "30m", "target": '$endurance_users' },
            { "duration": "5m", "target": 0 }
        ],
        "thresholds": {
            "http_req_duration": ["p(95)<3000"],
            "http_req_failed": ["rate<0.05"]
        }
    }'
    
    # Run k6 test
    k6 run --out json="$results_file" "$script_file"
    
    # Generate endurance test report
    generate_test_report "endurance" "$results_file"
    
    print_status "Endurance test completed"
}

# Function to run spike test
run_spike_test() {
    if [[ "$RUN_SPIKE_TEST" != true ]]; then
        return
    fi

    print_info "Running spike test..."
    
    local script_file=$(create_k6_script "spike")
    local results_file="$TEST_RESULTS_DIR/spike_test_results.json"
    
    # Configure spike test with sudden load increase
    local spike_users=$((CONCURRENT_USERS * 10))
    
    export K6_OPTIONS='{
        "stages": [
            { "duration": "1m", "target": '$CONCURRENT_USERS' },
            { "duration": "30s", "target": '$spike_users' },
            { "duration": "1m", "target": '$CONCURRENT_USERS' },
            { "duration": "30s", "target": 0 }
        ],
        "thresholds": {
            "http_req_duration": ["p(95)<10000"],
            "http_req_failed": ["rate<0.3"]
        }
    }'
    
    # Run k6 test
    k6 run --out json="$results_file" "$script_file"
    
    # Generate spike test report
    generate_test_report "spike" "$results_file"
    
    print_status "Spike test completed"
}

# Function to run volume test
run_volume_test() {
    if [[ "$RUN_VOLUME_TEST" != true ]]; then
        return
    fi

    print_info "Running volume test..."
    
    # Volume test uses Apache Bench for high-volume requests
    local ab_results_file="$TEST_RESULTS_DIR/volume_test_results.txt"
    local requests_count=$((CONCURRENT_USERS * 1000))
    
    print_info "Testing with $requests_count requests and $CONCURRENT_USERS concurrent users"
    
    # Test multiple endpoints
    local endpoints=(
        "$BASE_URL/health"
        "$BASE_URL/api/v1/flights/search?origin=JFK&destination=LAX&date=2024-06-01"
        "$BASE_URL/api/v1/pricing/calculate"
    )
    
    for endpoint in "${endpoints[@]}"; do
        print_info "Testing endpoint: $endpoint"
        
        ab -n "$requests_count" -c "$CONCURRENT_USERS" -g "$TEST_RESULTS_DIR/volume_$(basename "$endpoint").gnuplot" \
           -e "$TEST_RESULTS_DIR/volume_$(basename "$endpoint").csv" \
           "$endpoint" >> "$ab_results_file" 2>&1
    done
    
    # Generate volume test report
    generate_volume_test_report "$ab_results_file"
    
    print_status "Volume test completed"
}

# Function to generate test report
generate_test_report() {
    local test_type=$1
    local results_file=$2
    local report_file="$TEST_RESULTS_DIR/${test_type}_test_report.html"
    
    print_info "Generating $test_type test report..."
    
    # Parse k6 JSON results
    local avg_duration=$(jq -r '.metrics.http_req_duration.avg' "$results_file" 2>/dev/null || echo "N/A")
    local p95_duration=$(jq -r '.metrics.http_req_duration.p95' "$results_file" 2>/dev/null || echo "N/A")
    local success_rate=$(jq -r '(1 - .metrics.http_req_failed.rate) * 100' "$results_file" 2>/dev/null || echo "N/A")
    local total_requests=$(jq -r '.metrics.http_reqs.count' "$results_file" 2>/dev/null || echo "N/A")
    
    # Create HTML report
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>IAROS ${test_type^} Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin: 20px 0; }
        .metric { background: #fff; border: 1px solid #ddd; padding: 15px; border-radius: 5px; text-align: center; }
        .metric-value { font-size: 24px; font-weight: bold; color: #333; }
        .metric-label { font-size: 14px; color: #666; margin-top: 5px; }
        .pass { color: #4CAF50; }
        .fail { color: #F44336; }
        .warning { color: #FF9800; }
    </style>
</head>
<body>
    <div class="header">
        <h1>IAROS ${test_type^} Test Report</h1>
        <p>Generated on: $(date)</p>
        <p>Base URL: $BASE_URL</p>
        <p>Concurrent Users: $CONCURRENT_USERS</p>
        <p>Test Duration: $TEST_DURATION</p>
    </div>
    
    <div class="metrics">
        <div class="metric">
            <div class="metric-value">$total_requests</div>
            <div class="metric-label">Total Requests</div>
        </div>
        <div class="metric">
            <div class="metric-value">$avg_duration ms</div>
            <div class="metric-label">Average Response Time</div>
        </div>
        <div class="metric">
            <div class="metric-value">$p95_duration ms</div>
            <div class="metric-label">95th Percentile Response Time</div>
        </div>
        <div class="metric">
            <div class="metric-value $(echo "$success_rate > 90" | bc -l >/dev/null 2>&1 && echo "pass" || echo "fail")">$success_rate%</div>
            <div class="metric-label">Success Rate</div>
        </div>
    </div>
    
    <h2>Performance Thresholds</h2>
    <table border="1" style="border-collapse: collapse; width: 100%;">
        <tr>
            <th>Metric</th>
            <th>Threshold</th>
            <th>Actual</th>
            <th>Status</th>
        </tr>
        <tr>
            <td>95th Percentile Response Time</td>
            <td>&lt; 2000ms</td>
            <td>$p95_duration ms</td>
            <td class="$(echo "$p95_duration < 2000" | bc -l >/dev/null 2>&1 && echo "pass" || echo "fail")">$(echo "$p95_duration < 2000" | bc -l >/dev/null 2>&1 && echo "PASS" || echo "FAIL")</td>
        </tr>
        <tr>
            <td>Success Rate</td>
            <td>&gt; 90%</td>
            <td>$success_rate%</td>
            <td class="$(echo "$success_rate > 90" | bc -l >/dev/null 2>&1 && echo "pass" || echo "fail")">$(echo "$success_rate > 90" | bc -l >/dev/null 2>&1 && echo "PASS" || echo "FAIL")</td>
        </tr>
    </table>
    
    <h2>Test Configuration</h2>
    <pre>$(cat "$script_file" | head -20)</pre>
</body>
</html>
EOF

    print_status "Test report generated: $report_file"
}

# Function to generate volume test report
generate_volume_test_report() {
    local results_file=$1
    local report_file="$TEST_RESULTS_DIR/volume_test_report.html"
    
    print_info "Generating volume test report..."
    
    # Parse Apache Bench results
    local requests_per_second=$(grep "Requests per second" "$results_file" | tail -1 | awk '{print $4}')
    local mean_time=$(grep "Time per request" "$results_file" | head -1 | awk '{print $4}')
    local failed_requests=$(grep "Failed requests" "$results_file" | tail -1 | awk '{print $3}')
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>IAROS Volume Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin: 20px 0; }
        .metric { background: #fff; border: 1px solid #ddd; padding: 15px; border-radius: 5px; text-align: center; }
        .metric-value { font-size: 24px; font-weight: bold; color: #333; }
        .metric-label { font-size: 14px; color: #666; margin-top: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>IAROS Volume Test Report</h1>
        <p>Generated on: $(date)</p>
        <p>Base URL: $BASE_URL</p>
        <p>Concurrent Users: $CONCURRENT_USERS</p>
    </div>
    
    <div class="metrics">
        <div class="metric">
            <div class="metric-value">$requests_per_second</div>
            <div class="metric-label">Requests per Second</div>
        </div>
        <div class="metric">
            <div class="metric-value">$mean_time ms</div>
            <div class="metric-label">Mean Response Time</div>
        </div>
        <div class="metric">
            <div class="metric-value">$failed_requests</div>
            <div class="metric-label">Failed Requests</div>
        </div>
    </div>
    
    <h2>Detailed Results</h2>
    <pre>$(cat "$results_file")</pre>
</body>
</html>
EOF

    print_status "Volume test report generated: $report_file"
}

# Function to monitor system resources during tests
monitor_resources() {
    print_info "Starting resource monitoring..."
    
    local monitoring_script="$TEST_RESULTS_DIR/monitor_resources.sh"
    local monitoring_log="$TEST_RESULTS_DIR/resource_monitoring.log"
    
    cat > "$monitoring_script" << 'EOF'
#!/bin/bash
while true; do
    echo "$(date): CPU: $(top -bn1 | grep load | awk '{printf "%.2f%%", $(NF-2)}'), Memory: $(free | grep Mem | awk '{printf "%.2f%%", $3/$2 * 100.0}'), Disk: $(df -h / | awk 'NR==2{print $5}')" >> "$1"
    sleep 5
done
EOF
    
    chmod +x "$monitoring_script"
    "$monitoring_script" "$monitoring_log" &
    MONITOR_PID=$!
    
    print_status "Resource monitoring started (PID: $MONITOR_PID)"
}

# Function to stop resource monitoring
stop_monitoring() {
    if [[ -n "$MONITOR_PID" ]]; then
        kill "$MONITOR_PID" 2>/dev/null || true
        print_status "Resource monitoring stopped"
    fi
}

# Function to generate summary report
generate_summary_report() {
    print_info "Generating summary report..."
    
    local summary_file="$TEST_RESULTS_DIR/performance_test_summary.html"
    
    cat > "$summary_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>IAROS Performance Test Summary</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }
        .test-links { margin: 20px 0; }
        .test-links a { display: block; margin: 10px 0; padding: 10px; background: #e0e0e0; text-decoration: none; border-radius: 3px; }
        .test-links a:hover { background: #d0d0d0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>IAROS Performance Test Summary</h1>
        <p>Generated on: $(date)</p>
        <p>Base URL: $BASE_URL</p>
        <p>Test Results Directory: $TEST_RESULTS_DIR</p>
    </div>
    
    <h2>Test Reports</h2>
    <div class="test-links">
EOF

    # Add links to individual test reports
    for report in "$TEST_RESULTS_DIR"/*_test_report.html; do
        if [[ -f "$report" ]]; then
            local report_name=$(basename "$report" .html)
            echo "        <a href=\"$report_name.html\">$report_name</a>" >> "$summary_file"
        fi
    done
    
    cat >> "$summary_file" << EOF
    </div>
    
    <h2>Files Generated</h2>
    <ul>
$(find "$TEST_RESULTS_DIR" -name "*.html" -o -name "*.json" -o -name "*.csv" -o -name "*.log" | sed 's|'"$TEST_RESULTS_DIR"'/||' | sort | sed 's/^/        <li>/' | sed 's/$/<\/li>/')
    </ul>
</body>
</html>
EOF

    print_status "Summary report generated: $summary_file"
}

# Main function
main() {
    echo "🚀 IAROS Performance Testing"
    echo "============================"
    
    parse_args "$@"
    
    # Set default if no tests specified
    if [[ "$RUN_LOAD_TEST" == false && "$RUN_STRESS_TEST" == false && "$RUN_ENDURANCE_TEST" == false && "$RUN_SPIKE_TEST" == false && "$RUN_VOLUME_TEST" == false ]]; then
        RUN_LOAD_TEST=true
    fi
    
    check_prerequisites
    check_service_health
    
    # Start resource monitoring
    monitor_resources
    
    # Run selected tests
    run_load_test
    run_stress_test
    run_endurance_test
    run_spike_test
    run_volume_test
    
    # Stop resource monitoring
    stop_monitoring
    
    # Generate summary report
    generate_summary_report
    
    print_status "Performance testing completed!"
    print_info "Results available in: $TEST_RESULTS_DIR"
    print_info "Summary report: $TEST_RESULTS_DIR/performance_test_summary.html"
}

# Error handling
trap 'print_error "Performance testing failed at line $LINENO"; stop_monitoring' ERR

# Execute main function
main "$@" 