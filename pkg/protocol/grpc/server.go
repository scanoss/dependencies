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
	"github.com/google/uuid"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"os"
	"os/signal"
	zlog "scanoss.com/dependencies/pkg/logger"
	"strings"
)

const RequestIDKey = "x-request-id"
const ResponseIDKey = "x-response-id"
const DebugEnableKey = "x-debug-enable"

// ContextPropagationUnaryServerInterceptor intercepts the incoming request and checks for a Request ID.
//If none exists, it creates it, adds it to the logging dataset and set the Response ID
func ContextPropagationUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			s := ctxzap.Extract(ctx).Sugar()
			dbEn := md[DebugEnableKey] // Check if we have a request to enable debug.
			if len(dbEn) > 0 {
				dbVal := strings.Trim(dbEn[0], " ")
				if dbVal != "" && strings.ToLower(dbVal) == "true" {
					//
				}
			}

			var reqId string
			xrId := md[RequestIDKey] // Check if we have a request ID. If not create one
			if len(xrId) > 0 {
				reqId = strings.Trim(xrId[0], " ")
			}
			if len(reqId) == 0 { // No Request ID, create one
				reqId = uuid.New().String()
				md.Set(RequestIDKey, reqId)
				s.Debugf("Creating Request ID: %v", reqId)
				ctx = metadata.NewIncomingContext(ctx, md) // Add the Request ID to the incoming metadata
			}
			ctxzap.AddFields(ctx, zap.String(RequestIDKey, reqId)) // Add Request ID to the logging
			ctx = context.WithValue(ctx, RequestIDKey, reqId)      // Add Request ID to current context
			ctx = metadata.NewOutgoingContext(ctx, md)             // Add the incoming metadata to any outgoing requests

			header := metadata.New(map[string]string{ResponseIDKey: reqId}) // Set the Response ID
			if err := grpc.SendHeader(ctx, header); err != nil {
				s.Debugf("Warning: Unable to set response header '%v' %v: %v", ResponseIDKey, reqId, err)
			}
		}
		return handler(ctx, req)
	}
}

// TODO Add proper service startup/shutdown here

// RunServer runs gRPC service to publish
func RunServer(ctx context.Context, v2API pb.DependenciesServer, port string) error {
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	// register service
	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_zap.UnaryServerInterceptor(zlog.L),
			ContextPropagationUnaryServerInterceptor(),
		)),
	)
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
