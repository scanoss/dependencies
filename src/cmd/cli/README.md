# Dependency scan CLI tool
This tool is aimed to test DB connection and/or gRPC server. 
dep-scan could be executed without compilation from repo root using:

> go run src/cmd/cli/dep-scan.go 

You can do a dependecy scan using a dependecy JSON file:
> go run src/cmd/cli/dep-scan.go -file=path/to/JSON/File

You can also do the same scan but accross a gRPC server (that must be active)

>go run src/cmd/cli/dep-scan.go -file= *<path/to/JSON/File>* -grpc -grpchost= *<address:port>*
