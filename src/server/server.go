package main

import (
	//db "scanoss.com/dependencies/src/database"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/scanoss/papi/api/dependenciesv2"
	"google.golang.org/grpc"
	"gopkg.in/ini.v1"
	db "scanoss.com/dependencies/src/database"
	jp "scanoss.com/dependencies/src/jsonprocessing"
)

type Server struct {
	pb.DependenciesServer
}

const (
	defaultGrpcPort = 9000
)

func ServerInit(listenPort int) {

	strPort := fmt.Sprintf(":%d", listenPort)
	lis, err := net.Listen("tcp", strPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register services
	pb.RegisterDependenciesServer(grpcServer, &Server{})
	log.Printf("GRPC server listening on %v", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *Server) GetDependencies(ctx context.Context, in *pb.DependencyRequest) (*pb.DependencyResponse, error) {
	//log.Printf("Received: %v", in.)
	resp := jp.DepsProcess(in.Dependencies)
	println(resp)
	return &pb.DependencyResponse{Dependencies: resp}, nil
}

func main() {
	cfg, err := ini.Load("dependencies.conf")
	var host *string
	var port *int
	var user *string
	var password *string
	var dbname *string
	var grpcPort int

	//var file *string

	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}
	host = flag.String("host", "-", "Database host")
	port = flag.Int("port", 0, "Port Number")
	user = flag.String("user", "-", "Username")
	password = flag.String("password", "-", "User password")
	dbname = flag.String("dbname", "-", "Database name")

	//file = flag.String("file", "-", "Path to a JSON dependency file")
	//	purl := flag.String("purl", "-", "Purl project including version")
	flag.Parse()
	if (*host != "-") || (*port != 0) || (*user != "-") || (*password != "-") || (*dbname != "-") {
		var err int
		//fmt.Println("OVERIDE")
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
	//	portg := 9000
	//	grpcPort := &portg
	db.InitDB(*host, *port, *dbname, *user, *password)
	db.OpenDB()

	grpcPort = cfg.Section("grpcserver").Key("port").MustInt(9000)
	ServerInit(grpcPort)

	//

	//ServerInit(*grpcPort)
	/*
		if *purl != "-" {
			db.InitDB(*host, *port, *dbname, *user, *password)
			db.OpenDB()

			//	req := &dependenciesv2.DependencyRequest{}
			//prj := db.GetProjectInfo(*purl, 1)
			prj := db.GetProjectInfo(*purl, 1)
			fmt.Println(prj)
			//	fmt.Println(req)
			db.CloseDB()
		}

		if *file != "-" {

			db.InitDB(*host, *port, *dbname, *user, *password)
			db.OpenDB()
			dat, _ := os.ReadFile(*file)
			//	check(err)

			fmt.Println(jp.DepsProcess(string(dat)))
			//	req := &dependenciesv2.DependencyRequest{}
			//prj := db.GetProjectInfo(*purl, 1)
			//	prj := db.GetProjectInfo(*purl, 1)
			//fmt.Println(prj)
			//	fmt.Println(req)
			db.CloseDB()
		}*/
}
