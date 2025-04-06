#!/bin/bash
# backup_db.sh - Backs up all databases with logging.
set -euo pipefail
DB_HOST="prod-db.iaros.ai"
DB_NAME="iaros_db"
BACKUP_FILE="/backups/${DB_NAME}_$(date +'%Y%m%d%H%M').sql"
echo "Starting backup for database $DB_NAME on $DB_HOST..."
pg_dump -h "$DB_HOST" -U produser "$DB_NAME" > "$BACKUP_FILE"
echo "Backup complete: $BACKUP_FILE"
