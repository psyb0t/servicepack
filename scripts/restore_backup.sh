#!/bin/bash

set -e

BACKUP_FILE="$1"

# If no backup file specified, use the latest one
if [ -z "$BACKUP_FILE" ]; then
    if [ -d ".backup" ] && find .backup -name "*.tar.gz" -type f -print -quit 2>/dev/null | grep -q .; then
        BACKUP_FILE=$(find .backup -name "*.tar.gz" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -n1 | cut -d' ' -f2- | xargs basename)
        echo "No backup specified, using latest: $BACKUP_FILE"
    else
        echo "Error: No backups found in .backup/"
        echo "Usage: make restore-backup [BACKUP=filename.tar.gz]"
        exit 1
    fi
fi

# Check if backup exists in local backup directory
LOCAL_BACKUP=".backup/$BACKUP_FILE"
if [ ! -f "$LOCAL_BACKUP" ]; then
    echo "Error: Backup file not found: $LOCAL_BACKUP"
    echo ""
    echo "Available backups:"
    if [ -d ".backup" ] && find .backup -name "*.tar.gz" -type f -print -quit 2>/dev/null | grep -q .; then
        find .backup -name "*.tar.gz" -type f -exec basename {} \; 2>/dev/null
    else
        echo "  No backups found in .backup/"
    fi
    exit 1
fi

echo "WARNING: This will delete EVERYTHING in the current directory and restore from backup!"
echo "Backup file: $LOCAL_BACKUP"
echo ""
read -r -p "Are you sure? Type 'yes' to continue: " confirm

if [ "$confirm" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

echo "Nuking current directory (except .backup)..."
find . -mindepth 1 -maxdepth 1 ! -name '.backup' -exec rm -rf {} +

echo "Restoring from backup..."
tar -xzf "$LOCAL_BACKUP"

echo "Restore completed successfully!"
echo "Don't forget to run 'make dep' to update dependencies if needed."