DROP TABLE IF EXISTS golang_projects;
CREATE TABLE golang_projects
(
    component                   text    not null,
    version                     text    not null,
    version_id                  integer not null,
    version_date                text    not null,
    is_module                   boolean,
    is_package                  boolean,
    license                     text    not null,
    license_id                  integer not null,
    has_valid_go_mod_file       boolean,
    has_redistributable_license boolean,
    has_tagged_version          boolean,
    has_stable_version          boolean,
    repository                  text    not null,
    is_indexed                  boolean,
    purl_name                   text    not null,
    mine_id                     integer not null,
    index_timestamp             text    not null,
    primary key (purl_name, version)
);

INSERT INTO golang_projects (component, version, version_id, version_date, license, license_id, repository, purl_name, mine_id, index_timestamp, is_indexed) VALUES ('github.com/scanoss/papi', 'v0.0.1', 5958021, '', 'MIT', 5614, 'github.com/scanoss/papi', 'github.com/scanoss/papi', 45, '2022-02-21T19:51:21.112979Z', True);
INSERT INTO golang_projects (component, version, version_id, version_date, license, license_id, repository, purl_name, mine_id, index_timestamp, is_indexed) VALUES ('google.golang.org/grpc', 'v1.19.0', 5193086, '', 'Apache-2.0', 552, 'github.com/grpc/grpc-go', 'google.golang.org/grpc', 45, '2022-05-09T20:17:02.339878Z', True);
INSERT INTO golang_projects (component, version, version_id, version_date, license, license_id, repository, purl_name, mine_id, index_timestamp, is_indexed) VALUES ('google.golang.org/grpc', 'v1.7.0', 11640350, '', '', 9999, 'github.com/grpc/grpc-go', 'google.golang.org/grpc', 45, '2023-11-24T20:17:02.339878Z', True);
