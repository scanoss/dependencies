DROP TABLE IF EXISTS licenses;
CREATE TABLE licenses
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    license_name text not null unique,
    spdx_id      text not null default '',
    is_spdx      boolean not null default false,
    is_sanitized boolean not null default false
);

insert into licenses (id, license_name, spdx_id, is_spdx) values (15, 'GPL-2 or GPL-3', 'GPL-2.0-only/GPL-3.0-only/DoesNotExist', true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (109, '3-Clause BSD License', 'BSD-3-Clause', true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (552, 'Apache 2.0', 'Apache-2.0', true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (850, 'Apache License 2.0', 'Apache-2.0', true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (1378, 'BSD', '0BSD', true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (2815,'GPL-2.0-only','GPL-2.0-only',true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (2821,'GPL-3.0-only','GPL-3.0-only',true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (4863, 'ISC', 'ISC', true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (5236, 'LGPLv2.1+', 'LGPL-2.1-or-later', true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (5614, 'MIT', 'MIT', true);
insert into licenses (id, license_name, spdx_id, is_spdx) values (9999, '', '', false);
