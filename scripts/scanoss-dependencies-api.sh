#!/bin/bash

##########################################
#
# This script is designed to run by Systemd SCANOSS Dependencies API service.
# It rotates scanoss log file and starts Dependencies API.
# Install it in /usr/local/bin
#
################################################################
DEFAULT_ENV="prod"
ENVIRONMENT="${1:-$DEFAULT_ENV}"
LOGFILE=/var/log/scanoss/dependencies/scanoss-dependencies-${ENVIRONMENT}.log
CONF_FILE=/usr/local/etc/scanoss/dependencies/app-config-${ENVIRONMENT}.json
# Rotate log
if [ -f "$LOGFILE" ] ; then
  echo "rotating logfile..."
  TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
  BACKUP_FILE=$LOGFILE.$TIMESTAMP
  cp "$LOGFILE" "$BACKUP_FILE"
  gzip -f "$BACKUP_FILE"
fi
echo > "$LOGFILE"

#start API
echo "Starting SCANOSS Dependencies API"

exec /usr/local/bin/scanoss-dependencies-api --json-config "$CONF_FILE" > "$LOGFILE" 2>&1
