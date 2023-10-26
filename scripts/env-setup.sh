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

export C_PATH=/usr/local/etc/scanoss/dependencies
export LOG_DIR=/var/log/scanoss
export L_PATH="${LOG_DIR}/dependencies"

# Makes sure the scanoss user exists
export RUNTIME_USER=scanoss
if ! getent passwd $RUNTIME_USER > /dev/null ; then
  echo "Runtime user does not exist: $RUNTIME_USER"
  echo "Please create using: useradd --system $RUNTIME_USER"
  exit 1
fi
# Also, make sure we're running as root
if [ "$EUID" -ne 0 ] ; then
  echo "Please run as root"
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
if ! mkdir -p $C_PATH ; then
  echo "mkdir failed"
  exti 1
fi
if ! mkdir -p $L_PATH ; then
  echo "mkdir failed"
  exit 1
fi
if [ "$RUNTIME_USER" != "root" ] ; then
  echo "Changing ownership of $LOG_DIR to $RUNTIME_USER ..."
  if ! chown -R $RUNTIME_USER $LOG_DIR ; then
    echo "chown of $LOG_DIR to $RUNTIME_USER failed"
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
    echo "service stop failed"
    exit 1
  fi
  export service_stopped="true"
fi
echo "Copying service startup config..."
if [ -f "$SC_SERVICE_FILE" ] ; then
  if ! cp "$SC_SERVICE_FILE" /etc/systemd/system ; then
    echo "service copy failed"
    exti 1
  fi
fi
if ! cp scanoss-dependencies-api.sh /usr/local/bin ; then
  echo "dependencies startup script copy failed"
  exit 1
fi
# Copy in the configuration file if requested
CONF=app-config-prod.json
if [ -n "$ENVIRONMENT" ] ; then
  CONF="app-config-${ENVIRONMENT}.json"
fi
if [ -f "$CONF" ] ; then
  echo "Copying app config to $C_PATH ..."
  if ! cp "$CONF" $C_PATH ; then
    echo "copy $CONF failed"
    exit 1
  fi
else
  echo "Please put the config file into: $C_PATH/$CONF"
fi
# Copy the binaries if requested
BINARY=scanoss-dependencies-api
if [ -f $BINARY ] ; then
  echo "Copying app binary to /usr/local/bin ..."
  if ! cp $BINARY /usr/local/bin ; then
    echo "copy $BINARY failed"
    echo "Please make sure the service is stopped: systemctl stop scanoss-dependencies-api"
    exit 1
  fi
else
  echo "Please copy the API binary file into: /usr/local/bin/$BINARY"
fi

echo "Installation complete."
if [ "$service_stopped" == "true" ] ; then
  echo "Restarting service after install..."
  if ! systemctl start "$SC_SERVICE_NAME" ; then
    echo "failed to restart service"
    exit 1
  fi
  systemctl status "$SC_SERVICE_NAME"
fi
echo
echo "Review service config in: $C_PATH/$CONF"
echo "Review service logs in: $L_PATH"
echo "Start the service using: systemctl start $SC_SERVICE_NAME"
echo "Stop the service using: systemctl stop $SC_SERVICE_NAME"
echo "Get service status using: systemctl status $SC_SERVICE_NAME"
echo
