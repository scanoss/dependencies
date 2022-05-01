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
	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap/zapcore"
	"os"
	myconfig "scanoss.com/dependencies/pkg/config"
	zlog "scanoss.com/dependencies/pkg/logger"
	"scanoss.com/dependencies/pkg/protocol/grpc"
	"scanoss.com/dependencies/pkg/service"
	"strings"
	"time"
)

//go:generate bash ../../get_version.sh
//go:embed version.txt
var version string

// getConfig checks command line args for option to feed into the config parser
func getConfig() (*myconfig.ServerConfig, error) {
	var jsonConfig, envConfig string
	flag.StringVar(&jsonConfig, "json-config", "", "Application JSON config")
	flag.StringVar(&envConfig, "env-config", "", "Application dot-ENV config")
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
	switch strings.ToLower(cfg.App.Mode) {
	case "prod":
		var err error
		if cfg.App.Debug {
			err = zlog.NewSugaredProdLoggerLevel(zapcore.DebugLevel)
		} else {
			err = zlog.NewSugaredProdLogger()
		}
		if err != nil {
			return fmt.Errorf("failed to load logger: %v", err)
		}
		zlog.L.Debug("Running with debug enabled")
	default:
		if err := zlog.NewSugaredDevLogger(); err != nil {
			return fmt.Errorf("failed to load logger: %v", err)
		}
	}
	defer zlog.SyncZap()
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
	db.SetMaxOpenConns(100)
	err = db.Ping()
	if err != nil {
		zlog.S.Errorf("Failed to ping database: %v", err)
		return fmt.Errorf("failed to ping database: %v", err)
	}
	defer closeDbConnection(db)
	v2API := service.NewDependencyServer(db)
	ctx := context.Background()
	return grpc.RunServer(ctx, v2API, cfg.App.Port)
}
