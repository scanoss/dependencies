/*
 SPDX-License-Identifier: MIT
   Copyright (c) 2021, SCANOSS
   Permission is hereby granted, free of charge, to any person obtaining a copy
   of this software and associated documentation files (the "Software"), to deal
   in the Software without restriction, including without limitation the rights
   to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
   copies of the Software, and to permit persons to whom the Software is
   furnished to do so, subject to the following conditions:
   The above copyright notice and this permission notice shall be included in
   all copies or substantial portions of the Software.
   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
   OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
   THE SOFTWARE.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/scanoss/papi/api/dependenciesv2"
	"google.golang.org/grpc"
	"gopkg.in/ini.v1"
	db "scanoss.com/dependencies/src/database"
	jp "scanoss.com/dependencies/src/jsonprocessing"
)

func RequestServer(jsonData string, address string) {
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()
	client := pb.NewDependenciesClient(conn)
	req := &pb.DependencyRequest{}
	req.Dependencies = jsonData
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	r, err := client.GetDependencies(ctx, req)
	if err != nil {
		log.Fatalf("could not Send request: %v", err)
	}
	fmt.Println(r.Dependencies)
}

func main() {
	cfg, err := ini.Load("dependencies.conf")
	var host *string
	var port *int
	var user *string
	var password *string
	var dbname *string
	var file *string
	var useGrpc *bool
	var grpcHost *string

	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}
	host = flag.String("host", "-", "Database host")
	port = flag.Int("port", 0, "Port Number")
	user = flag.String("user", "-", "Username")
	password = flag.String("password", "-", "User password")
	dbname = flag.String("dbname", "-", "Database name")
	file = flag.String("file", "-", "Path to a JSON dependency file")
	purl := flag.String("purl", "-", "Purl project including version")
	useGrpc = flag.Bool("grpc", false, "Request information by using gRPC Server")
	grpcHost = flag.String("grpchost", "-", "GRPC server address:port")

	flag.Parse()
	if (*host != "-") || (*port != 0) || (*user != "-") || (*password != "-") || (*dbname != "-") {
		var err int
		err = 0
		if *host == "-" {
			fmt.Println("You must provide host address")
			err = 1
		}
		if *port == 0 {
			fmt.Println("You must provide  port number")
			err = 1
		}
		if *user == "-" {
			fmt.Println("You must provide a username")
			err = 1
		}
		if *password == "-" {
			fmt.Println("You must provide a password")
			err = 1
		}
		if err == 1 {
			os.Exit(err)
		}

	} else {

		*host = cfg.Section("database").Key("host").String()
		*port = cfg.Section("database").Key("port").MustInt(0)
		*dbname = cfg.Section("database").Key("dbname").String()

		*user = cfg.Section("database").Key("user").String()
		*password = cfg.Section("database").Key("password").String()
	}

	if *purl != "-" {
		db.InitDB(*host, *port, *dbname, *user, *password)
		db.OpenDB()
		prj := db.GetProjectInfo(*purl, 1)
		fmt.Println(prj)
		db.CloseDB()
	}

	if *file != "-" {

		db.InitDB(*host, *port, *dbname, *user, *password)
		db.OpenDB()
		dat, _ := os.ReadFile(*file)
		//	check(err)

		if *useGrpc {
			if *grpcHost == "-" {
				fmt.Println("You must define a gRPC host when using grpc option")
				os.Exit(1)
			} else {
				RequestServer(string(dat), *grpcHost)
			}

		} else {
			fmt.Println(jp.DepsProcess(string(dat)))
		}
		db.CloseDB()
	}

}
