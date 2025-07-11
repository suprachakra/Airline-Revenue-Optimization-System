#!/bin/bash

# IAROS Cleanup Script
# Comprehensive cleanup of temporary files, logs, caches, and development artifacts

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
DRY_RUN=false
CLEAN_LOGS=false
CLEAN_CACHE=false
CLEAN_TEMP=false
CLEAN_CONTAINERS=false
CLEAN_VOLUMES=false
CLEAN_IMAGES=false
CLEAN_NODE_MODULES=false
CLEAN_GO_CACHE=false
CLEAN_PYTHON_CACHE=false
FORCE=false

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
IAROS Cleanup Script

Usage: $0 [OPTIONS]

Options:
    --dry-run              Show what would be cleaned without doing it
    --logs                 Clean old log files
    --cache                Clean application caches
    --temp                 Clean temporary files
    --containers           Clean Docker containers
    --volumes              Clean Docker volumes
    --images               Clean Docker images
    --node-modules         Clean Node.js modules
    --go-cache             Clean Go build cache
    --python-cache         Clean Python cache
    --all                  Clean everything
    --force                Force cleanup without confirmation
    --help                 Show this help message

Examples:
    $0 --dry-run --all              # See what would be cleaned
    $0 --logs --temp                # Clean logs and temp files
    $0 --containers --volumes       # Clean Docker containers and volumes

EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --logs)
                CLEAN_LOGS=true
                shift
                ;;
            --cache)
                CLEAN_CACHE=true
                shift
                ;;
            --temp)
                CLEAN_TEMP=true
                shift
                ;;
            --containers)
                CLEAN_CONTAINERS=true
                shift
                ;;
            --volumes)
                CLEAN_VOLUMES=true
                shift
                ;;
            --images)
                CLEAN_IMAGES=true
                shift
                ;;
            --node-modules)
                CLEAN_NODE_MODULES=true
                shift
                ;;
            --go-cache)
                CLEAN_GO_CACHE=true
                shift
                ;;
            --python-cache)
                CLEAN_PYTHON_CACHE=true
                shift
                ;;
            --all)
                CLEAN_LOGS=true
                CLEAN_CACHE=true
                CLEAN_TEMP=true
                CLEAN_CONTAINERS=true
                CLEAN_VOLUMES=true
                CLEAN_IMAGES=true
                CLEAN_NODE_MODULES=true
                CLEAN_GO_CACHE=true
                CLEAN_PYTHON_CACHE=true
                shift
                ;;
            --force)
                FORCE=true
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

# Function to get directory size
get_dir_size() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        du -sh "$dir" 2>/dev/null | cut -f1 || echo "0"
    else
        echo "0"
    fi
}

# Function to confirm cleanup
confirm_cleanup() {
    if [[ "$FORCE" == true || "$DRY_RUN" == true ]]; then
        return 0
    fi
    
    echo
    print_warning "This will permanently delete files and free up disk space."
    read -p "Are you sure you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Cleanup cancelled"
        exit 0
    fi
}

# Function to clean logs
clean_logs() {
    if [[ "$CLEAN_LOGS" != true ]]; then
        return
    fi
    
    print_info "Cleaning old log files..."
    
    local log_dirs=(
        "/var/log/iaros"
        "$PROJECT_ROOT/logs"
        "/opt/iaros/logs"
        "$HOME/.iaros/logs"
    )
    
    local total_freed=0
    
    for log_dir in "${log_dirs[@]}"; do
        if [[ -d "$log_dir" ]]; then
            local size_before=$(get_dir_size "$log_dir")
            print_info "Checking $log_dir (size: $size_before)"
            
            # Remove logs older than 30 days
            local old_logs=$(find "$log_dir" -name "*.log" -mtime +30 -type f 2>/dev/null | wc -l)
            if [[ "$old_logs" -gt 0 ]]; then
                print_info "Found $old_logs old log files"
                
                if [[ "$DRY_RUN" == false ]]; then
                    find "$log_dir" -name "*.log" -mtime +30 -type f -delete
                fi
                
                local size_after=$(get_dir_size "$log_dir")
                print_status "Cleaned $log_dir - freed space"
            else
                print_info "No old logs found in $log_dir"
            fi
        fi
    done
    
    # Clean compressed logs
    find /var/log -name "*.gz" -mtime +7 -type f ${DRY_RUN:+-print} ${DRY_RUN:-'-delete'} 2>/dev/null || true
    
    print_status "Log cleanup completed"
}

# Function to clean cache
clean_cache() {
    if [[ "$CLEAN_CACHE" != true ]]; then
        return
    fi
    
    print_info "Cleaning application caches..."
    
    local cache_dirs=(
        "$PROJECT_ROOT/.cache"
        "$PROJECT_ROOT/frontend/web-portal/.cache"
        "$PROJECT_ROOT/frontend/mobile-app/.cache"
        "$HOME/.cache/iaros"
        "/tmp/iaros-cache"
    )
    
    for cache_dir in "${cache_dirs[@]}"; do
        if [[ -d "$cache_dir" ]]; then
            local size=$(get_dir_size "$cache_dir")
            print_info "Cleaning $cache_dir (size: $size)"
            
            if [[ "$DRY_RUN" == false ]]; then
                rm -rf "$cache_dir"
            fi
            
            print_status "Cleaned $cache_dir"
        fi
    done
    
    # Clean Redis cache (if local development)
    if command -v redis-cli &> /dev/null; then
        local redis_memory=$(redis-cli info memory | grep used_memory_human | cut -d: -f2)
        print_info "Redis memory usage: $redis_memory"
        
        if [[ "$DRY_RUN" == false ]]; then
            redis-cli flushdb >/dev/null 2>&1 || true
            print_status "Redis cache flushed"
        fi
    fi
    
    print_status "Cache cleanup completed"
}

# Function to clean temporary files
clean_temp() {
    if [[ "$CLEAN_TEMP" != true ]]; then
        return
    fi
    
    print_info "Cleaning temporary files..."
    
    local temp_patterns=(
        "$PROJECT_ROOT/tmp/*"
        "$PROJECT_ROOT/temp/*"
        "$PROJECT_ROOT/.tmp/*"
        "$PROJECT_ROOT/**/*.tmp"
        "$PROJECT_ROOT/**/*.temp"
        "$PROJECT_ROOT/**/*~"
        "$PROJECT_ROOT/**/core"
        "$PROJECT_ROOT/**/.DS_Store"
        "$PROJECT_ROOT/**/Thumbs.db"
    )
    
    for pattern in "${temp_patterns[@]}"; do
        if [[ "$DRY_RUN" == true ]]; then
            find "$PROJECT_ROOT" -name "$(basename "$pattern")" -type f 2>/dev/null | head -5
        else
            find "$PROJECT_ROOT" -name "$(basename "$pattern")" -type f -delete 2>/dev/null || true
        fi
    done
    
    # Clean system temp files
    if [[ "$DRY_RUN" == false ]]; then
        find /tmp -name "iaros*" -type f -mtime +1 -delete 2>/dev/null || true
        find /var/tmp -name "iaros*" -type f -mtime +1 -delete 2>/dev/null || true
    fi
    
    print_status "Temporary files cleanup completed"
}

# Function to clean Docker containers
clean_containers() {
    if [[ "$CLEAN_CONTAINERS" != true ]]; then
        return
    fi
    
    if ! command -v docker &> /dev/null; then
        print_warning "Docker not found, skipping container cleanup"
        return
    fi
    
    print_info "Cleaning Docker containers..."
    
    # Stop and remove exited containers
    local exited_containers=$(docker ps -a -q -f status=exited | wc -l)
    if [[ "$exited_containers" -gt 0 ]]; then
        print_info "Found $exited_containers exited containers"
        
        if [[ "$DRY_RUN" == false ]]; then
            docker container prune -f
        fi
        
        print_status "Removed exited containers"
    fi
    
    # Remove dangling containers
    local dangling_containers=$(docker ps -a -q -f dangling=true | wc -l)
    if [[ "$dangling_containers" -gt 0 ]]; then
        print_info "Found $dangling_containers dangling containers"
        
        if [[ "$DRY_RUN" == false ]]; then
            docker ps -a -q -f dangling=true | xargs -r docker rm
        fi
    fi
    
    print_status "Docker container cleanup completed"
}

# Function to clean Docker volumes
clean_volumes() {
    if [[ "$CLEAN_VOLUMES" != true ]]; then
        return
    fi
    
    if ! command -v docker &> /dev/null; then
        print_warning "Docker not found, skipping volume cleanup"
        return
    fi
    
    print_info "Cleaning Docker volumes..."
    
    # Remove dangling volumes
    local dangling_volumes=$(docker volume ls -q -f dangling=true | wc -l)
    if [[ "$dangling_volumes" -gt 0 ]]; then
        print_info "Found $dangling_volumes dangling volumes"
        
        if [[ "$DRY_RUN" == false ]]; then
            docker volume prune -f
        fi
        
        print_status "Removed dangling volumes"
    fi
    
    print_status "Docker volume cleanup completed"
}

# Function to clean Docker images
clean_images() {
    if [[ "$CLEAN_IMAGES" != true ]]; then
        return
    fi
    
    if ! command -v docker &> /dev/null; then
        print_warning "Docker not found, skipping image cleanup"
        return
    fi
    
    print_info "Cleaning Docker images..."
    
    # Remove dangling images
    local dangling_images=$(docker images -q -f dangling=true | wc -l)
    if [[ "$dangling_images" -gt 0 ]]; then
        print_info "Found $dangling_images dangling images"
        
        if [[ "$DRY_RUN" == false ]]; then
            docker image prune -f
        fi
        
        print_status "Removed dangling images"
    fi
    
    # Remove unused images
    if [[ "$DRY_RUN" == false ]]; then
        docker image prune -a -f --filter "until=24h" 2>/dev/null || true
    fi
    
    print_status "Docker image cleanup completed"
}

# Function to clean Node.js modules
clean_node_modules() {
    if [[ "$CLEAN_NODE_MODULES" != true ]]; then
        return
    fi
    
    print_info "Cleaning Node.js modules..."
    
    # Find node_modules directories
    local node_modules_dirs=$(find "$PROJECT_ROOT" -name "node_modules" -type d 2>/dev/null)
    
    if [[ -n "$node_modules_dirs" ]]; then
        echo "$node_modules_dirs" | while read dir; do
            local size=$(get_dir_size "$dir")
            print_info "Found node_modules: $dir (size: $size)"
            
            if [[ "$DRY_RUN" == false ]]; then
                rm -rf "$dir"
            fi
        done
        
        print_status "Node.js modules cleanup completed"
        print_info "Run 'npm install' to reinstall dependencies"
    else
        print_info "No node_modules directories found"
    fi
}

# Function to clean Go cache
clean_go_cache() {
    if [[ "$CLEAN_GO_CACHE" != true ]]; then
        return
    fi
    
    if ! command -v go &> /dev/null; then
        print_warning "Go not found, skipping Go cache cleanup"
        return
    fi
    
    print_info "Cleaning Go cache..."
    
    # Clean Go module cache
    if [[ "$DRY_RUN" == false ]]; then
        go clean -modcache
        go clean -cache
    fi
    
    # Clean build artifacts
    local build_artifacts=$(find "$PROJECT_ROOT" -name "main" -type f -executable 2>/dev/null | wc -l)
    if [[ "$build_artifacts" -gt 0 ]]; then
        print_info "Found $build_artifacts build artifacts"
        
        if [[ "$DRY_RUN" == false ]]; then
            find "$PROJECT_ROOT" -name "main" -type f -executable -delete
        fi
    fi
    
    print_status "Go cache cleanup completed"
}

# Function to clean Python cache
clean_python_cache() {
    if [[ "$CLEAN_PYTHON_CACHE" != true ]]; then
        return
    fi
    
    print_info "Cleaning Python cache..."
    
    # Clean __pycache__ directories
    local pycache_dirs=$(find "$PROJECT_ROOT" -name "__pycache__" -type d 2>/dev/null)
    
    if [[ -n "$pycache_dirs" ]]; then
        echo "$pycache_dirs" | while read dir; do
            local size=$(get_dir_size "$dir")
            print_info "Cleaning $dir (size: $size)"
            
            if [[ "$DRY_RUN" == false ]]; then
                rm -rf "$dir"
            fi
        done
    fi
    
    # Clean .pyc files
    local pyc_files=$(find "$PROJECT_ROOT" -name "*.pyc" -type f 2>/dev/null | wc -l)
    if [[ "$pyc_files" -gt 0 ]]; then
        print_info "Found $pyc_files .pyc files"
        
        if [[ "$DRY_RUN" == false ]]; then
            find "$PROJECT_ROOT" -name "*.pyc" -type f -delete
        fi
    fi
    
    print_status "Python cache cleanup completed"
}

# Function to display cleanup summary
display_summary() {
    print_info "Cleanup Summary:"
    echo
    
    if [[ "$DRY_RUN" == true ]]; then
        print_warning "DRY RUN - No files were actually deleted"
    else
        print_status "Cleanup completed successfully"
    fi
    
    # Show current disk usage
    local disk_usage=$(df -h "$PROJECT_ROOT" | tail -1 | awk '{print $5}')
    print_info "Current disk usage: $disk_usage"
    
    # Show largest directories
    print_info "Largest directories in project:"
    du -sh "$PROJECT_ROOT"/* 2>/dev/null | sort -hr | head -5
}

# Main function
main() {
    echo "🧹 IAROS Cleanup Script"
    echo "======================="
    
    parse_args "$@"
    
    # Default to temp cleanup if no specific options
    if [[ "$CLEAN_LOGS" == false && "$CLEAN_CACHE" == false && "$CLEAN_TEMP" == false && "$CLEAN_CONTAINERS" == false && "$CLEAN_VOLUMES" == false && "$CLEAN_IMAGES" == false && "$CLEAN_NODE_MODULES" == false && "$CLEAN_GO_CACHE" == false && "$CLEAN_PYTHON_CACHE" == false ]]; then
        CLEAN_TEMP=true
    fi
    
    if [[ "$DRY_RUN" == true ]]; then
        print_info "DRY RUN MODE - No files will be deleted"
    fi
    
    confirm_cleanup
    
    # Run cleanup tasks
    clean_logs
    clean_cache
    clean_temp
    clean_containers
    clean_volumes
    clean_images
    clean_node_modules
    clean_go_cache
    clean_python_cache
    
    display_summary
}

# Error handling
trap 'print_error "Cleanup failed at line $LINENO"' ERR

# Execute main function
main "$@" 