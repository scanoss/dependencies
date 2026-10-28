// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2026 SCANOSS.COM
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
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcboeker/go-duckdb/v2"
	"go.uber.org/zap"
)

// Reloader repoints an open serving handle at the file its configured path
// resolves to now. Production rotates the serving export daily by repointing the
// full.duckdb symlink; a process keeps reading the file it opened until it
// reloads.
type Reloader interface {
	// Reload opens the file the path resolves to, when that is not the file in
	// use, and swaps it in. It reports the file now in use and whether it
	// changed. On error the current file stays in use.
	Reload() (file string, changed bool, err error)
}

// noReload is the Reloader of the drivers that are not reloaded: postgres is a
// server, and sqlite only backs the tests.
type noReload struct{}

func (noReload) Reload() (string, bool, error) { return "", false, nil }

// validateTimeout bounds the checks run on a new file before it is swapped in.
const validateTimeout = time.Minute

// generation is one opened serving file and the connections taken from it.
type generation struct {
	connector *duckdb.Connector
	file      string // the resolved path, never the symlink
	open      int    // connections taken from it and not yet closed
	retired   bool   // superseded: closed once its last connection is
}

// reloadingConnector is the driver.Connector behind a duckdb handle. The
// *sqlx.DB handed to the models stays the same for the life of the process;
// a reload swaps the generation new connections come from. Connections taken
// from a superseded generation finish what they are running, are then dropped by
// the pool (IsValid, ResetSession), and the superseded file is closed with the
// last of them.
type reloadingConnector struct {
	path string // as configured: the symlink production repoints
	mu   sync.Mutex
	cur  *generation
	// dropIdle closes the pool's idle connections, so a quiet service lets go of
	// the superseded file at once instead of when they idle out.
	dropIdle func()
}

// openGeneration opens the file path resolves to, read-only. It opens the
// resolved path, not the symlink: go-duckdb caches database instances by path,
// so reopening the symlink while the old instance is open returns the old file.
func openGeneration(path string) (*generation, error) {
	file, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %v: %w", path, err)
	}
	if _, err = os.Stat(file); err != nil {
		return nil, err
	}
	connector, err := duckdb.NewConnector(file+"?access_mode=read_only", nil)
	if err != nil {
		return nil, err
	}
	return &generation{connector: connector, file: file}, nil
}

func newReloadingConnector(path string) (*reloadingConnector, error) {
	g, err := openGeneration(path)
	if err != nil {
		return nil, err
	}
	return &reloadingConnector{path: path, cur: g}, nil
}

// Connect takes a connection from the current generation.
func (r *reloadingConnector) Connect(ctx context.Context) (driver.Conn, error) {
	r.mu.Lock()
	g := r.cur
	if g == nil {
		r.mu.Unlock()
		return nil, errors.New("the serving database is closed")
	}
	g.open++
	r.mu.Unlock()
	c, err := g.connector.Connect(ctx)
	if err != nil {
		r.release(g)
		return nil, err
	}
	dc, ok := c.(*duckdb.Conn)
	if !ok {
		_ = c.Close()
		r.release(g)
		return nil, fmt.Errorf("unexpected duckdb connection type %T", c)
	}
	return &conn{Conn: dc, gen: g, r: r}, nil
}

func (r *reloadingConnector) Driver() driver.Driver { return duckdb.Driver{} }

// Close closes the current generation once its connections are closed. sql.DB
// calls it from Close.
func (r *reloadingConnector) Close() error {
	r.mu.Lock()
	g := r.cur
	r.cur = nil
	r.mu.Unlock()
	if g != nil {
		r.retire(g)
	}
	return nil
}

// current reports whether g is the generation new connections come from.
func (r *reloadingConnector) current(g *generation) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cur == g
}

// release drops one connection of g, closing g if it was its last and g is
// retired.
func (r *reloadingConnector) release(g *generation) {
	r.mu.Lock()
	g.open--
	done := g.retired && g.open == 0
	r.mu.Unlock()
	if done {
		_ = g.connector.Close()
	}
}

// retire marks g superseded, closing it now if no connection is using it.
func (r *reloadingConnector) retire(g *generation) {
	r.mu.Lock()
	g.retired = true
	done := g.open == 0
	r.mu.Unlock()
	if done {
		_ = g.connector.Close()
	}
}

// Reload implements Reloader. The new file is validated before it is swapped
// in: it must answer, and carry every table the file in use carries.
func (r *reloadingConnector) Reload() (string, bool, error) {
	r.mu.Lock()
	old := r.cur
	r.mu.Unlock()
	if old == nil {
		return "", false, errors.New("the serving database is closed")
	}
	file, err := filepath.EvalSymlinks(r.path)
	if err != nil {
		return old.file, false, fmt.Errorf("failed to resolve %v: %w", r.path, err)
	}
	if file == old.file {
		return old.file, false, nil
	}
	next, err := openGeneration(r.path)
	if err != nil {
		return old.file, false, fmt.Errorf("failed to open %v: %w", file, err)
	}
	if err = validate(old, next); err != nil {
		_ = next.connector.Close()
		return old.file, false, fmt.Errorf("rejected %v: %w", next.file, err)
	}
	r.mu.Lock()
	if r.cur != old { // closed, or reloaded meanwhile
		r.mu.Unlock()
		_ = next.connector.Close()
		return old.file, false, errors.New("the serving database changed during the reload")
	}
	r.cur = next
	r.mu.Unlock()
	r.retire(old)
	if r.dropIdle != nil {
		r.dropIdle()
	}
	return next.file, true, nil
}

// validate checks that next answers and carries the tables old carries, so a
// truncated or foreign file is never swapped in.
func validate(old, next *generation) error {
	ctx, cancel := context.WithTimeout(context.Background(), validateTimeout)
	defer cancel()
	want, err := tables(ctx, old.connector)
	if err != nil {
		return fmt.Errorf("failed to list the tables in use: %w", err)
	}
	got, err := tables(ctx, next.connector)
	if err != nil {
		return err
	}
	for name := range want {
		if !got[name] {
			return fmt.Errorf("table %v is missing", name)
		}
	}
	return nil
}

// tables lists the tables of a generation through a driver connection of its
// own, outside the pool.
func tables(ctx context.Context, connector *duckdb.Connector) (map[string]bool, error) {
	c, err := connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = c.Close() }()
	queryer, ok := c.(driver.QueryerContext)
	if !ok {
		return nil, fmt.Errorf("unexpected duckdb connection type %T", c)
	}
	rows, err := queryer.QueryContext(ctx, "SELECT table_name FROM information_schema.tables", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	names := map[string]bool{}
	row := make([]driver.Value, len(rows.Columns()))
	for {
		if err = rows.Next(row); err != nil {
			if errors.Is(err, io.EOF) {
				return names, nil
			}
			return nil, err
		}
		if name, isString := row[0].(string); isString {
			names[name] = true
		}
	}
}

// conn is a duckdb connection tied to the generation it came from. Embedding
// *duckdb.Conn keeps every optional interface the driver implements
// (NamedValueChecker, QueryerContext, ...), which database/sql looks up on the
// connection.
type conn struct {
	*duckdb.Conn
	gen  *generation
	r    *reloadingConnector
	once sync.Once
}

// IsValid lets the pool drop a connection to a superseded file instead of
// reusing it.
func (c *conn) IsValid() bool { return c.r.current(c.gen) }

// ResetSession runs before an idle connection is reused.
func (c *conn) ResetSession(context.Context) error {
	if !c.IsValid() {
		return driver.ErrBadConn
	}
	return nil
}

func (c *conn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.r.release(c.gen) })
	return err
}

// openDuckDB opens the serving file behind a reloading connector.
func openDuckDB(path string) (*sqlx.DB, Reloader, error) {
	r, err := newReloadingConnector(path)
	if err != nil {
		return nil, nil, err
	}
	database := sqlx.NewDb(sql.OpenDB(r), DriverDuckDB)
	r.dropIdle = func() {
		database.SetMaxIdleConns(0)
		database.SetMaxIdleConns(embeddedMaxIdleConns)
	}
	return database, r, nil
}

// ReloadOnSIGHUP reloads the serving file on every SIGHUP - what
// `systemctl reload` sends - until stop is called. SIGHUP is taken for every
// driver, so a reload never terminates the process, which is Go's default.
// It returns the function that stops listening.
func ReloadOnSIGHUP(r Reloader, s *zap.SugaredLogger) func() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-signals:
				file, changed, err := r.Reload()
				switch {
				case err != nil:
					s.Errorf("SIGHUP: serving database not reloaded, still serving %v: %v", file, err)
				case changed:
					s.Infof("SIGHUP: serving database reloaded, now serving %v", file)
				case file == "":
					s.Infof("SIGHUP: this database driver is not reloaded")
				default:
					s.Infof("SIGHUP: serving database unchanged, still serving %v", file)
				}
			}
		}
	}()
	return func() {
		signal.Stop(signals)
		close(done)
	}
}
