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

// Package grpc handles all the gRPC communication for the Dependency Service
// It takes care of starting and stopping the listener, etc.
package grpc

import (
	"context"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	zlog "scanoss.com/dependencies/pkg/logger"
)

// TODO Add proper service startup/shutdown here

// RunServer runs gRPC service to publish
func RunServer(ctx context.Context, v2API pb.DependenciesServer, port string) error {
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	// register service
	server := grpc.NewServer()
	pb.RegisterDependenciesServer(server, v2API)
	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			// sig is a ^C, handle it
			zlog.S.Info("shutting down gRPC server...")
			server.GracefulStop()
			<-ctx.Done()
		}
	}()
	// start gRPC server
	zlog.S.Info("starting gRPC server...")
	return server.Serve(listen)
}
