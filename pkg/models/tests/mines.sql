DROP TABLE IF EXISTS mines;
CREATE TABLE mines (
                       id    INTEGER PRIMARY KEY,
                       mine_name TEXT  DEFAULT '',
                       purl_type  TEXT  DEFAULT ''
);
INSERT INTO mines (id, mine_name, purl_type) VALUES (0, 'maven.org', 'maven');
INSERT INTO mines (id, mine_name, purl_type) VALUES (1, 'rubygems.org', 'gem');
INSERT INTO mines (id, mine_name, purl_type) VALUES (2, 'npmjs.org', 'npm');
INSERT INTO mines (id, mine_name, purl_type) VALUES (3, 'pythonhosted.org', 'pypi');
INSERT INTO mines (id, mine_name, purl_type) VALUES (4, 'gitee.com', 'gitee');
INSERT INTO mines (id, mine_name, purl_type) VALUES (5, 'github.com', 'github');
INSERT INTO mines (id, mine_name, purl_type) VALUES (6, 'stackoverflow.com', 'stackoverflow');
INSERT INTO mines (id, mine_name, purl_type) VALUES (7, 'fedoraproject.org', 'rpm');
INSERT INTO mines (id, mine_name, purl_type) VALUES (8, 'rpmfind.net', 'rpm');
INSERT INTO mines (id, mine_name, purl_type) VALUES (9, 'nuget.org', 'nuget');
INSERT INTO mines (id, mine_name, purl_type) VALUES (10, 'debian.org', 'deb');
INSERT INTO mines (id, mine_name, purl_type) VALUES (11, 'sourceforge.net', 'sourceforge');
INSERT INTO mines (id, mine_name, purl_type) VALUES (12, 'googlesource.com', 'googlesource');
INSERT INTO mines (id, mine_name, purl_type) VALUES (13, 'gnome.org', 'gnome');
INSERT INTO mines (id, mine_name, purl_type) VALUES (14, 'java2s.com', 'java2s');
INSERT INTO mines (id, mine_name, purl_type) VALUES (15, 'spring.io', 'maven');
INSERT INTO mines (id, mine_name, purl_type) VALUES (16, 'drupal.org', 'drupal');
INSERT INTO mines (id, mine_name, purl_type) VALUES (17, 'apache.org', 'apache');
INSERT INTO mines (id, mine_name, purl_type) VALUES (18, 'cpan.org', 'cpan');
INSERT INTO mines (id, mine_name, purl_type) VALUES (19, 'opensuse.org', 'rpm');
INSERT INTO mines (id, mine_name, purl_type) VALUES (20, 'kernel.org', 'kernel');
INSERT INTO mines (id, mine_name, purl_type) VALUES (21, 'launchpad.net', 'deb');
INSERT INTO mines (id, mine_name, purl_type) VALUES (22, 'angularjs.org', 'angular');
INSERT INTO mines (id, mine_name, purl_type) VALUES (23, 'nasm.us', 'nasm');
INSERT INTO mines (id, mine_name, purl_type) VALUES (24, 'videolan.org', 'videolan');
INSERT INTO mines (id, mine_name, purl_type) VALUES (25, 'gnu.org', 'gnu');
INSERT INTO mines (id, mine_name, purl_type) VALUES (26, 'zlib.net', 'zlib');
INSERT INTO mines (id, mine_name, purl_type) VALUES (27, 'nodejs.org', 'npm');
INSERT INTO mines (id, mine_name, purl_type) VALUES (28, 'centos.org', 'rpm');
INSERT INTO mines (id, mine_name, purl_type) VALUES (29, 'apple.com', 'apple');
INSERT INTO mines (id, mine_name, purl_type) VALUES (30, 'rpmfusion.org', 'rpm');
INSERT INTO mines (id, mine_name, purl_type) VALUES (31, 'isc.org', 'isc');
INSERT INTO mines (id, mine_name, purl_type) VALUES (32, 'nmap.org', 'nmap');
INSERT INTO mines (id, mine_name, purl_type) VALUES (33, 'postgresql', 'postgresql');
INSERT INTO mines (id, mine_name, purl_type) VALUES (34, 'mozilla.org', 'mozilla');
INSERT INTO mines (id, mine_name, purl_type) VALUES (35, 'jquery.com', 'jquery');
INSERT INTO mines (id, mine_name, purl_type) VALUES (36, 'sudo.ws', 'sudo');
INSERT INTO mines (id, mine_name, purl_type) VALUES (37, 'slf4j.org', 'slf4j');
INSERT INTO mines (id, mine_name, purl_type) VALUES (38, 'gnome.org', 'gitlab');
INSERT INTO mines (id, mine_name, purl_type) VALUES (39, 'gitlab.com', 'gitlab');
INSERT INTO mines (id, mine_name, purl_type) VALUES (40, 'bitbucket.org', 'bitbucket');
INSERT INTO mines (id, mine_name, purl_type) VALUES (41, 'wordpress.org', 'wordpress');
INSERT INTO mines (id, mine_name, purl_type) VALUES (45, 'pkg.go.dev', 'golang');
