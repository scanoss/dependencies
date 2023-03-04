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
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jpillora/ipfilter"
	pb "github.com/scanoss/papi/api/dependenciesv2"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	myconfig "scanoss.com/dependencies/pkg/config"
)

// RunServer runs REST grpc gateway to forward requests onto the gRPC server
func RunServer(config *myconfig.ServerConfig, ctx context.Context, grpcPort, httpPort string, allowedIPs, deniedIPs []string, startTLS bool) (*http.Server, error) {
	mux := runtime.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: mux,
	}
	if len(allowedIPs) > 0 || len(deniedIPs) > 0 { // Configure the list of allowed/denied IPs to connect
		zlog.S.Debugf("Filtering requests by allowed: %v, denied: %v, block-by-default: %v", allowedIPs, deniedIPs, config.Filtering.BlockByDefault)
		handler := ipfilter.Wrap(mux, ipfilter.Options{AllowedIPs: allowedIPs, BlockedIPs: deniedIPs,
			BlockByDefault: config.Filtering.BlockByDefault, TrustProxy: config.Filtering.TrustProxy,
		})
		srv.Handler = handler // assign the filtered handler
	}
	var opts []grpc.DialOption
	if startTLS {
		creds, err := credentials.NewClientTLSFromFile(config.TLS.CertFile, "")
		if err != nil {
			zlog.S.Errorf("Problem loading TLS file: %s - %v", config.TLS.CertFile, err)
			return nil, fmt.Errorf("failed to load TLS credentials from file")
		}
		opts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	} else {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	// Open TCP port (in the background) and listen for requests
	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		if err := pb.RegisterDependenciesHandlerFromEndpoint(ctx, mux, "localhost:"+grpcPort, opts); err != nil {
			zlog.S.Panicf("Failed to start HTTP gateway %v", err)
		}
		var httpErr error
		if startTLS {
			zlog.S.Infof("starting REST server with TLS on %v ...", srv.Addr)
			httpErr = srv.ListenAndServeTLS(config.TLS.CertFile, config.TLS.KeyFile)
		} else {
			zlog.S.Infof("starting REST server on %v ...", srv.Addr)
			httpErr = srv.ListenAndServe()
		}
		if httpErr != nil && fmt.Sprintf("%s", httpErr) != "http: Server closed" {
			zlog.S.Panicf("issue encountered when starting service: %v", httpErr)
		}
	}()
	return srv, nil
}
