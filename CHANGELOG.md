# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.16.0] - 2026-10-28
### Changed
- **The service now reads the serving knowledge base instead of the mining database.** The mining path is gone; there is no configuration that selects it. Endpoint routes and response shapes are unchanged
- The database is configured with `Database.Driver` (`DB_DRIVER`: `duckdb`, the default, `sqlite` or `postgres`) and the new `Database.Path` (`DB_PATH`), the serving file, opened read-only. `postgres` still takes `Dsn` or the host settings
- Component and version resolution no longer goes through `go-component-helper`'s `GetComponentsVersion`, which resolves against `all_urls`, `mines` and `versions` - none of which exist in the serving schema. It is replaced by `pkg/resolver`, which looks the purl up in every spelling the export stores: `pkg:npm/@angular/core` now resolves as well as `pkg:npm/%40angular/core`, where it answered `NO_INFO`
- **Licences read `component_version_license`** by (purl_id, version), through `license` and `"SPDX_license_data"`, rather than `all_urls`, `projects` and `licenses`. Whether a licence is SPDX-approved is whether its identifier is in the SPDX registry. A release with more than one licence row (about 1% of releases) reports all of them, where mining reported one; compound identifiers (`MIT/Apache-2.0`) are still split
- There is no component-level licence any more: mining fell back to `projects` when a release had none, and serving has no such table. Instead, a release with no licence of its own borrows one, in the order `scanoss.api`'s licence lookup uses, and reports it with `info_code` `WARNING` and an `info_message` naming where it came from:
  1. the source component (`component.source_purl_id`) at the same version - "Showing results from the source code purl (…)";
  2. the nearest version of the component that has a licence - "Showing results from version …";
  3. the nearest version of the source component that has one - "Showing results from the source code purl (…) version …".

  A status that already reports a problem (`REQUIREMENT_NOT_MET`, `VERSION_NOT_FOUND`) keeps its code and gains the message. 43% of `component_version_license` rows point at license 523, which has neither a name nor an identifier - mining's "no licence" row, so mining answered these releases without a licence as well. A release whose only rows say there is no licence (523, "None detected", "UNKNOWN", "unknown") also borrows, and keeps them when nothing can be borrowed; those rows are never borrowed. Nearness is semver distance, ties to the higher version, with versions that differ only in build metadata (golang pseudo-versions) ordered by it. Measured on samples of the export, the fallback recovers a licence for 17-21% of such releases, 36% for components with 1,000 stars or more; a release with nothing to borrow still reports `NO_INFO`
- **Transitive dependencies read `component_version_dependency`** instead of the per-ecosystem `<ecosystem>_dependencies` JSON tables. Every scope is followed, as before, and a requirement still resolves to the first version it names. Rust crates are looked up under the export's `cargo` purl type; responses still say `crates`
- Golang modules are ordinary serving components. The export carries them at pseudo-versions only, so a requested tag is answered live from pkg.go.dev, then from the GitHub repository the module path names, as `golang_projects` used to; a tag answered that way reports `SUCCESS`. The live lookup of a module is given up after 5 seconds, both of its attempts together, and the GitHub repository and the knowledge base answer instead: a slow pkg.go.dev otherwise held the whole request (one call was seen taking 19 s)
- An exact requested version is reported as the caller spelled it. The export drops the leading `v` mining stored (`v5.4.5` is `5.4.5`), and a golang version without it is not the version asked for
- Binaries are built with `CGO_ENABLED=1`; the release builds in a `golang:1.25-bookworm` container with an arm64 cgo cross toolchain, and the Docker image builds on `golang:1.25-bookworm` and runs on `debian:bookworm-slim` with `libstdc++6` and `ca-certificates`. The DuckDB driver is a cgo binding, so a pure-Go build can no longer be produced
- Upgraded `scanoss/go-grpc-helper` to `v0.17.0`; added `marcboeker/go-duckdb/v2` `v2.4.3`
- The tests run on SQLite against a transcription of the serving schema (`pkg/models/test_schema.go`), with data-only fixtures
### Removed
- `COMP_COMMIT_MISSING` / `Components.CommitMissing`: the serving knowledge base is read-only, so live pkg.go.dev answers are no longer written back. The live lookup itself remains
### Fixed
- Transitive lookups of namespaced packages: the package identifier kept the namespace only for maven, so a scoped npm entry (`pkg:npm/@angular/core`) or a composer one (`pkg:composer/laravel/framework`) was looked up by its bare name and answered "not found"
- The service embeds `UnimplementedDependenciesServer` instead of the papi interface, so a method added to the contract answers `Unimplemented` instead of dereferencing a nil interface and killing the process

### Security
- Upgraded `google.golang.org/grpc` to v1.83.1, `golang.org/x/net` to v0.57.0 and the OpenTelemetry modules (`otel`, `otel/sdk`, `otlptrace`, `otlptracegrpc`) to v1.45.0, and `github.com/antchfx/xpath` to v1.3.6 to fix open Dependabot security advisories

## [0.15.0] - 2026-06-22
### Added
- `/health` liveness endpoint (GET) on the REST gateway
### Changed
- Upgraded `scanoss/go-grpc-helper` to `v0.16.0`

## [0.14.0] - 2026-04-16
### Changed
- Replaced `error_code`/`error_message` with `info_code`/`info_message` in dependency response
- Use `domain.StatusCode.String()` directly for component status codes (deprecated `StatusCodeToErrorCode` mapping)
- Renamed status codes: `ComponentWithoutInfo` → `NoInfo`, `VersionNotFound` → `RequirementNotMet`
- Updated dependencies to the latest versions

## [0.13.0] - 2026-03-23
### Added
- Added per-component error status (`error_code`, `error_message`) to dependency response
- Added `resolveComponentVersion` to determine the best version for the dependency response based on pinned version, semver requirement, or URL lookup
- Added `resolveLicenses` method to handle compound license ID resolution
- Added `github.com/scanoss/go-component-helper` to v0.5.0
### Changed
- Completed `convertDependencyOutput` to build proto response with dependencies, licenses, and error status
- Updated model constructors to use `*sqlx.DB` instead of `*sqlx.Conn`
- Upgraded `package-url/packageurl-go` to v.1.5
- Upgraded `google.golang.org/grpc` to v1.79.2
- Updated CI workflows (`go-ci`, `release`) to `actions/checkout@v4`, `actions/setup-go@v5`, and `go-version-file`
- Updated `golangci-lint-action` to v9 and linter config to v2 format
- Updated `golangci-lint` Docker version to v2.10.1 in Makefile
- Updated `github.com/scanoss/go-purl-helper` to v0.3.0
- Upgraded `github.com/lib/pq` from v1.11.2 to v1.12.0
- Upgraded `github.com/scanoss/papi` from v0.32.0 to v0.32.1
- Upgraded `go.opentelemetry.io/otel` from v1.40.0 to v1.42.0
- Upgraded `go.opentelemetry.io/otel/metric` from v1.40.0 to v1.42.0
- Upgraded `modernc.org/sqlite` from v1.46.1 to v1.47.0
### Removed
- Removed `github.com/Masterminds/semver/v3` direct dependency

## [0.12.2] - 2026-01-12
### Fixed
- Included missing component URLs

## [0.12.1] - 2025-12-23
### Fixed
- Fixed fallback URL lookup to iterate through all mine IDs

## [0.12.0] - 2025-12-09
### Fixed
- Resolved project license when no match is found for a specific version

## [0.11.0] - 2025-12-09
### Added
- Added `requirement` field on dependency response

## [0.10.1] - 2025-11-27
- Fix bugs with conan packages 

## [0.10.0] - 2025-09-16
### Changed
- Updated `github.com/scanoss/papi` to v0.20.1
- **BREAKING CHANGE**: Changed `/v2/dependencies/transitive` to `/v2/dependencies/transitive/components`
- Deprecated `/v2/dependencies/dependencies` endpoint (use `/v2/licenses/components` instead)

## [0.9.0] - 2025-08-29
### Changed
- Updated `github.com/scanoss/papi` to v0.17.0 
- Replaced REST endpoint POST `/api/v2/dependencies/dependencies` by `/v2/dependencies/dependencies`
- Replaced REST endpoint POST `/api/v2/dependencies/transitive` by `/v2/dependencies/transitive`
- Replaced REST endpoint POST `/api/v2/dependencies/echo` by `/v2/dependencies/echo`

## [0.8.0] - 2025-06-26
### Added
- Added transitive dependency service
- Added sqlite configuration on env-setup.sh
### Changed 
- Changed sqlite database driver to `modernc.org/sqlite`
- Upgraded project dependencies

## [0.7.4] - 2025-06-05
### Fixed
- Fixed empty dependency version 

## [0.7.3] - 2024-09-04
### Changed
- Updated license
- Upgraded project dependencies

## [0.7.2] - 2023-11-27
### Added
- Added SQL tracing support for advanced debug
### Fixed
- Fixed issue with golang license reporting for certain components

## [0.7.1] - 2023-11-20
### Added
- Added Open Telemetry spans/traces/metrics
- Upgraded to Go 1.20

## [0.7.0] - 2023-03-14
### Added
- Updates based on golanglint-ci feedback

## [0.6.0] - 2022-10-17
### Added
- Added gRPC middleware logging support including request ID

## [0.5.0] - 2022-10-06
### Added
- Added license split support

## [0.4.0] - 2022-06-08
### Added
- Added golang project lookup support

## [0.3.0] - 2022-05-30
### Added
- Cleaned up license and version search

## [0.2.0] - 2022-05-01
### Added
- Added semver filtering

## [0.0.1] - 2022-04-22
### Added
- Added version search support

[0.2.0]: https://github.com/scanoss/dependencies/compare/v0.0.1...v0.2.0
[0.3.0]: https://github.com/scanoss/dependencies/compare/v0.2.0...v0.3.0
[0.0.1]: https://github.com/scanoss/dependencies/compare/v0.0.0...v0.0.1
[0.4.0]: https://github.com/scanoss/dependencies/compare/v0.3.0...v0.4.0
[0.5.0]: https://github.com/scanoss/dependencies/compare/v0.4.0...v0.5.0
[0.6.0]: https://github.com/scanoss/dependencies/compare/v0.5.0...v0.6.0
[0.7.0]: https://github.com/scanoss/dependencies/compare/v0.6.0...v0.7.0
[0.7.1]: https://github.com/scanoss/dependencies/compare/v0.7.0...v0.7.1
[0.7.2]: https://github.com/scanoss/dependencies/compare/v0.7.1...v0.7.2
[0.7.3]: https://github.com/scanoss/dependencies/compare/v0.7.2...v0.7.3
[0.7.4]: https://github.com/scanoss/dependencies/compare/v0.7.3...v0.7.4
[0.8.0]: https://github.com/scanoss/dependencies/compare/v0.7.4...v0.8.0
[0.9.0]: https://github.com/scanoss/dependencies/compare/v0.8.0...v0.9.0
[0.10.0]: https://github.com/scanoss/dependencies/compare/v0.9.0...v0.10.0
[0.10.1]: https://github.com/scanoss/dependencies/compare/v0.10.0...v0.10.1
[0.11.0]: https://github.com/scanoss/dependencies/compare/v0.10.1...v0.11.0
[0.12.0]: https://github.com/scanoss/dependencies/compare/v0.11.0...v0.12.0
[0.12.1]: https://github.com/scanoss/dependencies/compare/v0.12.0...v0.12.1
[0.12.2]: https://github.com/scanoss/dependencies/compare/v0.12.1...v0.12.2
[0.13.0]: https://github.com/scanoss/dependencies/compare/v0.12.2...v0.13.0
[0.14.0]: https://github.com/scanoss/dependencies/compare/v0.13.0...v0.14.0
[0.15.0]: https://github.com/scanoss/dependencies/compare/v0.14.0...v0.15.0
[0.16.0]: https://github.com/scanoss/dependencies/compare/v0.15.0...v0.16.0
