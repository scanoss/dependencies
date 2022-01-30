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
	"flag"
	"fmt"
	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	myconfig "scanoss.com/dependencies/pkg/config"
	"scanoss.com/dependencies/pkg/protocol/grpc"
	"scanoss.com/dependencies/pkg/service"
	"time"
)

// getConfig checks command line args for option to feed into the config parser
func getConfig() (*myconfig.ServerConfig, error) {
	var jsonConfig, envConfig string
	flag.StringVar(&jsonConfig, "json-config", "", "Application JSON config")
	flag.StringVar(&envConfig, "env-config", "", "Application dot-ENV config")
	flag.Parse()
	var feeders []config.Feeder
	if len(jsonConfig) > 0 {
		feeders = append(feeders, feeder.Json{Path: jsonConfig})
	}
	if len(envConfig) > 0 {
		feeders = append(feeders, feeder.DotEnv{Path: envConfig})
	}
	myConfig, err := myconfig.NewServerConfig(feeders)
	return myConfig, err
}

// RunServer runs the gRPC Dependency Server
func RunServer() error {
	cfg, err := getConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	fmt.Printf("Config: %+v\n", cfg)
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
	db, err := sqlx.Open(cfg.Database.Driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	db.SetConnMaxIdleTime(30 * time.Minute)
	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxIdleConns(20)
	db.SetMaxOpenConns(100)
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}
	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			fmt.Printf("Error closing DB: %v", err)
		}
	}(db)
	v2API := service.NewDependencyServer(db)
	ctx := context.Background()
	return grpc.RunServer(ctx, v2API, cfg.App.Port)
}
