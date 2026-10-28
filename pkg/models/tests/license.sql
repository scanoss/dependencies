-- Test data only. The schema comes from testSchemaDDL in pkg/models/test_schema.go; do not add DDL here.
-- Serving `license` (the catalogue) and "SPDX_license_data" (the SPDX registry).
INSERT INTO license (id, long_name, SPDX, content) VALUES (1, 'MIT', 'MIT', NULL);
INSERT INTO license (id, long_name, SPDX, content) VALUES (2, 'Apache 2.0', 'Apache-2.0', NULL);
INSERT INTO license (id, long_name, SPDX, content) VALUES (15, 'GPL-2 or GPL-3', 'GPL-2.0-only/GPL-3.0-only', NULL);
INSERT INTO license (id, long_name, SPDX, content) VALUES (20, 'GPL-2.0-only', 'GPL-2.0-only', NULL);
INSERT INTO license (id, long_name, SPDX, content) VALUES (21, 'GPL-3.0-only', 'GPL-3.0-only', NULL);
INSERT INTO license (id, long_name, SPDX, content) VALUES (30, 'Custom Licence', 'LicenseRef-custom', NULL);
INSERT INTO SPDX_license_data (SPDX_id, name, is_osi_approved) VALUES ('MIT', 'MIT License', 1);
INSERT INTO SPDX_license_data (SPDX_id, name, is_osi_approved) VALUES ('Apache-2.0', 'Apache License 2.0', 1);
INSERT INTO SPDX_license_data (SPDX_id, name, is_osi_approved) VALUES ('GPL-2.0-only', 'GNU General Public License v2.0 only', 1);
INSERT INTO SPDX_license_data (SPDX_id, name, is_osi_approved) VALUES ('GPL-3.0-only', 'GNU General Public License v3.0 only', 1);
