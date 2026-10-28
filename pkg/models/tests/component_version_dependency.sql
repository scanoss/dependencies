-- Test data only. The schema comes from testSchemaDDL in pkg/models/test_schema.go; do not add DDL here.
-- Serving `component_version_dependency`: one row per declared dependency, every
-- scope. vue-phone@1.0.9 -> lodash -> (none); the scoped package depends on a
-- scoped one; the maven library on another maven artifact.
INSERT INTO component_version_dependency (purl_id, version, dep_purl, dep_version, dep_scope, dep_resolved_version) VALUES ('uuid-vue-phone', '1.0.9', 'pkg:npm/lodash', '^4.17.0', 'dep', '4.17.21');
INSERT INTO component_version_dependency (purl_id, version, dep_purl, dep_version, dep_scope, dep_resolved_version) VALUES ('uuid-vue-phone', '1.0.9', 'pkg:npm/%40types/node', '~18.0.0', 'dev', '18.0.6');
INSERT INTO component_version_dependency (purl_id, version, dep_purl, dep_version, dep_scope, dep_resolved_version) VALUES ('uuid-vue-phone', '1.0.9', 'pkg:npm/isbinaryfile', '', 'peer', NULL);
INSERT INTO component_version_dependency (purl_id, version, dep_purl, dep_version, dep_scope, dep_resolved_version) VALUES ('uuid-scoped', '1.0.0', 'pkg:npm/%40scope/other', '2.0.0', 'dep', '2.0.0');
INSERT INTO component_version_dependency (purl_id, version, dep_purl, dep_version, dep_scope, dep_resolved_version) VALUES ('uuid-maven-lib', '1.0.0', 'pkg:maven/org.example/core', '2.0.0', 'compile', '2.0.0');
INSERT INTO component_version_dependency (purl_id, version, dep_purl, dep_version, dep_scope, dep_resolved_version) VALUES ('uuid-gin', 'v1.9.1', 'pkg:golang/github.com/json-iterator/go', 'v1.1.12', 'dep', 'v1.1.12');
