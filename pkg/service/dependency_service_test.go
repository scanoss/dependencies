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

package service

import (
	"context"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	common "github.com/scanoss/papi/api/commonv2"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	"reflect"
	"scanoss.com/dependencies/pkg/models"
	"testing"
)

func TestDependencyServer_Echo(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	s := NewDependencyServer(db)

	type args struct {
		ctx context.Context
		req *common.EchoRequest
	}
	tests := []struct {
		name    string
		s       pb.DependenciesServer
		args    args
		want    *common.EchoResponse
		wantErr bool
	}{
		{
			name: "Echo",
			s:    s,
			args: args{
				ctx: ctx,
				req: &common.EchoRequest{Message: "Hello there!"},
			},
			want: &common.EchoResponse{Message: "Hello there!"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.Echo(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("service.Echo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("service.Echo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDependencyServer_GetDependencies(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	err = models.LoadTestSqlData(db)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when loading test data", err)
	}
	s := NewDependencyServer(db)

	type args struct {
		ctx context.Context
		req *pb.DependencyRequest
	}
	tests := []struct {
		name    string
		s       pb.DependenciesServer
		args    args
		want    *pb.DependencyResponse
		wantErr bool
	}{
		{
			name: "Get Deps Simple True",
			s:    s,
			args: args{
				ctx: ctx,
				req: &pb.DependencyRequest{Dependencies: "some rubbish", Depth: 1},
			},
			want: &pb.DependencyResponse{Dependencies: "", Status: &common.StatusResponse{Status: common.StatusCode_FAILED, Message: "No dependency request data supplied"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.GetDependencies(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("service.GetDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("service.GetDependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}
