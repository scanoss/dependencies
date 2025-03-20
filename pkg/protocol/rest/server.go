// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2023 SCANOSS.COM
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

// Package rest handles all the REST communication for the Dependency Service
// It takes care of starting and stopping the listener, etc.
package rest

import (
	"context"
	"net/http"

	gw "github.com/scanoss/go-grpc-helper/pkg/grpc/gateway"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	myconfig "scanoss.com/dependencies/pkg/config"
)

// RunServer runs REST grpc gateway to forward requests onto the gRPC server.
func RunServer(config *myconfig.ServerConfig, ctx context.Context, grpcPort, httpPort string,
	allowedIPs, deniedIPs []string, startTLS bool) (*http.Server, error) {
	// configure the gateway for forwarding to gRPC
	srv, mux, grpcGateway, opts, err := gw.SetupGateway(grpcPort, httpPort, config.TLS.CertFile, "",
		allowedIPs, deniedIPs, config.Filtering.BlockByDefault, config.Filtering.TrustProxy,
		startTLS)
	if err != nil {
		return nil, err
	}
	// Open TCP port (in the background) and listen for requests
	go func() {
		ctx2, cancel := context.WithCancel(ctx)
		defer cancel()
		if err := pb.RegisterDependenciesHandlerFromEndpoint(ctx2, mux, grpcGateway, opts); err != nil {
			zlog.S.Panicf("Failed to start HTTP gateway %v", err)
		}
		gw.StartGateway(srv, config.TLS.CertFile, config.TLS.KeyFile, startTLS)
	}()
	return srv, nil
}
