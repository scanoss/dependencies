package models

import (
	"context"
	_ "fmt"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	myconfig "scanoss.com/dependencies/pkg/config"
)

func TestGolangDependencies(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	ctx := ctxzap.ToContext(context.Background(), zlog.L)
	s := ctxzap.Extract(ctx).Sugar()
	db := sqliteSetup(t)           // Setup SQL Lite DB
	conn := sqliteConn(t, ctx, db) // Get a connection from the pool
	err = LoadTestSQLData(db, ctx, conn)
	defer db.Close()
	defer CloseConn(conn)
	myConfig, err := myconfig.NewServerConfig(nil)
	if err != nil {
		t.Fatalf("failed to load Config: %v", err)
	}
	myConfig.Components.CommitMissing = true
	myConfig.Database.Trace = true

	var dependenciesModel *DependencyModel

	// Invalid ecosystem
	dependenciesModel = NewDependencyModel(ctx, s, db)
	unresolvedDependencies, err := dependenciesModel.GetDependencies("vue-phone", "1.0.8", "notExists")
	if err == nil {
		t.Errorf("FAILED: Expected an error when passing an invalid ecosystem, got err = nil")
	}

	unresolvedDependencies, err = dependenciesModel.GetDependencies("vue-phone", "1.0.9", "npm")
	if err != nil {
		t.Errorf("FAILED: Expected no errors, got err = %v", err)
	}
	if len(unresolvedDependencies) == 0 {
		t.Errorf("FAILED: Expected dependencies, got 0, err = %v", err)
	}
}
