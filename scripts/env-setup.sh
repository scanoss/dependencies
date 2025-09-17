#!/bin/bash

##########################################
#
# This script will copy all the required files into the correct locations on the server
# Config goes into: /usr/local/etc/scanoss/dependencies
# Logs go into: /var/log/scanoss/dependencies
# Service definition goes into: /etc/systemd/system
# Binary & startup go into: /usr/local/bin
#
################################################################

show_help() {
  echo "$0 [-h|--help] [-f|--force] [environment]"
  echo "   Setup and copy the required files into place on a server to run the SCANOSS DEPENDENCIES API"
  echo "   [environment] allows the optional specification of a suffix to allow multiple services"
  echo "   -f | --force  Run without interactive prompts (skip questions, skip SQLite setup, do not overwrite config)"
  exit 1
}

# --- Parse flags ---
FORCE=false
DEFAULT_ENV=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      show_help
      ;;
    -f|--force)
      FORCE=true
      shift
      ;;
    *)
      ENVIRONMENT="$1"
      shift
      ;;
  esac
done

ENVIRONMENT="${ENVIRONMENT:-$DEFAULT_ENV}"
export BASE_C_PATH=/usr/local/etc/scanoss
export CONFIG_DIR="${BASE_C_PATH}/dependencies"
export LOG_DIR=/var/log/scanoss
export L_PATH="${LOG_DIR}/dependencies"
export DB_PATH_BASE=/var/lib/scanoss
export SQLITE_PATH="${DB_PATH_BASE}/db/sqlite/dependencies"
export SQLITE_DB_NAME=base.sqlite
export TARGET_SQLITE_DB_NAME=db.sqlite
export CONF_DOWNLOAD_URL="https://raw.githubusercontent.com/scanoss/dependencies/refs/heads/main/config/app-config-prod.json"

# Makes sure the scanoss user exists
export RUNTIME_USER=scanoss
if ! getent passwd $RUNTIME_USER > /dev/null ; then
  echo "Runtime user does not exist: $RUNTIME_USER."
  echo "Please create using: useradd --system $RUNTIME_USER."
  exit 1
fi
# Also, make sure we're running as root
if [ "$EUID" -ne 0 ] ; then
  echo "Please run as root."
  exit 1
fi

if [ "$FORCE" = false ]; then
  read -p "Install SCANOSS Dependencies API $ENVIRONMENT (y/n) [n]? " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]] ; then
    echo "Stopping."
    exit 1
  fi
else
  echo "[FORCE] Auto-accept installation of SCANOSS Dependencies API $ENVIRONMENT"
fi

# Setup all the required folders and ownership
echo "Setting up Dependencies API system folders..."
mkdir -p "$CONFIG_DIR" || { echo "Error: Problem creating $CONFIG_DIR"; exit 1; }
mkdir -p "$L_PATH" || { echo "Error: Problem creating $L_PATH"; exit 1; }

if [ "$RUNTIME_USER" != "root" ] ; then
  echo "Changing ownership of $LOG_DIR to $RUNTIME_USER ..."
  chown -R $RUNTIME_USER $LOG_DIR || { echo "Error: chown failed"; exit 1; }
fi

# Setup the service on the system
SC_SERVICE_FILE="scanoss-dependencies-api.service"
SC_SERVICE_NAME="scanoss-dependencies-api"
if [ -n "$ENVIRONMENT" ] ; then
  SC_SERVICE_FILE="scanoss-dependencies-api-${ENVIRONMENT}.service"
  SC_SERVICE_NAME="scanoss-dependencies-api-${ENVIRONMENT}"
fi

service_stopped=""
if [ -f "/etc/systemd/system/$SC_SERVICE_FILE" ] ; then
  echo "Stopping $SC_SERVICE_NAME service first..."
  systemctl stop "$SC_SERVICE_NAME" || { echo "Error: service stop failed"; exit 1; }
  service_stopped="true"
fi

echo "Copying service startup config..."
if [ -f "$SC_SERVICE_FILE" ] ; then
  cp "$SC_SERVICE_FILE" /etc/systemd/system || { echo "Error: service copy failed"; exit 1; }
fi
cp scanoss-dependencies-api.sh /usr/local/bin || { echo "Error: startup script copy failed"; exit 1; }

####################################################
#                SEARCH CONFIG FILE                #
####################################################
CONF=app-config-prod.json
if [ -n "$ENVIRONMENT" ] ; then
  CONF="app-config-${ENVIRONMENT}.json"
fi
CONFIG_FILE_PATH=""
if [ -f "./$CONF" ]; then
    CONFIG_FILE_PATH="./$CONF"
elif [ -f "../$CONF" ]; then
    CONFIG_FILE_PATH="../$CONF"
fi

####################################################
#                   SETUP SQLITE DB                #
####################################################
if [ "$FORCE" = true ]; then
  echo "[FORCE] Skipping all SQLite DB setup."
else
  SQLITE_DB_PATH=""
  if [ -f "./$SQLITE_DB_NAME" ]; then
      SQLITE_DB_PATH="./$SQLITE_DB_NAME"
  elif [ -f "../$SQLITE_DB_NAME" ]; then
      SQLITE_DB_PATH="../$SQLITE_DB_NAME"
  fi

  mkdir -p "$SQLITE_PATH" || { echo "Error: Failed to create directory $SQLITE_PATH"; exit 1; }
  SQLITE_TARGET_PATH="$SQLITE_PATH/$TARGET_SQLITE_DB_NAME"

  if [ -n "$SQLITE_DB_PATH" ]; then
      if [ -f "$SQLITE_TARGET_PATH" ]; then
          read -p "SQLite file found. Replace $SQLITE_TARGET_PATH? (n/y) [n]: " -n 1 -r
          echo
          if [[ "$REPLY" =~ ^[Yy]$ ]] ; then
            cp "$SQLITE_DB_PATH" "$SQLITE_TARGET_PATH" || { echo "Error copying DB"; exit 1; }
          else
            echo "Skipping DB copy."
          fi
      else
          echo "Copying SQLite DB..."
          cp "$SQLITE_DB_PATH" "$SQLITE_TARGET_PATH" || { echo "Error copying DB"; exit 1; }
      fi
  else
    echo "Warning: No SQLite DB detected. Skipping DB setup."
  fi
fi

####################################################
#                  COPY CONFIG FILE                #
####################################################
####################################################
#                  COPY CONFIG FILE                #
####################################################
if [ "$FORCE" = true ]; then
  echo "[FORCE] Skipping config file setup."
else
  TARGET_CONFIG_PATH="$CONFIG_DIR/$CONF"
  if [ -n "$CONFIG_FILE_PATH" ]; then
    if [ -f "$TARGET_CONFIG_PATH" ]; then
        read -p "Config file exists. Replace $TARGET_CONFIG_PATH? (n/y) [n]: " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]] ; then
          cp "$CONFIG_FILE_PATH" "$CONFIG_DIR/" || { echo "Error copying config"; exit 1; }
        else
          echo "Skipping config copy."
        fi
    else
        cp "$CONFIG_FILE_PATH" "$CONFIG_DIR/" || { echo "Error copying config"; exit 1; }
    fi
  else
     read -p "Config not found. Download example $CONF file? (n/y) [n]: " -n 1 -r
     echo
     if [[ $REPLY =~ ^[Yy]$ ]] ; then
       curl -s "$CONF_DOWNLOAD_URL" > "$CONFIG_DIR/$CONF" || echo "Error downloading config"
     else
       echo "Warning: Please put the config file into: $CONFIG_DIR/$CONF"
     fi
  fi
fi


####################################################
#         CHANGE OWNERSHIP AND PERMISSIONS         #
####################################################
chown -R $RUNTIME_USER:$RUNTIME_USER "$BASE_C_PATH" || { echo "Error chown $BASE_C_PATH"; exit 1; }
find "$CONFIG_DIR" -type d -exec chmod 0750 "{}" \;
find "$CONFIG_DIR" -type f -exec chmod 0600 "{}" \;
chown -R $RUNTIME_USER:$RUNTIME_USER "$DB_PATH_BASE"
find "$DB_PATH_BASE" -type d -exec chmod 0750 "{}" \;
find "$DB_PATH_BASE" -type f -exec chmod 0640 "{}" \;

# Copy the binaries if requested
BINARY=scanoss-dependencies-api
if [ -f $BINARY ] ; then
  echo "Copying app binary to /usr/local/bin ..."
  cp $BINARY /usr/local/bin || { echo "Error copying $BINARY"; exit 1; }
else
  echo "Please copy the API binary file into: /usr/local/bin/$BINARY"
fi

echo "Installation complete."
if [ "$service_stopped" == "true" ] ; then
  echo "Restarting service after install..."
  systemctl start "$SC_SERVICE_NAME" || { echo "Error restarting service"; exit 1; }
  systemctl status "$SC_SERVICE_NAME"
fi

echo
echo "Review service config in: $TARGET_CONFIG_PATH"
echo "Review service logs in: $L_PATH"
echo "Start the service using: systemctl start $SC_SERVICE_NAME"
echo "Stop the service using: systemctl stop $SC_SERVICE_NAME"
echo "Get service status using: systemctl status $SC_SERVICE_NAME"
