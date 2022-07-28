FROM golang:1.17 as build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go generate ./pkg/cmd/server.go
#RUN go build -o ./scanoss-dependencies ./cmd/server
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./scanoss-dependencies ./cmd/server


FROM debian:buster-slim

WORKDIR /app
 
COPY --from=build /app/scanoss-dependencies /app/scanoss-dependencies

EXPOSE 50051

ENTRYPOINT ["./scanoss-dependencies"]
#CMD ["--help"]
