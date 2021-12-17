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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	pb "github.com/scanoss/papi/api/dependenciesv2"
	"google.golang.org/grpc"
)

/**server Address:port*/
const (
	address = "localhost:9000"
)

func main() {
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewDependenciesClient(conn)

	req := &pb.DependencyRequest{}

	jsonFile := fmt.Sprintf("%s", os.Args[1])
	jsonData, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		log.Print("sonamos")
	}
	req.Dependencies = string(jsonData)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	r, err := client.GetDependencies(ctx, req)
	if err != nil {
		log.Fatalf("could not Send request: %v", err)
	}
	fmt.Println(r.Dependencies)
}
