DROP TABLE IF EXISTS projects;
CREATE TABLE projects
(
    mine_id             integer not null,
    vendor              text    not null,
    component           text    not null,
    first_version_date  text,
    latest_version_date text,
    license             text,
    versions            integer,
    source_vendor       text,
    source_component    text,
    git_created_at      text,
    git_updated_at      text,
    git_pushed_at       text,
    git_watchers        integer,
    git_issues          integer,
    git_forks           integer,
    git_license         text,
    source_mine_id      integer,
    purl_name           text    not null,
    source_purl_name    text,
    verified            text,
    primary key (mine_id, purl_name)
);
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (1, 'taballa.hp-PD', 'tablestyle', '2013-07-05', '2013-08-26', 'MIT', 8, null, null, null, null, null, null, null, null, null, null, 'tablestyle', null, null);
