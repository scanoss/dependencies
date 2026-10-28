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

package db

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestOpenRejectsBadConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantMsg string
	}{
		{"unknown driver", Config{Driver: "mysql"}, "unsupported database driver"},
		{"empty driver", Config{}, "unsupported database driver"},
		{"duckdb without a path", Config{Driver: DriverDuckDB}, "requires a database path"},
		{"sqlite without a path", Config{Driver: DriverSQLite}, "requires a database path"},
		{"postgres without a dsn", Config{Driver: DriverPostgres}, "requires a dsn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Open(tt.cfg)
			if err == nil {
				t.Fatalf("Open(%+v) succeeded, want an error", tt.cfg)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error = %q, want it to mention %q", err, tt.wantMsg)
			}
		})
	}
}

func TestOpenSQLite(t *testing.T) {
	database, err := Open(Config{Driver: DriverSQLite, Path: filepath.Join(t.TempDir(), "serving.db")})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	// The embedded drivers get the smaller pool: a connection is a cursor over a
	// local file, not a socket to a server.
	if got := database.Stats().MaxOpenConnections; got != 16 {
		t.Errorf("MaxOpenConnections = %d, want 16", got)
	}
}

// The whole serving layer relies on `?` placeholders surviving Rebind, which is what
// lets one query run on all three backends.
func TestRebindKeepsQuestionMarksForDuckDB(t *testing.T) {
	const query = "SELECT purl FROM component WHERE purl = ? AND vendor = ?"
	if got := sqlx.Rebind(sqlx.BindType(DriverDuckDB), query); got != query {
		t.Errorf("Rebind rewrote the duckdb query to %q; the driver takes positional `?`", got)
	}
}

// DuckDB is opened read-only: the serving file is large and immutable, and a
// read-only handle avoids WAL and temp writes while letting readers share it.
func TestOpenDuckDBReadOnlyRejectsAMissingFile(t *testing.T) {
	_, err := Open(Config{Driver: DriverDuckDB, Path: filepath.Join(t.TempDir(), "absent.duckdb")})
	if err == nil {
		t.Fatal("opening a missing file read-only should fail rather than create it")
	}
}
