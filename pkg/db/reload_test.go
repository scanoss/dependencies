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
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// The releases the two exports of a rotation answer.
const (
	day1 = "day-1"
	day2 = "day-2"
)

// writeExport creates a serving file whose db_version answers release.
func writeExport(t *testing.T, path, release string, extraTables ...string) {
	t.Helper()
	database, err := sql.Open(DriverDuckDB, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	stmts := []string{"CREATE TABLE db_version (release VARCHAR)", "INSERT INTO db_version VALUES ('" + release + "')"}
	for _, name := range extraTables {
		stmts = append(stmts, "CREATE TABLE "+name+" (id INTEGER)")
	}
	for _, stmt := range stmts {
		if _, err = database.Exec(stmt); err != nil {
			t.Fatalf("%v: %v", stmt, err)
		}
	}
}

// repoint swaps the symlink the way the rotation does: a new link renamed over
// the old one.
func repoint(t *testing.T, link, target string) {
	t.Helper()
	tmp := link + ".new"
	if err := os.Symlink(target, tmp); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, link); err != nil {
		t.Fatal(err)
	}
}

func release(t *testing.T, q sqlx.QueryerContext) string {
	t.Helper()
	var got string
	if err := sqlx.GetContext(context.Background(), q, &got, "SELECT release FROM db_version"); err != nil {
		t.Fatal(err)
	}
	return got
}

// rotation lays out two daily exports and the full.duckdb symlink at the first.
func rotation(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	writeExport(t, filepath.Join(dir, "full_1.duckdb"), day1, "component")
	writeExport(t, filepath.Join(dir, "full_2.duckdb"), day2, "component", "license")
	link := filepath.Join(dir, "full.duckdb")
	if err := os.Symlink("full_1.duckdb", link); err != nil {
		t.Fatal(err)
	}
	return dir, link
}

func openRotation(t *testing.T, link string) (*sqlx.DB, Reloader) {
	t.Helper()
	database, reloader, err := OpenReloadable(Config{Driver: DriverDuckDB, Path: link})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database, reloader
}

func TestReloadFollowsTheSymlink(t *testing.T) {
	dir, link := rotation(t)
	database, reloader := openRotation(t, link)
	if got := release(t, database); got != day1 {
		t.Fatalf("release = %q, want day-1", got)
	}

	// Repointing alone changes nothing: the process keeps the file it opened.
	repoint(t, link, "full_2.duckdb")
	if got := release(t, database); got != day1 {
		t.Fatalf("before the reload, release = %q, want day-1", got)
	}

	file, changed, err := reloader.Reload()
	if err != nil || !changed || file != filepath.Join(dir, "full_2.duckdb") {
		t.Fatalf("Reload() = %v, %v, %v", file, changed, err)
	}
	if got := release(t, database); got != day2 {
		t.Errorf("after the reload, release = %q, want day-2", got)
	}

	file, changed, err = reloader.Reload()
	if err != nil || changed || file != filepath.Join(dir, "full_2.duckdb") {
		t.Errorf("a second Reload() = %v, %v, %v; want no change", file, changed, err)
	}
}

// A query running across the reload finishes on the file it started on; the
// superseded file is closed with its last connection, so its disk space is freed
// once deleted.
func TestReloadLetsInFlightWorkFinish(t *testing.T) {
	_, link := rotation(t)
	database, reloader := openRotation(t, link)
	ctx := context.Background()
	busy, err := database.Connx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := reloader.(*reloadingConnector)
	if !ok {
		t.Fatalf("duckdb reloader is a %T", reloader)
	}
	old := r.cur

	repoint(t, link, "full_2.duckdb")
	if _, _, err = reloader.Reload(); err != nil {
		t.Fatal(err)
	}
	if got := release(t, busy); got != day1 {
		t.Errorf("the connection taken before the reload reads %q, want day-1", got)
	}
	if got := release(t, database); got != day2 {
		t.Errorf("new work reads %q, want day-2", got)
	}
	r.mu.Lock()
	open := old.open
	r.mu.Unlock()
	if open != 1 {
		t.Errorf("the superseded file has %d connections, want the busy one", open)
	}

	if err = busy.Close(); err != nil {
		t.Fatal(err)
	}
	// Returned to the pool, the stale connection is dropped rather than kept idle.
	r.mu.Lock()
	open = old.open
	r.mu.Unlock()
	if open != 0 {
		t.Errorf("the superseded file still has %d connections", open)
	}
}

// Idle connections to the superseded file are closed by the reload itself, so a
// service with no traffic does not hold the old file until they idle out.
func TestReloadDropsIdleConnections(t *testing.T) {
	_, link := rotation(t)
	database, reloader := openRotation(t, link)
	ctx := context.Background()
	// Leave a few idle connections in the pool.
	var held []*sqlx.Conn
	for i := 0; i < 3; i++ {
		c, err := database.Connx(ctx)
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, c)
	}
	for _, c := range held {
		_ = c.Close()
	}
	r, ok := reloader.(*reloadingConnector)
	if !ok {
		t.Fatalf("duckdb reloader is a %T", reloader)
	}
	old := r.cur
	repoint(t, link, "full_2.duckdb")
	if _, _, err := reloader.Reload(); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if old.open != 0 {
		t.Errorf("the superseded file still has %d idle connections", old.open)
	}
}

func TestReloadKeepsTheFileInUseOnABadFile(t *testing.T) {
	dir, link := rotation(t)
	database, reloader := openRotation(t, link)
	// Missing the component table the file in use carries.
	writeExport(t, filepath.Join(dir, "full_3.duckdb"), "day-3")
	if err := os.WriteFile(filepath.Join(dir, "full_4.duckdb"), []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"full_3.duckdb", "full_4.duckdb", "full_missing.duckdb"} {
		t.Run(target, func(t *testing.T) {
			repoint(t, link, target)
			file, changed, err := reloader.Reload()
			if err == nil || changed {
				t.Fatalf("Reload() = %v, %v, %v; want it rejected", file, changed, err)
			}
			if file != filepath.Join(dir, "full_1.duckdb") {
				t.Errorf("file in use = %v, want full_1.duckdb", file)
			}
			if got := release(t, database); got != day1 {
				t.Errorf("release = %q, want day-1", got)
			}
		})
	}
}

// Queries run while the file is swapped back and forth; run with -race.
func TestReloadUnderLoad(t *testing.T) {
	dir, link := rotation(t)
	writeExport(t, filepath.Join(dir, "full_3.duckdb"), "day-3", "component", "license")
	database, reloader := openRotation(t, link)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				var got string
				if err := database.GetContext(ctx, &got, "SELECT release FROM db_version"); err != nil && ctx.Err() == nil {
					errs <- err
					return
				}
			}
		}()
	}
	for i := 0; i < 20; i++ {
		target := "full_3.duckdb"
		if i%2 == 0 {
			target = "full_2.duckdb"
		}
		repoint(t, link, target)
		if _, _, err := reloader.Reload(); err != nil {
			t.Fatalf("reload %d: %v", i, err)
		}
	}
	cancel()
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("query failed during the reloads: %v", err)
	}
}

func TestReloadOnSIGHUP(t *testing.T) {
	_, link := rotation(t)
	database, reloader := openRotation(t, link)
	stop := ReloadOnSIGHUP(reloader, zap.NewNop().Sugar())
	defer stop()

	repoint(t, link, "full_2.duckdb")
	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for release(t, database) != day2 {
		if time.Now().After(deadline) {
			t.Fatal("SIGHUP did not reload the serving file")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestOtherDriversDoNotReload(t *testing.T) {
	database, reloader, err := OpenReloadable(Config{Driver: DriverSQLite, Path: filepath.Join(t.TempDir(), "serving.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if file, changed, rerr := reloader.Reload(); file != "" || changed || rerr != nil {
		t.Errorf("Reload() = %v, %v, %v; want a no-op", file, changed, rerr)
	}
}
