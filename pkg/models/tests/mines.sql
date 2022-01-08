DROP TABLE IF EXISTS mines;
CREATE TABLE mines (
                       id    INTEGER PRIMARY KEY,
                       name TEXT  DEFAULT '',
                       purl_type  TEXT  DEFAULT ''
);
INSERT INTO mines (id, name, purl_type) VALUES (0,  'maven.org',    'maven');
INSERT INTO mines (id, name, purl_type) VALUES (1,  'rubygems.org', 'gem');
INSERT INTO mines (id, name, purl_type) VALUES (2,  'npmjs.org',    'npm');
-- INSERT INTO mines (id, name, purl_type) VALUES (27, 'nodejs.org',   'npm');
