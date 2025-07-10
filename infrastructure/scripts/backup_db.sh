#!/bin/bash
# backup_db.sh - Enterprise Database Backup System for IAROS
# Author: IAROS Infrastructure Team
# Version: 2.0.0

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONFIG_DIR="${PROJECT_ROOT}/infrastructure/config"
LOG_DIR="${PROJECT_ROOT}/logs"
BACKUP_DIR="${PROJECT_ROOT}/backups"
ARCHIVE_DIR="${PROJECT_ROOT}/backups/archive"

# Database Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-iaros_db}"
DB_USER="${DB_USER:-iaros_user}"
DB_PASSWORD="${DB_PASSWORD:-}"

# Backup Configuration
RETENTION_DAYS=${RETENTION_DAYS:-30}
COMPRESSION_LEVEL=${COMPRESSION_LEVEL:-6}
ENCRYPTION_ENABLED=${ENCRYPTION_ENABLED:-true}
ENCRYPTION_KEY_FILE="${CONFIG_DIR}/backup.key"
PARALLEL_JOBS=${PARALLEL_JOBS:-4}
BACKUP_TYPE="${1:-full}" # full, incremental, differential

# S3 Configuration (optional)
S3_ENABLED=${S3_ENABLED:-false}
S3_BUCKET=${S3_BUCKET:-""}
S3_REGION=${S3_REGION:-"us-east-1"}

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[0;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/backup.log"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/backup.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/backup.log"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "${LOG_DIR}/backup.log"
}

# Error handling
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        log_error "Backup failed with exit code $exit_code"
        # Clean up temporary files
        rm -f "${BACKUP_DIR}/.tmp_"*
        # Send failure notification
        send_notification "FAILED" "Database backup failed"
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

get_timestamp() {
    date '+%Y%m%d_%H%M%S'
}

get_date() {
    date '+%Y-%m-%d'
}

# Validation functions
validate_environment() {
    log_info "Validating backup environment..."
    
    # Check required commands
    local required_commands=("pg_dump" "pg_isready" "gzip" "openssl")
    for cmd in "${required_commands[@]}"; do
        if ! check_command "$cmd"; then
            log_error "Required command '$cmd' not found"
            exit 1
        fi
    done
    
    # Check S3 tools if enabled
    if [[ "$S3_ENABLED" == "true" ]]; then
        if ! check_command "aws"; then
            log_error "AWS CLI not found but S3 backup is enabled"
            exit 1
        fi
    fi
    
    # Create directories
    mkdir -p "$BACKUP_DIR" "$ARCHIVE_DIR" "$LOG_DIR"
    
    # Check disk space
    local available_space=$(df "$BACKUP_DIR" | awk 'NR==2 {print $4}')
    local available_gb=$((available_space / 1024 / 1024))
    
    if [[ $available_gb -lt 10 ]]; then
        log_warn "Available disk space: ${available_gb}GB. Consider cleaning old backups."
    fi
    
    log_success "Environment validation completed"
}

validate_database_connection() {
    log_info "Validating database connection..."
    
    # Test database connectivity
    if ! PGPASSWORD="$DB_PASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" &> /dev/null; then
        log_error "Cannot connect to database $DB_NAME on $DB_HOST:$DB_PORT"
        exit 1
    fi
    
    # Check database size
    local db_size=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT pg_size_pretty(pg_database_size('$DB_NAME'));" | xargs)
    log_info "Database size: $db_size"
    
    log_success "Database connection validated"
}

# Encryption functions
generate_encryption_key() {
    if [[ ! -f "$ENCRYPTION_KEY_FILE" ]]; then
        log_info "Generating encryption key..."
        openssl rand -base64 32 > "$ENCRYPTION_KEY_FILE"
        chmod 600 "$ENCRYPTION_KEY_FILE"
        log_success "Encryption key generated"
    fi
}

encrypt_file() {
    local input_file="$1"
    local output_file="$2"
    
    if [[ "$ENCRYPTION_ENABLED" == "true" ]]; then
        log_info "Encrypting backup file..."
        openssl enc -aes-256-cbc -salt -in "$input_file" -out "$output_file" -pass file:"$ENCRYPTION_KEY_FILE"
        rm -f "$input_file"
    else
        mv "$input_file" "$output_file"
    fi
}

decrypt_file() {
    local input_file="$1"
    local output_file="$2"
    
    if [[ "$ENCRYPTION_ENABLED" == "true" ]]; then
        log_info "Decrypting backup file..."
        openssl enc -aes-256-cbc -d -in "$input_file" -out "$output_file" -pass file:"$ENCRYPTION_KEY_FILE"
    else
        cp "$input_file" "$output_file"
    fi
}

# Backup functions
perform_full_backup() {
    local timestamp=$(get_timestamp)
    local backup_file="${BACKUP_DIR}/.tmp_${DB_NAME}_full_${timestamp}.sql"
    local compressed_file="${BACKUP_DIR}/.tmp_${DB_NAME}_full_${timestamp}.sql.gz"
    local final_file="${BACKUP_DIR}/${DB_NAME}_full_${timestamp}.sql.gz.enc"
    
    log_info "Starting full database backup..."
    
    # Perform database dump with optimizations
    PGPASSWORD="$DB_PASSWORD" pg_dump \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --verbose \
        --no-owner \
        --no-acl \
        --compress=0 \
        --jobs="$PARALLEL_JOBS" \
        --format=custom \
        --file="$backup_file"
    
    # Convert custom format to SQL for compression
    local sql_file="${backup_file}.sql"
    PGPASSWORD="$DB_PASSWORD" pg_restore \
        --no-owner \
        --no-acl \
        --format=custom \
        --file="$sql_file" \
        "$backup_file"
    
    rm -f "$backup_file"
    
    # Compress the backup
    log_info "Compressing backup file..."
    gzip -"$COMPRESSION_LEVEL" "$sql_file"
    
    # Encrypt the backup
    if [[ "$ENCRYPTION_ENABLED" == "true" ]]; then
        generate_encryption_key
        encrypt_file "${sql_file}.gz" "$final_file"
    else
        mv "${sql_file}.gz" "$final_file"
    fi
    
    # Validate backup
    validate_backup "$final_file"
    
    log_success "Full backup completed: $(basename "$final_file")"
    echo "$final_file"
}

perform_incremental_backup() {
    local timestamp=$(get_timestamp)
    local backup_file="${BACKUP_DIR}/${DB_NAME}_incremental_${timestamp}.sql.gz.enc"
    
    log_info "Starting incremental backup..."
    
    # Find the last full backup
    local last_full_backup=$(find "$BACKUP_DIR" -name "${DB_NAME}_full_*.sql.gz.enc" -type f -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2-)
    
    if [[ -z "$last_full_backup" ]]; then
        log_warn "No full backup found. Performing full backup instead."
        perform_full_backup
        return
    fi
    
    # Get the timestamp of the last backup
    local last_backup_time=$(stat -c %Y "$last_full_backup")
    local last_backup_date=$(date -d "@$last_backup_time" '+%Y-%m-%d %H:%M:%S')
    
    log_info "Last full backup: $last_backup_date"
    
    # Perform WAL-based incremental backup (PostgreSQL specific)
    # This is a simplified version - in production, use tools like WAL-E or pgBackRest
    local temp_file="${BACKUP_DIR}/.tmp_incremental_${timestamp}.sql"
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        COPY (
            SELECT * FROM pg_logical_slot_get_changes('backup_slot', NULL, NULL)
            WHERE xid > (
                SELECT max(xid) FROM pg_logical_slot_get_changes('backup_slot', NULL, 1)
                WHERE lsn <= pg_current_wal_lsn()
            )
        ) TO '$temp_file';
    " || {
        log_warn "Logical replication slot not available. Creating differential backup..."
        perform_differential_backup
        return
    }
    
    # Compress and encrypt
    gzip -"$COMPRESSION_LEVEL" "$temp_file"
    if [[ "$ENCRYPTION_ENABLED" == "true" ]]; then
        encrypt_file "${temp_file}.gz" "$backup_file"
    else
        mv "${temp_file}.gz" "$backup_file"
    fi
    
    log_success "Incremental backup completed: $(basename "$backup_file")"
    echo "$backup_file"
}

perform_differential_backup() {
    local timestamp=$(get_timestamp)
    local backup_file="${BACKUP_DIR}/${DB_NAME}_differential_${timestamp}.sql.gz.enc"
    
    log_info "Starting differential backup..."
    
    # This is a simplified differential backup
    # In production, implement based on table modification timestamps
    local temp_file="${BACKUP_DIR}/.tmp_differential_${timestamp}.sql"
    
    # Get tables modified since last backup
    PGPASSWORD="$DB_PASSWORD" pg_dump \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --data-only \
        --where="updated_at > (CURRENT_DATE - INTERVAL '1 day')" \
        --file="$temp_file" \
        2>/dev/null || {
            log_warn "Differential backup failed. Performing full backup..."
            perform_full_backup
            return
        }
    
    # Compress and encrypt
    gzip -"$COMPRESSION_LEVEL" "$temp_file"
    if [[ "$ENCRYPTION_ENABLED" == "true" ]]; then
        encrypt_file "${temp_file}.gz" "$backup_file"
    else
        mv "${temp_file}.gz" "$backup_file"
    fi
    
    log_success "Differential backup completed: $(basename "$backup_file")"
    echo "$backup_file"
}

validate_backup() {
    local backup_file="$1"
    
    log_info "Validating backup file..."
    
    # Check file exists and has content
    if [[ ! -f "$backup_file" ]]; then
        log_error "Backup file not found: $backup_file"
        exit 1
    fi
    
    local file_size=$(stat -c%s "$backup_file")
    if [[ $file_size -lt 1024 ]]; then
        log_error "Backup file too small: ${file_size} bytes"
        exit 1
    fi
    
    # Test decompression and decryption
    local temp_test_file="/tmp/backup_test_$(get_timestamp).sql"
    
    if [[ "$ENCRYPTION_ENABLED" == "true" ]]; then
        if ! decrypt_file "$backup_file" "$temp_test_file.gz" 2>/dev/null; then
            log_error "Failed to decrypt backup file"
            exit 1
        fi
        gunzip "$temp_test_file.gz" 2>/dev/null || {
            log_error "Failed to decompress backup file"
            exit 1
        }
    else
        gunzip -c "$backup_file" > "$temp_test_file" 2>/dev/null || {
            log_error "Failed to decompress backup file"
            exit 1
        }
    fi
    
    # Validate SQL content
    if ! head -n 10 "$temp_test_file" | grep -q "PostgreSQL database dump" 2>/dev/null; then
        log_warn "Backup file may not be a valid PostgreSQL dump"
    fi
    
    rm -f "$temp_test_file"
    
    # Calculate and store checksum
    local checksum=$(sha256sum "$backup_file" | cut -d' ' -f1)
    echo "$checksum  $(basename "$backup_file")" >> "${BACKUP_DIR}/checksums.txt"
    
    log_success "Backup validation completed (Size: $(du -h "$backup_file" | cut -f1))"
}

# Cloud storage functions
upload_to_s3() {
    local backup_file="$1"
    
    if [[ "$S3_ENABLED" != "true" ]]; then
        return 0
    fi
    
    log_info "Uploading backup to S3..."
    
    local s3_key="database-backups/$(basename "$backup_file")"
    
    if aws s3 cp "$backup_file" "s3://$S3_BUCKET/$s3_key" --region "$S3_REGION" --storage-class STANDARD_IA; then
        log_success "Backup uploaded to S3: s3://$S3_BUCKET/$s3_key"
        
        # Create a manifest entry
        cat >> "${BACKUP_DIR}/s3_manifest.txt" << EOF
$(date '+%Y-%m-%d %H:%M:%S') - $(basename "$backup_file") - s3://$S3_BUCKET/$s3_key
EOF
    else
        log_error "Failed to upload backup to S3"
        return 1
    fi
}

# Cleanup functions
cleanup_old_backups() {
    log_info "Cleaning up old backups (retention: $RETENTION_DAYS days)..."
    
    # Move old backups to archive
    find "$BACKUP_DIR" -name "${DB_NAME}_*.sql.gz.enc" -type f -mtime +$RETENTION_DAYS -exec mv {} "$ARCHIVE_DIR/" \;
    
    # Remove very old archives
    find "$ARCHIVE_DIR" -name "${DB_NAME}_*.sql.gz.enc" -type f -mtime +$((RETENTION_DAYS * 3)) -delete
    
    # Clean up old logs
    find "$LOG_DIR" -name "backup.log.*" -type f -mtime +$RETENTION_DAYS -delete
    
    # Update checksums file
    local temp_checksums="/tmp/checksums_$(get_timestamp).txt"
    if [[ -f "${BACKUP_DIR}/checksums.txt" ]]; then
        while read -r line; do
            local filename=$(echo "$line" | cut -d' ' -f3-)
            if [[ -f "${BACKUP_DIR}/$filename" ]]; then
                echo "$line" >> "$temp_checksums"
            fi
        done < "${BACKUP_DIR}/checksums.txt"
        
        mv "$temp_checksums" "${BACKUP_DIR}/checksums.txt"
    fi
    
    log_success "Cleanup completed"
}

# Notification functions
send_notification() {
    local status="$1"
    local message="$2"
    
    # Create notification payload
    local notification_data=$(cat << EOF
{
    "timestamp": "$(date -Iseconds)",
    "status": "$status",
    "database": "$DB_NAME",
    "backup_type": "$BACKUP_TYPE",
    "message": "$message",
    "host": "$(hostname)"
}
EOF
)
    
    # Send to monitoring system (example with webhook)
    if [[ -n "${WEBHOOK_URL:-}" ]]; then
        curl -X POST "$WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "$notification_data" \
            &> /dev/null || true
    fi
    
    # Send email notification (if configured)
    if [[ -n "${NOTIFICATION_EMAIL:-}" ]] && check_command "mail"; then
        echo "$notification_data" | mail -s "IAROS Database Backup $status" "$NOTIFICATION_EMAIL" || true
    fi
}

# Backup execution
execute_backup() {
    local backup_file=""
    
    case "$BACKUP_TYPE" in
        "full")
            backup_file=$(perform_full_backup)
            ;;
        "incremental")
            backup_file=$(perform_incremental_backup)
            ;;
        "differential")
            backup_file=$(perform_differential_backup)
            ;;
        *)
            log_error "Invalid backup type: $BACKUP_TYPE"
            exit 1
            ;;
    esac
    
    # Upload to cloud storage
    upload_to_s3 "$backup_file"
    
    # Generate backup report
    generate_backup_report "$backup_file"
    
    # Send success notification
    send_notification "SUCCESS" "Database backup completed successfully"
    
    return 0
}

generate_backup_report() {
    local backup_file="$1"
    local report_file="${BACKUP_DIR}/backup_report_$(get_timestamp).txt"
    
    cat > "$report_file" << EOF
IAROS Database Backup Report
============================
Date: $(date)
Database: $DB_NAME
Host: $DB_HOST:$DB_PORT
Backup Type: $BACKUP_TYPE
Backup File: $(basename "$backup_file")
File Size: $(du -h "$backup_file" | cut -f1)
Checksum: $(sha256sum "$backup_file" | cut -d' ' -f1)
Encryption: $ENCRYPTION_ENABLED
Compression Level: $COMPRESSION_LEVEL
Retention Days: $RETENTION_DAYS
S3 Upload: $S3_ENABLED

Backup Process Duration: $(($(date +%s) - START_TIME)) seconds

Database Statistics:
$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\l+" | head -20)
EOF
    
    log_info "Backup report generated: $report_file"
}

# Main execution
main() {
    local START_TIME=$(date +%s)
    
    log_info "Starting IAROS Database Backup System"
    log_info "======================================"
    log_info "Backup Type: $BACKUP_TYPE"
    log_info "Database: $DB_NAME on $DB_HOST:$DB_PORT"
    
    # Load configuration from file if exists
    if [[ -f "$CONFIG_DIR/backup.conf" ]]; then
        source "$CONFIG_DIR/backup.conf"
    fi
    
    # Validation
    validate_environment
    validate_database_connection
    
    # Execute backup
    execute_backup
    
    # Cleanup
    cleanup_old_backups
    
    local end_time=$(date +%s)
    local duration=$((end_time - START_TIME))
    
    log_success "Database backup completed successfully in ${duration} seconds"
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
