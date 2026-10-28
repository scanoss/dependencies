-- Test data only. The schema comes from testSchemaDDL in pkg/models/test_schema.go; do not add DDL here.
-- Serving `component_version_license`: declared licences per release. sort-paths
-- has none; dual carries two; goffice carries a compound identifier.
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-lodash', '4.17.0', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-lodash', '4.17.21', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-electron-debug', '3.1.0', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-electron-debug', '3.2.0', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-isbinaryfile', '4.0.8', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-isbinaryfile', '4.0.10', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-goffice', '0.10.52-4', 15, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-vue-phone', '1.0.9', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-scoped', '1.0.0', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-dual', '1.0.0', 2, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-dual', '1.0.0', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-maven-lib', '1.0.0', 2, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-zap', 'v1.27.0', 1, 0);
INSERT INTO component_version_license (purl_id, version, license_id, source) VALUES ('uuid-gin', 'v1.9.1', 1, 0);
