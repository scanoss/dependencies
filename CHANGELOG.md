# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Upcoming changes...

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