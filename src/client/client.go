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
	//address = "localhost:9000"
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

	//var name string
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
	/*
		startOfScanning := time.Now()
		var wg sync.WaitGroup
		for _, f := range files {
			wg.Add(1)
			log.Println(f.Name())

			go func(filename string) {
				defer wg.Done()
				if !strings.Contains(filename, "spdx") {
					thisFile := time.Now()
					log.Printf("envio %s", filename)
					scan := &pb.ScanRequest{}
					//scanresp := &pb.ScanResponse{}
					scan.File = filename
					scan.ClientContext = r.ClientContext
					currentFile := fmt.Sprintf("%s/%s", os.Args[2], filename)
					dat, err := ioutil.ReadFile(currentFile)
					scan.FileContent = string(dat)
					if err != nil {
						log.Print("sonamos")
					}
					scanresp, err := client.ScanWFP(ctx, scan)
					log.Printf("Got response for: %s:(%d bytes) and took %f", scan.File, len(scanresp.Results), float32(time.Since(thisFile).Milliseconds())/1000.0)
				}
			}(f.Name())
		}
		wg.Wait()
		log.Printf("Scanning the folder took %f", float32(time.Since(startOfScanning).Milliseconds())/1000.0)
	},*/
}
