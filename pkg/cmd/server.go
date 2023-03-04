// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2022 SCANOSS.COM
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package cmd

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	myconfig "scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/protocol/grpc"
	"scanoss.com/dependencies/pkg/protocol/rest"
	"scanoss.com/dependencies/pkg/service"
	"strings"
	"syscall"
	"time"
)

//go:generate bash ../../get_version.sh
//go:embed version.txt
var version string

// getConfig checks command line args for option to feed into the config parser
func getConfig() (*myconfig.ServerConfig, error) {
	var jsonConfig, envConfig, loggingConfig string
	flag.StringVar(&jsonConfig, "json-config", "", "Application JSON config")
	flag.StringVar(&envConfig, "env-config", "", "Application dot-ENV config")
	flag.StringVar(&loggingConfig, "logging-config", "", "Logging config file")
	debug := flag.Bool("debug", false, "Enable debug")
	ver := flag.Bool("version", false, "Display current version")
	flag.Parse()
	if *ver {
		fmt.Printf("Version: %v", version)
		os.Exit(1)
	}
	var feeders []config.Feeder
	if len(jsonConfig) > 0 {
		feeders = append(feeders, feeder.Json{Path: jsonConfig})
	}
	if len(envConfig) > 0 {
		feeders = append(feeders, feeder.DotEnv{Path: envConfig})
	}
	if *debug {
		err := os.Setenv("APP_DEBUG", "1")
		if err != nil {
			fmt.Printf("Warning: Failed to set env APP_DEBUG to 1: %v", err)
			return nil, err
		}
	}
	myConfig, err := myconfig.NewServerConfig(feeders)
	if len(loggingConfig) > 0 {
		myConfig.Logging.ConfigFile = loggingConfig // Override any logging config file with this one.
	}
	return myConfig, err
}

// closeDbConnection closes the specified DB connection
func closeDbConnection(db *sqlx.DB) {
	err := db.Close()
	if err != nil {
		zlog.S.Warnf("Problem closing DB: %v", err)
	}
}

// RunServer runs the gRPC Dependency Server
func RunServer() error {
	// Load command line options and config
	cfg, err := getConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	// Check mode to determine which logger to load
	{
		var err error
		switch strings.ToLower(cfg.App.Mode) {
		case "prod":
			if len(cfg.Logging.ConfigFile) > 0 {
				err = zlog.NewSugaredLoggerFromFile(cfg.Logging.ConfigFile)
			} else {
				err = zlog.NewSugaredProdLogger()
			}
		default:
			if len(cfg.Logging.ConfigFile) > 0 {
				err = zlog.NewSugaredLoggerFromFile(cfg.Logging.ConfigFile)
			} else {
				err = zlog.NewSugaredDevLogger()
			}
		}
		if err != nil {
			return fmt.Errorf("failed to load logger: %v", err)
		}
		if cfg.App.Debug {
			zlog.SetLevel("debug")
		}
		zlog.L.Debug("Running with debug enabled")
		defer zlog.SyncZap()
	}
	startTLS, err := checkTLS(cfg)
	if err != nil {
		return err
	}
	allowedIPs, deniedIPs, err := loadFiltering(cfg)
	if err != nil {
		return err
	}
	zlog.S.Infof("Starting SCANOSS Dependency Service: %v", strings.TrimSpace(version))
	// Setup database connection pool
	var dsn string
	if len(cfg.Database.Dsn) > 0 {
		dsn = cfg.Database.Dsn
	} else {
		dsn = fmt.Sprintf("%s://%s:%s@%s/%s?sslmode=%s",
			cfg.Database.Driver,
			cfg.Database.User,
			cfg.Database.Passwd,
			cfg.Database.Host,
			cfg.Database.Schema,
			cfg.Database.SslMode)
	}
	zlog.S.Debug("Connecting to Database...")
	db, err := sqlx.Open(cfg.Database.Driver, dsn)
	if err != nil {
		zlog.S.Errorf("Failed to open database: %v", err)
		return fmt.Errorf("failed to open database: %v", err)
	}
	db.SetConnMaxIdleTime(30 * time.Minute) // TODO add to app config
	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxIdleConns(20)
	db.SetMaxOpenConns(200)
	err = db.Ping()
	if err != nil {
		zlog.S.Errorf("Failed to ping database: %v", err)
		return fmt.Errorf("failed to ping database: %v", err)
	}
	defer closeDbConnection(db)
	if cfg.Logging.DynamicLogging && len(cfg.Logging.DynamicPort) > 0 {
		zlog.S.Infof("Setting up dynamic logging level on %v.", cfg.Logging.DynamicPort)
		zlog.SetupDynamicLogging(cfg.Logging.DynamicPort)
		zlog.S.Infof("Use the following to get the current status: curl -X GET %v/log/level", cfg.Logging.DynamicPort)
		zlog.S.Infof("Use the following to set the current status: curl -X PUT %v/log/level -d level=debug", cfg.Logging.DynamicPort)
	}
	v2API := service.NewDependencyServer(db, cfg)
	ctx := context.Background()
	var srv *http.Server
	// Start the REST grpc-gateway if requested
	if len(cfg.App.RESTPort) > 0 {
		if srv, err = rest.RunServer(cfg, ctx, cfg.App.GRPCPort, cfg.App.RESTPort, allowedIPs, deniedIPs, startTLS); err != nil {
			return err
		}
	}
	// Start the gRPC service
	server, err := grpc.RunServer(cfg, v2API, cfg.App.GRPCPort, allowedIPs, deniedIPs, startTLS)
	if err != nil {
		return err
	}
	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	<-c
	if srv != nil {
		zlog.S.Info("shutting down REST server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Set a deadline for gracefully shutting down
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			zlog.S.Warnf("error shutting down server %s", err)
			return fmt.Errorf("issue encountered while shutting down service")
		} else {
			zlog.S.Info("REST server gracefully stopped")
		}
	}
	if server != nil {
		zlog.S.Info("shutting down gRPC server...")
		server.GracefulStop()
		zlog.S.Info("gRPC server gracefully stopped")
	}
	return nil
}

// TODO add these to shared library

// checkFile validates if the given file exists or not.
func checkFile(filename string) (bool, error) {
	if len(filename) == 0 {
		return false, fmt.Errorf("no file specified to check")
	}
	fileDetails, err := os.Stat(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("file doest no exist")
		}
		return false, err
	}
	if fileDetails.IsDir() {
		return false, fmt.Errorf("is a directory and not a file")
	}
	return true, nil
}

// checkTLS tests if TLS should be enabled or not.
func checkTLS(config *myconfig.ServerConfig) (bool, error) {
	var startTLS = false
	if len(config.TLS.CertFile) > 0 && len(config.TLS.KeyFile) > 0 {
		cf, err := checkFile(config.TLS.CertFile)
		if err != nil || !cf {
			zlog.S.Errorf("Cert file is not accessible: %v", config.TLS.CertFile)
			if err != nil {
				return false, err
			} else {
				return false, fmt.Errorf("cert file not accesible: %v", config.TLS.CertFile)
			}
		}
		kf, err := checkFile(config.TLS.KeyFile)
		if err != nil || !kf {
			zlog.S.Errorf("Key file is not accessible: %v", config.TLS.KeyFile)
			if err != nil {
				return false, err
			} else {
				return false, fmt.Errorf("key file not accesible: %v", config.TLS.KeyFile)
			}
		}
		startTLS = true
	}
	return startTLS, nil
}

// loadFiltering loads the IP filtering options if available.
func loadFiltering(config *myconfig.ServerConfig) ([]string, []string, error) {
	var allowedIPs []string
	if len(config.Filtering.AllowListFile) > 0 {
		cf, err := checkFile(config.Filtering.AllowListFile)
		if err != nil || !cf {
			zlog.S.Errorf("Allow List file is not accessible: %v", config.Filtering.AllowListFile)
			if err != nil {
				return nil, nil, err
			} else {
				return nil, nil, fmt.Errorf("allow list file not accesible: %v", config.Filtering.AllowListFile)
			}
		}
		allowedIPs, err = myconfig.LoadFile(config.Filtering.AllowListFile)
		if err != nil {
			return nil, nil, err
		}
	}
	var deniedIPs []string
	if len(config.Filtering.DenyListFile) > 0 {
		cf, err := checkFile(config.Filtering.DenyListFile)
		if err != nil || !cf {
			zlog.S.Errorf("Deny List file is not accessible: %v", config.Filtering.DenyListFile)
			if err != nil {
				return nil, nil, err
			} else {
				return nil, nil, fmt.Errorf("deny list file not accesible: %v", config.Filtering.DenyListFile)
			}
		}
		deniedIPs, err = myconfig.LoadFile(config.Filtering.DenyListFile)
		if err != nil {
			return nil, nil, err
		}
	}
	return allowedIPs, deniedIPs, nil
}
