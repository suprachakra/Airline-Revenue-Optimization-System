#!/bin/bash

# IAROS Backup Script
# Comprehensive backup solution for databases, configurations, and application data

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
BACKUP_ROOT="${BACKUP_ROOT:-/opt/iaros/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"

# Backup options
BACKUP_DATABASES=true
BACKUP_CONFIGS=true
BACKUP_LOGS=true
BACKUP_CERTIFICATES=true
ENCRYPT_BACKUP=true
COMPRESS_BACKUP=true
VERIFY_BACKUP=true

# Database credentials (from environment or config)
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-iaros}"
DB_USER="${DB_USER:-iaros_user}"
DB_PASSWORD="${DB_PASSWORD:-iaros_pass}"

REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"

MONGO_HOST="${MONGO_HOST:-localhost}"
MONGO_PORT="${MONGO_PORT:-27017}"
MONGO_DB="${MONGO_DB:-iaros}"
MONGO_USER="${MONGO_USER:-iaros_user}"
MONGO_PASSWORD="${MONGO_PASSWORD:-iaros_pass}"

# S3 Configuration for remote backups
S3_BUCKET="${S3_BUCKET:-iaros-backups}"
S3_REGION="${S3_REGION:-us-east-1}"

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
IAROS Backup Script

Usage: $0 [OPTIONS]

Options:
    --db-only              Backup only databases
    --config-only          Backup only configurations
    --no-encrypt           Skip encryption of backups
    --no-compress          Skip compression of backups
    --no-verify            Skip backup verification
    --retention DAYS       Set retention period (default: 30 days)
    --backup-dir DIR       Set backup directory (default: /opt/iaros/backups)
    --s3-upload            Upload backups to S3
    --restore FILE         Restore from backup file
    -h, --help             Show this help message

Examples:
    $0                                    # Full backup with all options
    $0 --db-only                          # Database only backup
    $0 --s3-upload                        # Backup and upload to S3
    $0 --restore backup_20240115_143022   # Restore from backup

EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --db-only)
                BACKUP_CONFIGS=false
                BACKUP_LOGS=false
                BACKUP_CERTIFICATES=false
                shift
                ;;
            --config-only)
                BACKUP_DATABASES=false
                BACKUP_LOGS=false
                shift
                ;;
            --no-encrypt)
                ENCRYPT_BACKUP=false
                shift
                ;;
            --no-compress)
                COMPRESS_BACKUP=false
                shift
                ;;
            --no-verify)
                VERIFY_BACKUP=false
                shift
                ;;
            --retention)
                RETENTION_DAYS="$2"
                shift 2
                ;;
            --backup-dir)
                BACKUP_ROOT="$2"
                shift 2
                ;;
            --s3-upload)
                UPLOAD_TO_S3=true
                shift
                ;;
            --restore)
                RESTORE_MODE=true
                RESTORE_FILE="$2"
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
    print_info "Checking backup prerequisites..."

    # Check required tools
    local required_tools=("pg_dump" "redis-cli" "mongodump" "tar")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            print_error "$tool is required but not installed"
            exit 1
        fi
    done

    # Check if encryption is requested and GPG is available
    if [[ "$ENCRYPT_BACKUP" == true ]] && ! command -v gpg &> /dev/null; then
        print_warning "GPG not found, disabling encryption"
        ENCRYPT_BACKUP=false
    fi

    # Check if S3 upload is requested and AWS CLI is available
    if [[ "$UPLOAD_TO_S3" == true ]] && ! command -v aws &> /dev/null; then
        print_warning "AWS CLI not found, disabling S3 upload"
        UPLOAD_TO_S3=false
    fi

    # Create backup directory
    mkdir -p "$BACKUP_ROOT"
    
    print_status "Prerequisites check completed"
}

# Function to test database connections
test_connections() {
    print_info "Testing database connections..."

    # Test PostgreSQL connection
    if [[ "$BACKUP_DATABASES" == true ]]; then
        if ! PGPASSWORD="$DB_PASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME"; then
            print_error "Cannot connect to PostgreSQL database"
            exit 1
        fi
        print_status "PostgreSQL connection successful"

        # Test Redis connection
        if ! redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping > /dev/null; then
            print_error "Cannot connect to Redis"
            exit 1
        fi
        print_status "Redis connection successful"

        # Test MongoDB connection
        if ! mongosh --host "$MONGO_HOST:$MONGO_PORT" --username "$MONGO_USER" --password "$MONGO_PASSWORD" --authenticationDatabase admin --eval "db.adminCommand('ping')" > /dev/null; then
            print_warning "Cannot connect to MongoDB, skipping MongoDB backup"
        else
            print_status "MongoDB connection successful"
        fi
    fi
}

# Function to backup PostgreSQL database
backup_postgresql() {
    print_info "Backing up PostgreSQL database..."

    local backup_file="$BACKUP_DIR/postgresql_${TIMESTAMP}.sql"
    
    PGPASSWORD="$DB_PASSWORD" pg_dump \
        --host="$DB_HOST" \
        --port="$DB_PORT" \
        --username="$DB_USER" \
        --dbname="$DB_NAME" \
        --format=custom \
        --compress=9 \
        --no-owner \
        --no-privileges \
        --file="$backup_file"

    if [[ -f "$backup_file" ]]; then
        local file_size=$(du -h "$backup_file" | cut -f1)
        print_status "PostgreSQL backup completed: $backup_file ($file_size)"
        echo "$backup_file" >> "$BACKUP_DIR/backup_manifest.txt"
    else
        print_error "PostgreSQL backup failed"
        exit 1
    fi
}

# Function to backup Redis data
backup_redis() {
    print_info "Backing up Redis data..."

    local backup_file="$BACKUP_DIR/redis_${TIMESTAMP}.rdb"
    
    # Create Redis backup using BGSAVE
    redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" BGSAVE
    
    # Wait for backup to complete
    while [[ $(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" LASTSAVE) == $(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" LASTSAVE) ]]; do
        sleep 1
    done
    
    # Copy the dump file
    local redis_dump_path="/var/lib/redis/dump.rdb"
    if [[ -f "$redis_dump_path" ]]; then
        cp "$redis_dump_path" "$backup_file"
        print_status "Redis backup completed: $backup_file"
        echo "$backup_file" >> "$BACKUP_DIR/backup_manifest.txt"
    else
        print_warning "Redis dump file not found, creating alternative backup"
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" --rdb "$backup_file"
        print_status "Redis backup completed using --rdb: $backup_file"
        echo "$backup_file" >> "$BACKUP_DIR/backup_manifest.txt"
    fi
}

# Function to backup MongoDB data
backup_mongodb() {
    print_info "Backing up MongoDB data..."

    local backup_dir="$BACKUP_DIR/mongodb_${TIMESTAMP}"
    
    mongodump \
        --host "$MONGO_HOST:$MONGO_PORT" \
        --username "$MONGO_USER" \
        --password "$MONGO_PASSWORD" \
        --authenticationDatabase admin \
        --db "$MONGO_DB" \
        --out "$backup_dir"

    if [[ -d "$backup_dir" ]]; then
        # Create tar archive of MongoDB backup
        tar -czf "$backup_dir.tar.gz" -C "$(dirname "$backup_dir")" "$(basename "$backup_dir")"
        rm -rf "$backup_dir"
        
        local file_size=$(du -h "$backup_dir.tar.gz" | cut -f1)
        print_status "MongoDB backup completed: $backup_dir.tar.gz ($file_size)"
        echo "$backup_dir.tar.gz" >> "$BACKUP_DIR/backup_manifest.txt"
    else
        print_error "MongoDB backup failed"
        exit 1
    fi
}

# Function to backup configurations
backup_configurations() {
    print_info "Backing up configurations..."

    local config_backup="$BACKUP_DIR/configurations_${TIMESTAMP}.tar.gz"
    local config_dirs=(
        "$PROJECT_ROOT/infrastructure/config"
        "$PROJECT_ROOT/infrastructure/k8s"
        "$PROJECT_ROOT/services/*/config.yaml"
        "$PROJECT_ROOT/.env*"
    )

    # Create temporary directory for configurations
    local temp_config_dir="$BACKUP_DIR/temp_configs"
    mkdir -p "$temp_config_dir"

    # Copy configuration files
    for config_path in "${config_dirs[@]}"; do
        if [[ -e $config_path ]]; then
            cp -r $config_path "$temp_config_dir/" 2>/dev/null || true
        fi
    done

    # Create tar archive
    if [[ -d "$temp_config_dir" ]] && [[ "$(ls -A "$temp_config_dir")" ]]; then
        tar -czf "$config_backup" -C "$BACKUP_DIR" temp_configs
        rm -rf "$temp_config_dir"
        
        local file_size=$(du -h "$config_backup" | cut -f1)
        print_status "Configuration backup completed: $config_backup ($file_size)"
        echo "$config_backup" >> "$BACKUP_DIR/backup_manifest.txt"
    else
        print_warning "No configuration files found to backup"
        rm -rf "$temp_config_dir"
    fi
}

# Function to backup logs
backup_logs() {
    print_info "Backing up logs..."

    local log_backup="$BACKUP_DIR/logs_${TIMESTAMP}.tar.gz"
    local log_dirs=(
        "/var/log/iaros"
        "$PROJECT_ROOT/logs"
        "/opt/iaros/logs"
    )

    # Create temporary directory for logs
    local temp_log_dir="$BACKUP_DIR/temp_logs"
    mkdir -p "$temp_log_dir"

    # Copy log files (only recent ones to avoid huge backups)
    for log_dir in "${log_dirs[@]}"; do
        if [[ -d "$log_dir" ]]; then
            # Copy logs from last 7 days
            find "$log_dir" -name "*.log" -mtime -7 -exec cp {} "$temp_log_dir/" \; 2>/dev/null || true
        fi
    done

    # Create tar archive
    if [[ -d "$temp_log_dir" ]] && [[ "$(ls -A "$temp_log_dir")" ]]; then
        tar -czf "$log_backup" -C "$BACKUP_DIR" temp_logs
        rm -rf "$temp_log_dir"
        
        local file_size=$(du -h "$log_backup" | cut -f1)
        print_status "Log backup completed: $log_backup ($file_size)"
        echo "$log_backup" >> "$BACKUP_DIR/backup_manifest.txt"
    else
        print_warning "No recent log files found to backup"
        rm -rf "$temp_log_dir"
    fi
}

# Function to backup certificates
backup_certificates() {
    print_info "Backing up certificates..."

    local cert_backup="$BACKUP_DIR/certificates_${TIMESTAMP}.tar.gz"
    local cert_dirs=(
        "/etc/ssl/certs/iaros"
        "/opt/iaros/certs"
        "$PROJECT_ROOT/infrastructure/security/certificates"
    )

    # Create temporary directory for certificates
    local temp_cert_dir="$BACKUP_DIR/temp_certs"
    mkdir -p "$temp_cert_dir"

    # Copy certificate files
    for cert_dir in "${cert_dirs[@]}"; do
        if [[ -d "$cert_dir" ]]; then
            cp -r "$cert_dir" "$temp_cert_dir/" 2>/dev/null || true
        fi
    done

    # Create tar archive
    if [[ -d "$temp_cert_dir" ]] && [[ "$(ls -A "$temp_cert_dir")" ]]; then
        tar -czf "$cert_backup" -C "$BACKUP_DIR" temp_certs
        rm -rf "$temp_cert_dir"
        
        local file_size=$(du -h "$cert_backup" | cut -f1)
        print_status "Certificate backup completed: $cert_backup ($file_size)"
        echo "$cert_backup" >> "$BACKUP_DIR/backup_manifest.txt"
    else
        print_warning "No certificates found to backup"
        rm -rf "$temp_cert_dir"
    fi
}

# Function to create backup metadata
create_backup_metadata() {
    print_info "Creating backup metadata..."

    local metadata_file="$BACKUP_DIR/backup_metadata.json"
    
    cat > "$metadata_file" << EOF
{
    "backup_timestamp": "$TIMESTAMP",
    "backup_date": "$(date -Iseconds)",
    "backup_version": "3.0.0",
    "iaros_version": "$(git describe --tags --always 2>/dev/null || echo "unknown")",
    "backup_type": "full",
    "environment": "${ENVIRONMENT:-production}",
    "hostname": "$(hostname)",
    "user": "$(whoami)",
    "backup_options": {
        "databases": $BACKUP_DATABASES,
        "configurations": $BACKUP_CONFIGS,
        "logs": $BACKUP_LOGS,
        "certificates": $BACKUP_CERTIFICATES,
        "encrypted": $ENCRYPT_BACKUP,
        "compressed": $COMPRESS_BACKUP
    },
    "database_info": {
        "postgresql": {
            "host": "$DB_HOST",
            "port": "$DB_PORT",
            "database": "$DB_NAME",
            "user": "$DB_USER"
        },
        "redis": {
            "host": "$REDIS_HOST",
            "port": "$REDIS_PORT"
        },
        "mongodb": {
            "host": "$MONGO_HOST",
            "port": "$MONGO_PORT",
            "database": "$MONGO_DB"
        }
    }
}
EOF

    print_status "Backup metadata created: $metadata_file"
}

# Function to encrypt backup
encrypt_backup() {
    if [[ "$ENCRYPT_BACKUP" == false ]]; then
        return
    fi

    print_info "Encrypting backup archive..."

    local archive_file="$BACKUP_DIR.tar.gz"
    local encrypted_file="$BACKUP_DIR.tar.gz.gpg"

    # Check for GPG key
    local gpg_recipient="${GPG_RECIPIENT:-iaros-backup@company.com}"
    
    if ! gpg --list-keys "$gpg_recipient" &> /dev/null; then
        print_warning "GPG key for $gpg_recipient not found, skipping encryption"
        ENCRYPT_BACKUP=false
        return
    fi

    # Encrypt the archive
    gpg --trust-model always --encrypt --recipient "$gpg_recipient" \
        --output "$encrypted_file" "$archive_file"

    if [[ -f "$encrypted_file" ]]; then
        rm "$archive_file"
        mv "$encrypted_file" "$archive_file"
        print_status "Backup encrypted successfully"
    else
        print_error "Backup encryption failed"
        exit 1
    fi
}

# Function to verify backup integrity
verify_backup() {
    if [[ "$VERIFY_BACKUP" == false ]]; then
        return
    fi

    print_info "Verifying backup integrity..."

    local archive_file="$BACKUP_DIR.tar.gz"

    # Verify tar archive
    if tar -tzf "$archive_file" > /dev/null 2>&1; then
        print_status "Backup archive integrity verified"
    else
        print_error "Backup archive is corrupted"
        exit 1
    fi

    # Verify individual database backups
    if [[ "$BACKUP_DATABASES" == true ]]; then
        # Verify PostgreSQL backup
        local pg_backup=$(find "$BACKUP_DIR" -name "postgresql_*.sql" 2>/dev/null | head -1)
        if [[ -f "$pg_backup" ]]; then
            if pg_restore --list "$pg_backup" > /dev/null 2>&1; then
                print_status "PostgreSQL backup verified"
            else
                print_error "PostgreSQL backup verification failed"
                exit 1
            fi
        fi
    fi
}

# Function to upload backup to S3
upload_to_s3() {
    if [[ "$UPLOAD_TO_S3" != true ]]; then
        return
    fi

    print_info "Uploading backup to S3..."

    local archive_file="$BACKUP_DIR.tar.gz"
    local s3_key="backups/$(date +%Y/%m/%d)/iaros_backup_${TIMESTAMP}.tar.gz"

    if aws s3 cp "$archive_file" "s3://$S3_BUCKET/$s3_key" --region "$S3_REGION"; then
        print_status "Backup uploaded to S3: s3://$S3_BUCKET/$s3_key"
        
        # Create S3 metadata
        echo "s3://$S3_BUCKET/$s3_key" > "$BACKUP_DIR/s3_location.txt"
    else
        print_error "S3 upload failed"
        exit 1
    fi
}

# Function to cleanup old backups
cleanup_old_backups() {
    print_info "Cleaning up old backups (retention: $RETENTION_DAYS days)..."

    # Local cleanup
    find "$BACKUP_ROOT" -name "backup_*" -type d -mtime +$RETENTION_DAYS -exec rm -rf {} + 2>/dev/null || true
    find "$BACKUP_ROOT" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true

    # S3 cleanup (if S3 upload is enabled)
    if [[ "$UPLOAD_TO_S3" == true ]]; then
        local cutoff_date=$(date -d "$RETENTION_DAYS days ago" +%Y-%m-%d)
        aws s3 ls "s3://$S3_BUCKET/backups/" --recursive | \
        awk '$1 < "'$cutoff_date'" {print $4}' | \
        while read file; do
            aws s3 rm "s3://$S3_BUCKET/$file" --region "$S3_REGION"
        done
    fi

    print_status "Old backups cleaned up"
}

# Function to restore from backup
restore_from_backup() {
    if [[ "$RESTORE_MODE" != true ]]; then
        return
    fi

    print_info "Restoring from backup: $RESTORE_FILE"

    local backup_path="$BACKUP_ROOT/$RESTORE_FILE"
    
    if [[ ! -f "$backup_path" ]]; then
        print_error "Backup file not found: $backup_path"
        exit 1
    fi

    # Extract backup
    local temp_restore_dir="/tmp/iaros_restore_$$"
    mkdir -p "$temp_restore_dir"
    
    tar -xzf "$backup_path" -C "$temp_restore_dir"

    # Restore databases
    print_info "Restoring databases..."
    
    # PostgreSQL restore
    local pg_backup=$(find "$temp_restore_dir" -name "postgresql_*.sql" | head -1)
    if [[ -f "$pg_backup" ]]; then
        PGPASSWORD="$DB_PASSWORD" pg_restore \
            --host="$DB_HOST" \
            --port="$DB_PORT" \
            --username="$DB_USER" \
            --dbname="$DB_NAME" \
            --clean \
            --if-exists \
            "$pg_backup"
        print_status "PostgreSQL database restored"
    fi

    # Redis restore
    local redis_backup=$(find "$temp_restore_dir" -name "redis_*.rdb" | head -1)
    if [[ -f "$redis_backup" ]]; then
        # Stop Redis service, replace dump file, restart
        print_warning "Redis restore requires manual intervention"
        print_info "Copy $redis_backup to Redis data directory and restart Redis"
    fi

    # MongoDB restore
    local mongo_backup=$(find "$temp_restore_dir" -name "mongodb_*.tar.gz" | head -1)
    if [[ -f "$mongo_backup" ]]; then
        local mongo_extract_dir="$temp_restore_dir/mongo_restore"
        mkdir -p "$mongo_extract_dir"
        tar -xzf "$mongo_backup" -C "$mongo_extract_dir"
        
        mongorestore \
            --host "$MONGO_HOST:$MONGO_PORT" \
            --username "$MONGO_USER" \
            --password "$MONGO_PASSWORD" \
            --authenticationDatabase admin \
            --db "$MONGO_DB" \
            --drop \
            "$mongo_extract_dir"
        print_status "MongoDB database restored"
    fi

    # Cleanup
    rm -rf "$temp_restore_dir"
    print_status "Restore completed"
}

# Main backup function
perform_backup() {
    # Create backup directory
    BACKUP_DIR="$BACKUP_ROOT/backup_${TIMESTAMP}"
    mkdir -p "$BACKUP_DIR"

    print_info "Starting IAROS backup..."
    print_info "Backup directory: $BACKUP_DIR"

    # Initialize backup manifest
    echo "# IAROS Backup Manifest - $TIMESTAMP" > "$BACKUP_DIR/backup_manifest.txt"
    echo "# Generated on: $(date)" >> "$BACKUP_DIR/backup_manifest.txt"
    echo "" >> "$BACKUP_DIR/backup_manifest.txt"

    # Perform backups based on options
    if [[ "$BACKUP_DATABASES" == true ]]; then
        backup_postgresql
        backup_redis
        backup_mongodb
    fi

    if [[ "$BACKUP_CONFIGS" == true ]]; then
        backup_configurations
    fi

    if [[ "$BACKUP_LOGS" == true ]]; then
        backup_logs
    fi

    if [[ "$BACKUP_CERTIFICATES" == true ]]; then
        backup_certificates
    fi

    # Create metadata
    create_backup_metadata

    # Create final archive
    print_info "Creating backup archive..."
    local archive_file="$BACKUP_DIR.tar.gz"
    
    if [[ "$COMPRESS_BACKUP" == true ]]; then
        tar -czf "$archive_file" -C "$BACKUP_ROOT" "backup_$TIMESTAMP"
    else
        tar -cf "$archive_file" -C "$BACKUP_ROOT" "backup_$TIMESTAMP"
    fi

    # Remove temporary backup directory
    rm -rf "$BACKUP_DIR"

    local file_size=$(du -h "$archive_file" | cut -f1)
    print_status "Backup archive created: $archive_file ($file_size)"

    # Post-processing
    encrypt_backup
    verify_backup
    upload_to_s3
    cleanup_old_backups

    print_status "Backup completed successfully!"
    print_info "Backup location: $archive_file"
    
    if [[ "$UPLOAD_TO_S3" == true ]]; then
        print_info "S3 location: s3://$S3_BUCKET/backups/$(date +%Y/%m/%d)/iaros_backup_${TIMESTAMP}.tar.gz"
    fi
}

# Main function
main() {
    echo "🔄 IAROS Backup Script"
    echo "======================"
    
    parse_args "$@"
    check_prerequisites
    test_connections

    if [[ "$RESTORE_MODE" == true ]]; then
        restore_from_backup
    else
        perform_backup
    fi
}

# Error handling
trap 'print_error "Backup failed at line $LINENO"' ERR

# Execute main function
main "$@" 