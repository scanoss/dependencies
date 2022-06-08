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

package config

import (
	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
)

const (
	defaultGrpcPort = "50051"
)

// ServerConfig is configuration for Server
type ServerConfig struct {
	App struct {
		Name  string `env:"APP_NAME"`
		Port  string `env:"APP_PORT"`
		Debug bool   `env:"APP_DEBUG"`
		Mode  string `env:"APP_MODE"` // dev or prod
	}
	Database struct {
		Driver  string `env:"DB_DRIVER"`
		Host    string `env:"DB_HOST"`
		User    string `env:"DB_USER"`
		Passwd  string `env:"DB_PASSWD"`
		Schema  string `env:"DB_SCHEMA"`
		SslMode string `env:"DB_SSL_MODE"` // enable/disable
		Dsn     string `env:"DB_DSN"`
	}
	Components struct {
		CommitMissing bool `env:"COMP_COMMIT_MISSING"` // Write component details to the DB if they are looked up live
	}
}

// NewServerConfig loads all config options and return a struct for use
func NewServerConfig(feeders []config.Feeder) (*ServerConfig, error) {
	cfg := ServerConfig{}
	setServerConfigDefaults(&cfg)
	c := config.New()
	for _, f := range feeders {
		c.AddFeeder(f)
	}
	c.AddFeeder(feeder.Env{})
	c.AddStruct(&cfg)
	err := c.Feed()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// setServerConfigDefaults attempts to set reasonable defaults for the server config
func setServerConfigDefaults(cfg *ServerConfig) {
	cfg.App.Name = "SCANOSS Dependency Server"
	cfg.App.Port = defaultGrpcPort
	cfg.App.Mode = "dev"
	cfg.App.Debug = false
	cfg.Database.Driver = "postgres"
	cfg.Database.Host = "localhost"
	cfg.Database.User = "scanoss"
	cfg.Database.Schema = "scanoss"
	cfg.Database.SslMode = "disable"
	cfg.Components.CommitMissing = false
}
