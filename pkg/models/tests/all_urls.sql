DROP TABLE IF EXISTS all_urls;
CREATE TABLE all_urls
(
    package_hash text not null,
    vendor       text,
    component    text,
    version      text,
    date         text,
    url          text not null,
    url_hash     text not null,
    mine_id      integer,
    license      text,
    purl_name    text,
    primary key (package_hash, url, url_hash)
);
INSERT INTO all_urls (package_hash, vendor, component, version, date, url, url_hash, mine_id, license, purl_name) VALUES ('33523708f7fbef1c554d340a5cfd42aa', 'taballa.hp-PD', 'tablestyle', '0.0.8', '2013-08-26', 'https://rubygems.org/downloads/tablestyle-0.0.8.gem', '5e2c89ddef74a0873169f2e13f5efba6', 1, 'MIT', 'tablestyle');
INSERT INTO all_urls (package_hash, vendor, component, version, date, url, url_hash, mine_id, license, purl_name) VALUES ('4986c6fed8d0b30daa9caaa0e94101ff', 'taballa.hp-PD', 'tablestyle', '0.0.9', '2013-08-26', 'https://rubygems.org/downloads/tablestyle-0.0.9.gem', 'bd2af9a9445fdd5bfe1ab28dd26f9c42', 1, 'MIT', 'tablestyle');
INSERT INTO all_urls (package_hash, vendor, component, version, date, url, url_hash, mine_id, license, purl_name) VALUES ('4d66775f503b1e76582e7e5b2ea54d92', 'taballa.hp-PD', 'tablestyle', '0.0.10', '2013-08-26', 'https://rubygems.org/downloads/tablestyle-0.0.10.gem', '5a088240b44efa142be4b3c40f8ae9c1', 1, 'MIT', 'tablestyle');
INSERT INTO all_urls (package_hash, vendor, component, version, date, url, url_hash, mine_id, license, purl_name) VALUES ('85ace696e6aec401c611eac8da439d75', 'taballa.hp-PD', 'tablestyle', '0.0.5', '2013-07-08', 'https://rubygems.org/downloads/tablestyle-0.0.5.gem', '2e60ef936150975ca2071cc8782c062a', 1, 'MIT', 'tablestyle');
INSERT INTO all_urls (package_hash, vendor, component, version, date, url, url_hash, mine_id, license, purl_name) VALUES ('952edab5837f5f0e59638dd791187725', 'taballa.hp-PD', 'tablestyle', '0.0.11', '2013-08-26', 'https://rubygems.org/downloads/tablestyle-0.0.11.gem', '59fc4cf45a7d1425303a5bb897a463f4', 1, 'MIT', 'tablestyle');
INSERT INTO all_urls (package_hash, vendor, component, version, date, url, url_hash, mine_id, license, purl_name) VALUES ('b1cd1444c2f76e7564f57b0e047994a4', 'taballa.hp-PD', 'tablestyle', '0.0.4', '2013-07-08', 'https://rubygems.org/downloads/tablestyle-0.0.4.gem', 'b494b3e367d26b6ab2785ad3aee8afb7', 1, 'MIT', 'tablestyle');
INSERT INTO all_urls (package_hash, vendor, component, version, date, url, url_hash, mine_id, license, purl_name) VALUES ('bfada11fd2b2b8fa23943b8b6fe5cb3f', 'taballa.hp-PD', 'tablestyle', '0.0.12', '2013-08-26', 'https://rubygems.org/downloads/tablestyle-0.0.12.gem', '686dc352775b58652c9d9ddb2117f402', 1, 'MIT', 'tablestyle');
INSERT INTO all_urls (package_hash, vendor, component, version, date, url, url_hash, mine_id, license, purl_name) VALUES ('f586d603a9cb2460c4517cffad6ad5e4', 'taballa.hp-PD', 'tablestyle', '0.0.7', null, 'https://rubygems.org/downloads/tablestyle-0.0.7.gem', '2a3251711e7010ca15d232ec4ec4fb16', 1, 'MIT', 'tablestyle');
INSERT INTO all_urls (package_hash, vendor, component, version, date, url, url_hash, mine_id, license, purl_name) VALUES ('7282b2348ff82296a9f84d399fd2799d', 'taballa.hp-PD', 'tablestyle', '0.0.1', '2013-07-05', 'https://rubygems.org/downloads/tablestyle-0.0.1.gem', 'ce63a5ed13cc0446f4b61cb779c9d4e0', 1, '', 'tablestyle');
