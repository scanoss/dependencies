module scanoss.com/dependencies

go 1.20

require (
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/golobby/config/v3 v3.4.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/guseggert/pkggodev-client v0.0.0-20211029144512-2df8afe3ebe4
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/package-url/packageurl-go v0.1.2
	github.com/scanoss/go-grpc-helper v0.3.0
	github.com/scanoss/go-purl-helper v0.2.0
	github.com/scanoss/papi v0.2.0
	github.com/scanoss/zap-logging-helper v0.3.0
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/metric v1.24.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.62.1
)

//replace github.com/scanoss/papi => ../papi
//replace github.com/scanoss/go-grpc-helper => ../go-grpc-helper
//replace github.com/scanoss/zap-logging-helper => ../zap-logging-helper

require (
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/PuerkitoBio/goquery v1.7.1 // indirect
	github.com/andybalholm/cascadia v1.2.0 // indirect
	github.com/antchfx/htmlquery v1.2.4 // indirect
	github.com/antchfx/xmlquery v1.3.7 // indirect
	github.com/antchfx/xpath v1.2.0 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gocolly/colly/v2 v2.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golobby/cast v1.3.3 // indirect
	github.com/golobby/dotenv v1.3.2 // indirect
	github.com/golobby/env/v2 v2.2.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.1 // indirect
	github.com/kennygrant/sanitize v1.2.4 // indirect
	github.com/phuslu/iploc v1.0.20230201 // indirect
	github.com/saintfish/chardet v0.0.0-20120816061221-3af4cd4741ca // indirect
	github.com/scanoss/ipfilter/v2 v2.0.2 // indirect
	github.com/temoto/robotstxt v1.1.2 // indirect
	github.com/tomasen/realip v0.0.0-20180522021738-f0c99a92ddce // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.46.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.21.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0 // indirect
	go.opentelemetry.io/otel/sdk v1.21.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.21.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Details of how to use the "replace" command for local development
// https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive
// ie. replace github.com/scanoss/papi => ../papi
// require github.com/scanoss/papi v0.0.0-unpublished
