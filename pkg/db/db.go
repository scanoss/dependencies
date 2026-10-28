// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2025 SCANOSS.COM
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

// Package db opens the serving knowledge base.
//
// The serving schema is driver-agnostic: every query in this service uses `?`
// placeholders and goes through sqlx.Rebind, so the same SQL runs on DuckDB,
// SQLite and PostgreSQL. DuckDB is the production backend (an embedded columnar
// file); SQLite backs the tests and PostgreSQL is kept for deployments that
// serve the same schema from a shared instance.
package db

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"                  // postgres driver
	_ "github.com/marcboeker/go-duckdb/v2" // duckdb driver (cgo)
	_ "modernc.org/sqlite"                 // sqlite driver (pure Go)
)

// Supported serving drivers.
const (
	// DriverDuckDB opens the serving file with the embedded DuckDB engine. It is
	// cgo, which is why the build sets CGO_ENABLED=1.
	DriverDuckDB = "duckdb"
	// DriverSQLite opens a SQLite file carrying the serving schema (tests).
	DriverSQLite = "sqlite"
	// DriverPostgres connects to a PostgreSQL instance serving the same schema.
	DriverPostgres = "postgres"
)

func init() {
	// go-duckdb takes `?` positional parameters, so Rebind must leave them alone.
	// Without this, sqlx defaults to the `?` style anyway for unknown drivers, but
	// registering it makes the contract explicit and survives a sqlx upgrade.
	sqlx.BindDriver(DriverDuckDB, sqlx.QUESTION)
}

// Config describes how to reach the serving database. File-based drivers
// (duckdb, sqlite) use Path; postgres uses DSN.
type Config struct {
	Driver string
	Path   string
	DSN    string
}

// Open connects to the serving database and verifies the connection.
//
// DuckDB is opened read-only: the serving file is large and immutable, and a
// read-only handle lets several in-process readers share it without WAL or temp
// writes.
func Open(cfg Config) (*sqlx.DB, error) {
	database, _, err := OpenReloadable(cfg)
	return database, err
}

// OpenReloadable is Open, also returning what reloads the handle when the
// configured path is repointed at another file. Only duckdb reloads; the other
// drivers get a Reloader that does nothing.
func OpenReloadable(cfg Config) (*sqlx.DB, Reloader, error) {
	var (
		database *sqlx.DB
		reloader Reloader = noReload{}
		err      error
	)
	switch cfg.Driver {
	case DriverDuckDB:
		if len(cfg.Path) == 0 {
			return nil, nil, fmt.Errorf("the %s driver requires a database path", DriverDuckDB)
		}
		database, reloader, err = openDuckDB(cfg.Path)
	case DriverSQLite:
		if len(cfg.Path) == 0 {
			return nil, nil, fmt.Errorf("the %s driver requires a database path", DriverSQLite)
		}
		database, err = sqlx.Open(DriverSQLite, cfg.Path)
	case DriverPostgres:
		if len(cfg.DSN) == 0 {
			return nil, nil, fmt.Errorf("the %s driver requires a dsn", DriverPostgres)
		}
		database, err = sqlx.Open(DriverPostgres, cfg.DSN)
	default:
		return nil, nil, fmt.Errorf("unsupported database driver %q (expected one of: %s, %s, %s)",
			cfg.Driver, DriverDuckDB, DriverSQLite, DriverPostgres)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the %s database: %w", cfg.Driver, err)
	}
	setPoolOptions(database, cfg.Driver)
	if err = database.Ping(); err != nil {
		_ = database.Close()
		return nil, nil, fmt.Errorf("failed to ping the %s database: %w", cfg.Driver, err)
	}
	return database, reloader, nil
}

// embeddedMaxIdleConns is the idle pool of the embedded engines.
const embeddedMaxIdleConns = 4

// setPoolOptions bounds the handle. The embedded engines run in this process, so
// a connection is a cursor over one open file rather than a socket to a server:
// a large pool buys nothing and costs memory per connection. PostgreSQL keeps
// the wider pool the service has always used.
func setPoolOptions(database *sqlx.DB, driver string) {
	database.SetConnMaxIdleTime(30 * time.Minute)
	database.SetConnMaxLifetime(time.Hour)
	switch driver {
	case DriverDuckDB, DriverSQLite:
		database.SetMaxIdleConns(embeddedMaxIdleConns)
		database.SetMaxOpenConns(16)
	default:
		database.SetMaxIdleConns(20)
		database.SetMaxOpenConns(200)
	}
}
