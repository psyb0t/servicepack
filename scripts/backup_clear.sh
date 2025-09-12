#!/bin/bash

set -e

LOCAL_BACKUP=".backup"

if [ ! -d "$LOCAL_BACKUP" ]; then
    echo "No backup directory found."
    exit 0
fi

BACKUP_COUNT=$(find "$LOCAL_BACKUP" -name "*.tar.gz" -type f 2>/dev/null | wc -l)

if [ "$BACKUP_COUNT" -eq 0 ]; then
    echo "No backup files found."
    exit 0
fi

echo "Found $BACKUP_COUNT backup files:"
find "$LOCAL_BACKUP" -name "*.tar.gz" -type f -exec basename {} \; 2>/dev/null

echo ""
read -r -p "Delete ALL backup files? Type 'yes' to continue: " confirm

if [ "$confirm" != "yes" ]; then
    echo "Backup clear cancelled."
    exit 0
fi

echo "Deleting backup files..."
rm -f "$LOCAL_BACKUP"/*.tar.gz

echo "All backup files deleted."