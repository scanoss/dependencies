DROP TABLE IF EXISTS licenses;
CREATE TABLE licenses
(
    id           integer not null unique,
    license_name text not null primary key,
    alias_name   text,
    spdx_id      text not null default '',
    is_spdx      boolean not null default false
);

insert into licenses (id, license_name, alias_name, spdx_id, is_spdx) values (109, '3-Clause BSD License', null, 'BSD-3-Clause', true);
insert into licenses (id, license_name, alias_name, spdx_id, is_spdx) values (552, 'Apache 2.0', null, 'Apache-2.0', true);
insert into licenses (id, license_name, alias_name, spdx_id, is_spdx) values (850, 'Apache License 2.0', null, 'Apache-2.0', true);
insert into licenses (id, license_name, alias_name, spdx_id, is_spdx) values (1378, 'BSD', null, '0BSD', true);
insert into licenses (id, license_name, alias_name, spdx_id, is_spdx) values (4863, 'ISC', null, 'ISC', true);
insert into licenses (id, license_name, alias_name, spdx_id, is_spdx) values (5236, 'LGPLv2.1+', null, 'LGPL-2.1-or-later', true);
insert into licenses (id, license_name, alias_name, spdx_id, is_spdx) values (5614, 'MIT', null, 'MIT', true);
insert into licenses (id, license_name, alias_name, spdx_id, is_spdx) values (9999, '', null, '', false);
