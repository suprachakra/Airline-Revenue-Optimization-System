#!/bin/bash

# IAROS Log Analysis Script
# Comprehensive log analysis, parsing, and reporting

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
ANALYSIS_DIR="$PROJECT_ROOT/log-analysis/$TIMESTAMP"

# Default values
LOG_DIR="/var/log/iaros"
TIME_RANGE="24h"
ANALYSIS_TYPE="all"
OUTPUT_FORMAT="html"
REAL_TIME=false

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
IAROS Log Analysis Script

Usage: $0 [OPTIONS]

Options:
    --log-dir DIR           Log directory (default: /var/log/iaros)
    --time-range RANGE      Time range (1h, 24h, 7d, 30d)
    --analysis-type TYPE    Analysis type (errors, performance, security, all)
    --output-format FORMAT  Output format (html, json, csv)
    --real-time            Real-time log monitoring
    --help                  Show this help message

Examples:
    $0 --time-range 1h --analysis-type errors
    $0 --log-dir /opt/iaros/logs --output-format json
    $0 --real-time

EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --log-dir)
                LOG_DIR="$2"
                shift 2
                ;;
            --time-range)
                TIME_RANGE="$2"
                shift 2
                ;;
            --analysis-type)
                ANALYSIS_TYPE="$2"
                shift 2
                ;;
            --output-format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            --real-time)
                REAL_TIME=true
                shift
                ;;
            --help)
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
    local required_tools=("awk" "grep" "sed" "jq" "sort" "uniq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            print_error "$tool is required but not installed"
            exit 1
        fi
    done

    # Check log directory
    if [[ ! -d "$LOG_DIR" ]]; then
        print_error "Log directory not found: $LOG_DIR"
        exit 1
    fi

    # Create analysis directory
    mkdir -p "$ANALYSIS_DIR"
    
    print_status "Prerequisites check completed"
}

# Function to find log files
find_log_files() {
    local time_option=""
    
    case "$TIME_RANGE" in
        "1h")
            time_option="-mtime -0.042"  # 1 hour
            ;;
        "24h")
            time_option="-mtime -1"      # 24 hours
            ;;
        "7d")
            time_option="-mtime -7"      # 7 days
            ;;
        "30d")
            time_option="-mtime -30"     # 30 days
            ;;
        *)
            time_option="-mtime -1"      # Default to 24 hours
            ;;
    esac
    
    # Find log files based on time range
    find "$LOG_DIR" -name "*.log" $time_option -type f > "$ANALYSIS_DIR/log_files.txt"
    
    local file_count=$(wc -l < "$ANALYSIS_DIR/log_files.txt")
    print_info "Found $file_count log files for analysis"
}

# Function to analyze errors
analyze_errors() {
    print_info "Analyzing errors..."
    
    local error_report="$ANALYSIS_DIR/error_analysis.txt"
    
    # Extract error patterns
    grep -i "error\|exception\|fail\|critical" $(cat "$ANALYSIS_DIR/log_files.txt") | \
    awk '{print $1 " " $2 " " $3 " " $4 " " $5}' | \
    sort | uniq -c | sort -nr > "$error_report"
    
    # Generate error summary
    local total_errors=$(wc -l < "$error_report")
    print_status "Found $total_errors unique error patterns"
    
    # Top 10 errors
    head -10 "$error_report" > "$ANALYSIS_DIR/top_errors.txt"
}

# Function to analyze performance
analyze_performance() {
    print_info "Analyzing performance..."
    
    local perf_report="$ANALYSIS_DIR/performance_analysis.txt"
    
    # Extract response times
    grep -E "response_time|duration|latency" $(cat "$ANALYSIS_DIR/log_files.txt") | \
    awk '{for(i=1;i<=NF;i++) if($i ~ /[0-9]+ms/) print $i}' | \
    sed 's/ms//' | sort -n > "$ANALYSIS_DIR/response_times.txt"
    
    # Calculate performance metrics
    if [[ -s "$ANALYSIS_DIR/response_times.txt" ]]; then
        local avg_time=$(awk '{sum+=$1} END {print sum/NR}' "$ANALYSIS_DIR/response_times.txt")
        local p95_time=$(awk '{all[NR] = $1} END{print all[int(NR*0.95)]}' "$ANALYSIS_DIR/response_times.txt")
        
        echo "Average Response Time: ${avg_time}ms" > "$perf_report"
        echo "95th Percentile: ${p95_time}ms" >> "$perf_report"
    fi
    
    print_status "Performance analysis completed"
}

# Function to analyze security events
analyze_security() {
    print_info "Analyzing security events..."
    
    local security_report="$ANALYSIS_DIR/security_analysis.txt"
    
    # Extract security-related events
    grep -iE "authentication|authorization|login|logout|access denied|forbidden|unauthorized" \
    $(cat "$ANALYSIS_DIR/log_files.txt") | \
    awk '{print $1 " " $2 " " $3 " " $4 " " $5}' | \
    sort | uniq -c | sort -nr > "$security_report"
    
    # Detect potential security issues
    grep -i "failed login\|brute force\|suspicious" $(cat "$ANALYSIS_DIR/log_files.txt") \
    > "$ANALYSIS_DIR/security_alerts.txt" 2>/dev/null || true
    
    local security_alerts=$(wc -l < "$ANALYSIS_DIR/security_alerts.txt")
    print_status "Found $security_alerts potential security alerts"
}

# Function to generate HTML report
generate_html_report() {
    print_info "Generating HTML report..."
    
    local html_report="$ANALYSIS_DIR/log_analysis_report.html"
    
    cat > "$html_report" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>IAROS Log Analysis Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; }
        .error { color: #d32f2f; }
        .warning { color: #f57c00; }
        .info { color: #1976d2; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .chart { width: 100%; height: 400px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>IAROS Log Analysis Report</h1>
        <p>Generated on: $(date)</p>
        <p>Time Range: $TIME_RANGE</p>
        <p>Log Directory: $LOG_DIR</p>
    </div>
    
    <div class="section">
        <h2>Summary</h2>
        <p>Total Log Files: $(wc -l < "$ANALYSIS_DIR/log_files.txt")</p>
        <p>Analysis Type: $ANALYSIS_TYPE</p>
    </div>
    
    <div class="section">
        <h2>Top Errors</h2>
        <table>
            <tr><th>Count</th><th>Error Pattern</th></tr>
$(head -10 "$ANALYSIS_DIR/top_errors.txt" 2>/dev/null | while read line; do
    echo "            <tr><td>$(echo "$line" | awk '{print $1}')</td><td>$(echo "$line" | cut -d' ' -f2-)</td></tr>"
done)
        </table>
    </div>
    
    <div class="section">
        <h2>Performance Metrics</h2>
        <pre>$(cat "$ANALYSIS_DIR/performance_analysis.txt" 2>/dev/null || echo "No performance data available")</pre>
    </div>
    
    <div class="section">
        <h2>Security Events</h2>
        <pre>$(head -20 "$ANALYSIS_DIR/security_analysis.txt" 2>/dev/null || echo "No security events found")</pre>
    </div>
</body>
</html>
EOF

    print_status "HTML report generated: $html_report"
}

# Function for real-time monitoring
real_time_monitor() {
    print_info "Starting real-time log monitoring..."
    
    # Monitor latest log files
    local latest_logs=$(find "$LOG_DIR" -name "*.log" -type f -exec ls -lt {} + | head -5 | awk '{print $9}')
    
    print_info "Monitoring files: $latest_logs"
    
    # Use tail to follow logs and analyze in real-time
    tail -f $latest_logs | while read line; do
        # Check for errors
        if echo "$line" | grep -iq "error\|exception\|fail"; then
            print_error "ERROR: $line"
        fi
        
        # Check for warnings
        if echo "$line" | grep -iq "warning\|warn"; then
            print_warning "WARNING: $line"
        fi
        
        # Check for security events
        if echo "$line" | grep -iq "authentication\|login\|access denied"; then
            print_info "SECURITY: $line"
        fi
    done
}

# Main function
main() {
    echo "📊 IAROS Log Analysis"
    echo "===================="
    
    parse_args "$@"
    
    if [[ "$REAL_TIME" == true ]]; then
        real_time_monitor
        return
    fi
    
    check_prerequisites
    find_log_files
    
    # Run analysis based on type
    if [[ "$ANALYSIS_TYPE" == "all" || "$ANALYSIS_TYPE" == "errors" ]]; then
        analyze_errors
    fi
    
    if [[ "$ANALYSIS_TYPE" == "all" || "$ANALYSIS_TYPE" == "performance" ]]; then
        analyze_performance
    fi
    
    if [[ "$ANALYSIS_TYPE" == "all" || "$ANALYSIS_TYPE" == "security" ]]; then
        analyze_security
    fi
    
    # Generate reports
    case "$OUTPUT_FORMAT" in
        "html")
            generate_html_report
            ;;
        "json")
            print_info "JSON report generation not implemented yet"
            ;;
        "csv")
            print_info "CSV report generation not implemented yet"
            ;;
    esac
    
    print_status "Log analysis completed!"
    print_info "Results available in: $ANALYSIS_DIR"
}

# Error handling
trap 'print_error "Log analysis failed at line $LINENO"' ERR

# Execute main function
main "$@" 