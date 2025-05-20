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

if [ "$1" = "-h" ] || [ "$1" = "-help" ] ; then
  echo "$0 [-help] [environment]"
  echo "   Setup and copy the relevant files into place on a server to run the SCANOSS DEPENDENCIES API"
  echo "   [environment] allows the optional specification of a suffix to allow multiple services to be deployed at the same time (optional)"
  exit 1
fi
DEFAULT_ENV=""
ENVIRONMENT="${1:-$DEFAULT_ENV}"
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
read -p "Install SCANOSS Dependencies API $ENVIRONMENT (y/n) [n]? " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]] ; then
  echo "Starting installation..."
else
  echo "Stopping."
  exit 1
fi
# Setup all the required folders and ownership
echo "Setting up Dependencies API system folders..."
if ! mkdir -p $CONFIG_DIR ; then
  echo "Error: Problem creating dependency API system folders: $CONFIG_DIR."
  exti 1
fi
if ! mkdir -p $L_PATH ; then
  echo "Error: Problem creating dependency logging folder: $L_PATH."
  exit 1
fi
if [ "$RUNTIME_USER" != "root" ] ; then
  echo "Changing ownership of $LOG_DIR to $RUNTIME_USER ..."
  if ! chown -R $RUNTIME_USER $LOG_DIR ; then
    echo "Error: chown of $LOG_DIR to $RUNTIME_USER failed."
    exit 1
  fi
fi
# Setup the service on the system (defaulting to service name without environment)
SC_SERVICE_FILE="scanoss-dependencies-api.service"
SC_SERVICE_NAME="scanoss-dependencies-api"
if [ -n "$ENVIRONMENT" ] ; then
  SC_SERVICE_FILE="scanoss-dependencies-api-${ENVIRONMENT}.service"
  SC_SERVICE_NAME="scanoss-dependencies-api-${ENVIRONMENT}"
fi
export service_stopped=""
if [ -f "/etc/systemd/system/$SC_SERVICE_FILE" ] ; then
  echo "Stopping $SC_SERVICE_NAME service first..."
  if ! systemctl stop "$SC_SERVICE_NAME" ; then
    echo "Error: service stop failed"
    exit 1
  fi
  export service_stopped="true"
fi
echo "Copying service startup config..."
if [ -f "$SC_SERVICE_FILE" ] ; then
  if ! cp "$SC_SERVICE_FILE" /etc/systemd/system ; then
    echo "Error: service copy failed"
    exti 1
  fi
fi
if ! cp scanoss-dependencies-api.sh /usr/local/bin ; then
  echo "Error: dependencies startup script copy failed."
  exit 1
fi
####################################################
#                SEARCH CONFIG FILE                #
####################################################
CONF=app-config-prod.json
if [ -n "$ENVIRONMENT" ] ; then
  CONF="app-config-${ENVIRONMENT}.json"
fi
CONFIG_FILE_PATH=""
# Search on current dir
if [ -f "./$CONF" ]; then
    CONFIG_FILE_PATH="./$CONF"
# Search on parent dir
elif [ -f "../$CONF" ]; then
    CONFIG_FILE_PATH="../$CONF"
fi
############### END SEARCH CONFIG FILE ##############

####################################################
#                   SETUP SQLITE DB                #
####################################################
SQLITE_DB_PATH=""
# Search on current dir
if [ -f "./$SQLITE_DB_NAME" ]; then
    SQLITE_DB_PATH="./$SQLITE_DB_NAME"
# Search on parent dir
elif [ -f "../$SQLITE_DB_NAME" ]; then
    SQLITE_DB_PATH="../$SQLITE_DB_NAME"
fi
# Create SQLite DB dir
if [ ! -d "$SQLITE_PATH" ] ; then
  if ! mkdir -p "$SQLITE_PATH"; then
    echo "Error: Failed to create directory: $SQLITE_PATH"
    exit 1
  fi
fi
## If SQLite DB is found.
SQLITE_TARGET_PATH="$SQLITE_PATH/$TARGET_SQLITE_DB_NAME"
if [ -n "$SQLITE_DB_PATH" ]; then
    # If the target DB already exists, ask to replace it.
    if [ -f "$SQLITE_TARGET_PATH" ]; then
        read -p "SQLite file found at $(realpath "$SQLITE_DB_PATH"). Do you want to replace the ${SQLITE_TARGET_PATH}? (n/y) [n]: " -n 1 -r
              echo
       if [[ "$REPLY" =~ ^[Yy]$ ]] ; then
          echo "Copying SQLite from $(realpath "$SQLITE_DB_PATH") to $SQLITE_PATH"
          echo "Please be patient, this process might take some minutes..."
          if ! cp "$SQLITE_DB_PATH" "$SQLITE_TARGET_PATH"; then
              echo "Error: Failed to copy SQLite database."
              exit 1
          fi
          echo "Database successfully copied."
       else
         echo "Skipping DB copy."
       fi
    else
       # Copy database
       echo "Copying SQLite from $(realpath "$SQLITE_DB_PATH") to $SQLITE_PATH"
       echo "Please be patient, this process might take some minutes."
       if ! cp "$SQLITE_DB_PATH" "$SQLITE_TARGET_PATH"; then
           echo "Error: Failed to copy SQLite database from $SQLITE_DB_PATH to $SQLITE_PATH"
           exit 1
       fi
       echo "Database successfully copied."
    fi
else
  echo "Warning: No SQLite DB detected. Skipping DB setup."
fi
if [ ! -f "$SQLITE_TARGET_PATH" ] ; then
  echo "Warning: No database exists at: $SQLITE_TARGET_PATH"
  echo "Service startup will most likely fail."
fi
############### END SETUP SQLITE DB ################


####################################################
#                  COPY CONFIG FILE                #
####################################################
TARGET_CONFIG_PATH="$CONFIG_DIR/$CONF"
if [ -n "$CONFIG_FILE_PATH" ]; then
  if [ -f "$TARGET_CONFIG_PATH" ]; then
      read -p "Configuration file found at $(realpath "$TARGET_CONFIG_PATH"). Do you want to replace $TARGET_CONFIG_PATH? (n/y) [n]: " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]] ; then
          echo "Copying config file from $(realpath "$CONFIG_FILE_PATH") to $CONFIG_DIR ..."
          if ! cp "$CONFIG_FILE_PATH" "$CONFIG_DIR/"; then
            echo "Error: Failed to copy config file."
            exit 1
          fi
        else
          echo "Skipping config file copy."
        fi
  else
      echo "Copying config file from $(realpath "$CONFIG_FILE_PATH") to $CONFIG_DIR ..."
      if ! cp "$CONFIG_FILE_PATH" "$CONFIG_DIR/"; then
        echo "Error: Failed to copy config file."
        exit 1
      fi
  fi
else
   read -p "Configuration file not found. Do you want to download an example $CONF file? (n/y) [y]: " -n 1 -r
        echo
      if [[ $REPLY =~ ^[Yy]$ ]] ; then
          if curl $CONF_DOWNLOAD_URL > "$CONFIG_DIR/$CONF" ; then
            echo "Configuration file successfully downloaded to $CONFIG_DIR/$CONF"
          else
           echo "Error: Failed to download configuration file from $CONF_DOWNLOAD_URL"
          fi
      else
         echo "Warning: Please put the config file into: $CONFIG_DIR/$CONF"
      fi
fi

if [ ! -f "$TARGET_CONFIG_PATH" ] ; then
  echo "Warning: No application config file in place: $TARGET_CONFIG_PATH"
  echo "Service startup will most likely fail, especially in relation to the DB location."
fi
################ END CONFIG FILE ##################

####################################################
#         CHANGE OWNERSHIP AND PERMISSIONS         #
####################################################
# Change ownership to config folder
if ! chown -R $RUNTIME_USER:$RUNTIME_USER "$BASE_C_PATH"; then
  echo "Error: Problem changing ownership to config folder: $BASE_C_PATH"
  exit 1
fi
# Change permissions to config folder
if ! find "$CONFIG_DIR" -type d -exec chmod 0750 "{}" \; ; then
  echo "Error: Problem changing permissions to config folder: $CONFIG_DIR"
  exit 1
fi
# Change permissions to config folder files
if ! find "$CONFIG_DIR" -type f -exec chmod 0600 "{}" \; ; then
  echo "Error: Problem changing permissions to config files within: $CONFIG_DIR"
  exit 1
fi
# Change ownership to SQLite folder
if ! chown -R $RUNTIME_USER:$RUNTIME_USER "$DB_PATH_BASE"; then
    echo "Error: Failed to change ownership to $RUNTIME_USER"
    echo "Please check if the user exists and you have proper permissions."
    exit 1
fi
# Change permissions to config folder
if ! find "$DB_PATH_BASE" -type d -exec chmod 0750 "{}" \; ; then
  echo "Error: Problem changing permissions to DB folder: $CONFIG_DIR"
  exit 1
fi
# Change permissions to config folder files
if ! find "$DB_PATH_BASE" -type f -exec chmod 0640 "{}" \; ; then
  echo "Error: Problem changing permissions to DB files within: $CONFIG_DIR"
  exit 1
fi
######  END CHANGE OWNERSHIP AND PERMISSIONS #######

# Copy the binaries if requested
BINARY=scanoss-dependencies-api
if [ -f $BINARY ] ; then
  echo "Copying app binary to /usr/local/bin ..."
  if ! cp $BINARY /usr/local/bin ; then
    echo "Error: copy $BINARY failed."
    echo "Please make sure the service is stopped: systemctl stop scanoss-dependencies-api."
    exit 1
  fi
else
  echo "Please copy the API binary file into: /usr/local/bin/$BINARY"
fi

echo "Installation complete."
if [ "$service_stopped" == "true" ] ; then
  echo "Restarting service after install..."
  if ! systemctl start "$SC_SERVICE_NAME" ; then
    echo "Error: failed to restart service"
    exit 1
  fi
  systemctl status "$SC_SERVICE_NAME"
fi
echo
echo "Review service config in: $TARGET_CONFIG_PATH"
echo "Review service logs in: $L_PATH"
echo "Start the service using: systemctl start $SC_SERVICE_NAME"
echo "Stop the service using: systemctl stop $SC_SERVICE_NAME"
echo "Get service status using: systemctl status $SC_SERVICE_NAME"
echo
