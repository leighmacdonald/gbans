begin;

CREATE TABLE IF NOT EXISTS net_location
(
    net_location_id bigserial primary key,
    ip_from         inet         not null,
    ip_to           inet         not null,
    country_code    varchar(2)   not null default '',
    country_name    varchar(64)  not null default '',
    region_name     varchar(128) not null default '',
    city_name       varchar(128) not null default '',
    location        geometry     not null
);

CREATE INDEX IF NOT EXISTS net_location_ip_from_index on net_location (ip_from);
CREATE INDEX IF NOT EXISTS net_location_ip_to_index on net_location (ip_to);

CREATE TABLE IF NOT EXISTS net_proxy
(
    net_proxy_id bigserial primary key,
    ip_from      inet         not null,
    ip_to        inet         not null,
    proxy_type   varchar(3)   not null default '',
    country_code varchar(2)   not null default '',
    country_name varchar(64)  not null default '',
    region_name  varchar(128) not null default '',
    city_name    varchar(128) not null default '',
    isp          varchar(256) not null default '',
    domain_used  varchar(128) not null default '',
    usage_type   varchar(11)  not null default '',
    as_num       integer      not null default 0,
    as_name      varchar(256) not null default '',
    last_seen    timestamp   not null,
    threat       varchar(128) not null default ''
);

CREATE INDEX IF NOT EXISTS net_proxy_ip_from_index on net_proxy (ip_from);
CREATE INDEX IF NOT EXISTS net_proxy_ip_to_index on net_proxy (ip_to);

CREATE TABLE IF NOT EXISTS net_asn
(
    net_asn_id bigserial primary key,
    ip_from    inet         not null,
    ip_to      inet         not null,
    cidr       cidr         not null,
    as_num     integer      not null default 0,
    as_name    varchar(256) not null default ''
);

CREATE INDEX IF NOT EXISTS net_asn_ip_from_index on net_asn (ip_from);
CREATE INDEX IF NOT EXISTS net_asn_ip_to_index on net_asn (ip_to);
CREATE INDEX IF NOT EXISTS net_asn_cidr_to_index on net_asn (cidr);

commit;
