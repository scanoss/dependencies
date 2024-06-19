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
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/scanoss/go-grpc-helper/pkg/files"
	gd "github.com/scanoss/go-grpc-helper/pkg/grpc/database"
	gs "github.com/scanoss/go-grpc-helper/pkg/grpc/server"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	myconfig "scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/protocol/grpc"
	"scanoss.com/dependencies/pkg/protocol/rest"
	"scanoss.com/dependencies/pkg/service"
)

//go:generate bash ../../get_version.sh
//go:embed version.txt
var version string

// getConfig checks command line args for option to feed into the config parser.
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

// RunServer runs the gRPC Dependency Server.
func RunServer() error {
	// Load command line options and config
	cfg, err := getConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	// Check mode to determine which logger to load
	err = zlog.SetupAppLogger(cfg.App.Mode, cfg.Logging.ConfigFile, cfg.App.Debug)
	if err != nil {
		return err
	}
	defer zlog.SyncZap()
	// Check if TLS/SSL should be enabled
	startTLS, err := files.CheckTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	if err != nil {
		return err
	}
	// Check if IP filtering should be enabled
	allowedIPs, deniedIPs, err := files.LoadFiltering(cfg.Filtering.AllowListFile, cfg.Filtering.DenyListFile)
	if err != nil {
		return err
	}
	zlog.S.Infof("Starting SCANOSS Dependency Service: %v", strings.TrimSpace(version))
	// Setup database connection pool
	db, err := gd.OpenDBConnection(cfg.Database.Dsn, cfg.Database.Driver, cfg.Database.User, cfg.Database.Passwd,
		cfg.Database.Host, cfg.Database.Schema, cfg.Database.SslMode)
	if err != nil {
		return err
	}
	if err = gd.SetDBOptionsAndPing(db); err != nil {
		return err
	}
	defer gd.CloseDBConnection(db)
	// Setup dynamic logging (if necessary)
	zlog.SetupAppDynamicLogging(cfg.Logging.DynamicPort, cfg.Logging.DynamicLogging)
	// Register the dependency service
	v2API := service.NewDependencyServer(db, cfg)
	ctx := context.Background()
	// Start the REST grpc-gateway if requested
	var srv *http.Server
	if len(cfg.App.RESTPort) > 0 {
		if srv, err = rest.RunServer(cfg, ctx, cfg.App.GRPCPort, cfg.App.RESTPort, allowedIPs, deniedIPs, startTLS); err != nil {
			return err
		}
	}
	// Start the gRPC service
	server, err := grpc.RunServer(cfg, v2API, cfg.App.GRPCPort, allowedIPs, deniedIPs, startTLS, version)
	if err != nil {
		return err
	}
	// graceful shutdown
	return gs.WaitServerComplete(srv, server)
}
