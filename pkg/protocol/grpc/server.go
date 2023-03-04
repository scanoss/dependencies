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
	"fmt"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/jpillora/ipfilter"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	"github.com/scanoss/zap-logging-helper/pkg/grpc/interceptor"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	myconfig "scanoss.com/dependencies/pkg/config"
)

// RunServer runs gRPC service to serve incoming requests
func RunServer(config *myconfig.ServerConfig, v2API pb.DependenciesServer, port string, allowedIPs, deniedIPs []string, startTLS bool) (*grpc.Server, error) {
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}
	var interceptors []grpc.UnaryServerInterceptor
	// Configure the list of allowed/denied IPs to connect
	if len(allowedIPs) > 0 || len(deniedIPs) > 0 {
		ipFilter := ipfilter.New(ipfilter.Options{AllowedIPs: allowedIPs, BlockedIPs: deniedIPs,
			BlockByDefault: config.Filtering.BlockByDefault, TrustProxy: config.Filtering.TrustProxy,
		})
		interceptors = append(interceptors, ipFilter.IPFilterUnaryServerInterceptor())
	}
	interceptors = append(interceptors, grpczap.UnaryServerInterceptor(zlog.L))
	interceptors = append(interceptors, interceptor.ContextPropagationUnaryServerInterceptor()) // Needs to be called after UnaryServerInterceptor to make sure the logger is set
	var opts []grpc.ServerOption
	if startTLS {
		creds, err := credentials.NewServerTLSFromFile(config.TLS.CertFile, config.TLS.KeyFile)
		if err != nil {
			zlog.S.Errorf("Problem loading TLS file: %s - %v", config.TLS.CertFile, err)
			return nil, fmt.Errorf("failed to load TLS credentials from file")
		}
		opts = append(opts, grpc.Creds(creds))
	}
	opts = append(opts, grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(interceptors...)))
	// register service
	//server := grpc.NewServer(grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(interceptors...)))
	server := grpc.NewServer(opts...)
	pb.RegisterDependenciesServer(server, v2API)
	go func() {
		var httpErr error
		if startTLS {
			zlog.S.Infof("starting gRPC server with TLS on %v ...", listen.Addr())
			//httpErr = srv.ListenAndServeTLS(config.TLS.CertFile, config.TLS.KeyFile)
		} else {
			zlog.S.Infof("starting gRPC server on %v ...", listen.Addr())
		}
		httpErr = server.Serve(listen)
		if httpErr != nil && fmt.Sprintf("%s", httpErr) != "http: Server closed" {
			zlog.S.Panicf("issue encountered when starting service: %v", httpErr)
		}
	}()
	return server, nil
}
