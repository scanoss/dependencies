-- Test data only. The schema comes from testSchemaDDL in pkg/models/test_schema.go; do not add DDL here.
-- Serving `component`: one row per versionless purl, keyed by purl_id.
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/lodash', 'lodash', 'lodash', NULL, 'uuid-lodash');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/electron-debug', 'Sindre Sorhus', 'electron-debug', NULL, 'uuid-electron-debug');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/isbinaryfile', 'npmjs', 'isbinaryfile', NULL, 'uuid-isbinaryfile');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/sort-paths', 'npmjs', 'sort-paths', NULL, 'uuid-sort-paths');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:deb/debian/goffice', 'goffice', 'goffice', NULL, 'uuid-goffice');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/vue-phone', 'npmjs', 'vue-phone', NULL, 'uuid-vue-phone');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/%40scope/pkg', 'npmjs', '@scope/pkg', NULL, 'uuid-scoped');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:npm/dual', 'npmjs', 'dual', NULL, 'uuid-dual');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:maven/org.example/lib', 'org.example', 'lib', NULL, 'uuid-maven-lib');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:maven/org.example/core', 'org.example', 'core', NULL, 'uuid-maven-core');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:golang/go.uber.org/zap', 'go.uber.org', 'zap', NULL, 'uuid-zap');
INSERT INTO component (purl, vendor, component, source_purl_id, purl_id) VALUES ('pkg:github/gin-gonic/gin', 'gin-gonic', 'gin', NULL, 'uuid-gin');
