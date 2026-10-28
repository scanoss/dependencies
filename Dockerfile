FROM golang:1.25-bookworm AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go generate ./pkg/cmd/server.go
# cgo is required: the DuckDB driver used for the serving knowledge base is a
# cgo binding, so this cannot be a CGO_ENABLED=0 build.
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-w -s" -o ./scanoss-dependencies ./cmd/server

FROM debian:bookworm-slim

# The cgo binary links libstdc++ dynamically; ca-certificates keeps outbound TLS
# working (the pkg.go.dev live lookup needs it).
RUN apt-get update \
    && apt-get install -y --no-install-recommends libstdc++6 ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /app/scanoss-dependencies /app/scanoss-dependencies

EXPOSE 50051

ENTRYPOINT ["./scanoss-dependencies"]
#CMD ["--help"]
