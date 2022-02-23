FROM golang:1.17 as build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go build -o ./scanoss-dependencies ./cmd/server/

FROM debian:buster-slim

WORKDIR /app
 
COPY --from=build /app/ /app/

EXPOSE 50051

CMD [ "./scanoss-dependencies" ]
