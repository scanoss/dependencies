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
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Alexey Prokhorov', 'electron-log', '2016-05-02', '2021-07-31', 'MIT', 98, 'megahertz', 'electron-log', '2016-05-02', '2021-12-29', '2021-12-27', 895, 9, 117, 'MIT', 5, 'electron-log', 'megahertz/electron-log', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Jake Zatecky', 'react-checkbox-tree', '2016-02-04', '2021-08-09', 'MIT', 47, 'jakezatecky', 'react-checkbox-tree', '2016-02-04', '2021-12-30', '2021-12-20', 550, 80, 165, 'MIT', 5, 'react-checkbox-tree', 'jakezatecky/react-checkbox-tree', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'React Training', 'history', '2012-04-29', '2021-11-02', 'MIT', 98, 'ReactTraining', 'history', '2015-07-18', '2021-08-12', '2021-08-12', 7099, 115, 841, 'MIT', 5, 'history', 'reacttraining/history', '2021-08-12');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Nikhil Marathe', 'uuid', '2011-03-31', '2021-11-29', 'MIT', 34, 'uuidjs', 'uuid', '2010-12-28', '2022-01-02', '2021-12-07', 11844, 19, 792, 'MIT', 5, 'uuid', 'uuidjs/uuid', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Jeff Barczewski', 'react', '2011-10-26', '2021-12-06', 'MIT', 716, 'facebook', 'react', '2013-05-24', '2022-01-02', '2021-12-31', 180033, 941, 36560, 'MIT', 5, 'react', 'facebook/react', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'React Training', 'react-router-dom', '2016-12-14', '2021-11-09', 'MIT', 60, 'ReactTraining', 'react-router', '2014-05-16', '2021-08-12', '2021-08-11', 43727, 59, 8505, 'MIT', 5, 'react-router-dom', 'reacttraining/react-router', '2021-08-12');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Felix Geisendörfer', 'form-data', '2011-05-16', '2021-02-15', 'MIT', 38, 'form-data', 'form-data', '2011-05-16', '2021-12-31', '2021-11-30', 1959, 111, 258, 'MIT', 5, 'form-data', 'form-data/form-data', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Toru Nagashima', 'abort-controller', '2017-09-29', '2019-03-30', 'MIT', 11, 'mysticatea', 'abort-controller', '2017-09-29', '2021-12-30', '2021-03-30', 258, 17, 25, 'MIT', 5, 'abort-controller', 'mysticatea/abort-controller', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'tomkp', 'react-split-pane', '2015-06-14', '2020-08-10', 'MIT', 89, 'tomkp', 'react-split-pane', '2015-03-06', '2022-01-02', '2021-11-26', 2676, 142, 344, 'MIT', 5, 'react-split-pane', 'tomkp/react-split-pane', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Sindre Sorhus', 'electron-debug', '2015-06-01', '2020-12-21', 'MIT', 28, 'sindresorhus', 'electron-debug', '2015-06-01', '2021-12-11', '2021-01-23', 693, 11, 55, 'MIT', 5, 'electron-debug', 'sindresorhus/electron-debug', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Shane Harris', 'node-blob', '2019-03-21', '2019-03-21', 'MIT', 2, 'shaneharris', 'node-blob', null, null, null, null, null, null, null, 5, 'node-blob', 'shaneharris/node-blob', '2021-05-24');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Gergely Hornich', 'sort-paths', '2016-09-03', '2018-08-18', 'MIT', 3, 'ghornich', 'sort-paths', null, null, null, null, null, null, null, 5, 'sort-paths', 'ghornich/sort-paths', '2021-05-24');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'David Frank', 'node-fetch', '2015-01-27', '2021-11-08', 'MIT', 69, 'bitinn', 'node-fetch', null, null, null, null, null, null, null, 5, 'node-fetch', 'bitinn/node-fetch', '2021-05-24');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'npmjs', 'chart.js', '2014-07-08', '2021-12-05', 'MIT', 76, 'chartjs', 'Chart.js', '2013-03-17', '2022-01-02', '2022-01-02', 55758, 108, 11323, 'MIT', 5, 'chart.js', 'chartjs/chart.js', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Sindre Sorhus', 'p-queue', '2016-10-28', '2021-04-07', 'MIT', 34, 'sindresorhus', 'p-queue', '2016-10-28', '2022-01-01', '2021-12-15', 1875, 31, 135, 'MIT', 5, 'p-queue', 'sindresorhus/p-queue', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Ben Newman', 'regenerator-runtime', '2014-07-04', '2021-07-22', 'MIT', 27, 'facebook', 'regenerator', '2013-10-05', '2021-12-31', '2021-12-31', 3619, 77, 1186, 'MIT', 5, 'regenerator-runtime', 'facebook/regenerator', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Conor Hastings', 'react-syntax-highlighter', '2016-01-27', '2021-11-12', 'MIT', 121, 'react-syntax-highlighter', 'react-syntax-highlighter', '2016-01-27', '2022-01-02', '2021-11-15', 2435, 75, 195, 'MIT', 5, 'react-syntax-highlighter', 'react-syntax-highlighter/react-syntax-highlighter', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'npmjs', 'source-map-support', '2013-01-18', '2021-11-19', 'MIT', 66, 'evanw', 'node-source-map-support', '2013-01-18', '2022-01-02', '2021-12-24', 1950, 101, 226, 'MIT', 5, 'source-map-support', 'evanw/node-source-map-support', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Etienne Lemay', 'react-dom', '2014-05-06', '2021-12-06', 'MIT', 671, 'facebook', 'react', '2013-05-24', '2022-01-02', '2021-12-31', 180033, 941, 36560, 'MIT', 5, 'react-dom', 'facebook/react', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'npmjs', 'isbinaryfile', '2012-10-09', '2021-04-29', 'MIT', 30, 'gjtorikian', 'isBinaryFile', '2012-10-08', '2021-11-24', '2021-09-24', 141, 3, 20, 'MIT', 5, 'isbinaryfile', 'gjtorikian/isbinaryfile', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (2, 'Vladimir Krivosheev', 'electron-updater', '2015-05-01', '2021-12-01', 'MIT', 224, 'electron-userland', 'electron-builder', '2015-05-21', '2022-01-02', '2022-01-02', 11688, 301, 1470, 'MIT', 5, 'electron-updater', 'electron-userland/electron-builder', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (3, 'Giorgos Verigakis', 'progress', '2012-04-18', '2021-07-28', 'ISC', 9, 'verigak', 'progress', '2012-04-18', '2022-01-02', '2021-11-15', 1117, 28, 161, null, 5, 'progress', 'verigak/progress', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (3, 'Kenneth Reitz', 'requests', '2011-02-14', '2021-07-13', 'Apache 2.0', 143, 'psf', 'requests', '2011-02-13', '2022-01-02', '2021-12-29', 46620, 222, 8581, 'Apache-2.0', 5, 'requests', 'psf/requests', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (3, 'pypi', 'protobuf', '2008-07-10', '2021-10-29', '3-Clause BSD License', 92, 'protocolbuffers', 'protobuf', '2014-08-26', '2022-01-02', '2022-01-01', 52433, 1008, 13591, null, 5, 'protobuf', 'protocolbuffers/protobuf', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (3, 'The ICRAR DIA Team', 'crc32c', '2017-06-07', '2021-06-25', 'LGPLv2.1+', 13, 'ICRAR', 'crc32c', '2017-06-07', '2021-10-15', '2021-06-25', 26, 0, 15, null, 5, 'crc32c', 'icrar/crc32c', '2022-01-02');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (3, 'Audrey Roy Greenfeld', 'binaryornot', '2013-08-17', '2017-08-03', 'BSD', 8, 'audreyr', 'binaryornot', null, null, null, null, null, null, null, 5, 'binaryornot', 'audreyr/binaryornot', '2021-05-24');
INSERT INTO projects (mine_id, vendor, component, first_version_date, latest_version_date, license, versions, source_vendor, source_component, git_created_at, git_updated_at, git_pushed_at, git_watchers, git_issues, git_forks, git_license, source_mine_id, purl_name, source_purl_name, verified) VALUES (3, 'The gRPC Authors', 'grpcio', '2015-03-30', '2021-11-17', 'Apache License 2.0', 163, null, null, null, null, null, null, null, null, null, null, 'grpcio', null, null);
