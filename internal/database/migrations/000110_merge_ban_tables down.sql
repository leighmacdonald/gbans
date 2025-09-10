create table ban_net (
  net_id bigint primary key not null default nextval('ban_net_net_id_seq'::regclass),
  cidr ip4r not null,
  origin integer not null default 0,
  created_on timestamp with time zone not null,
  updated_on timestamp with time zone not null,
  reason_text text not null default ''::text,
  valid_until timestamp with time zone not null,
  deleted boolean not null default false,
  reason integer not null default 1,
  note text not null default ''::text,
  unban_reason_text text not null default ''::text,
  is_enabled boolean not null default true,
  source_id bigint not null default 0,
  target_id bigint not null default 0,
  appeal_state integer default 0
);

create unique index ban_net_cidr_uindex on ban_net using btree (cidr);

create table ban_group (
  ban_group_id bigint primary key not null default nextval('ban_group_ban_group_id_seq'::regclass),
  source_id bigint not null,
  target_id bigint not null default 0,
  group_id bigint not null,
  group_name text not null default ''::text,
  is_enabled boolean not null default true,
  deleted boolean not null default false,
  note text not null default ''::text,
  unban_reason_text text not null default ''::text,
  origin integer not null default 0,
  created_on timestamp with time zone not null,
  updated_on timestamp with time zone not null,
  valid_until timestamp with time zone not null,
  appeal_state integer default 0
);

create table ban_asn (
  ban_asn_id integer primary key not null default nextval('ban_asn_ban_asn_id_seq'::regclass),
  as_num bigint not null,
  origin integer not null default 0,
  source_id bigint not null,
  target_id bigint not null default 0,
  reason_text character varying,
  valid_until timestamp with time zone not null,
  created_on timestamp with time zone not null,
  updated_on timestamp with time zone not null,
  deleted boolean not null default false,
  reason integer not null default 1,
  unban_reason_text text not null default ''::text,
  note text not null default ''::text,
  is_enabled boolean not null default true,
  appeal_state integer default 0,
  foreign key (source_id) references public.person (steam_id) match simple on update no action on delete no action
);

create unique index ban_asn_as_num_uindex on ban_asn using btree (as_num);

ALTER TABLE IF EXISTS ban
DROP COLUMN IF EXISTS asn_num;

ALTER TABLE IF EXISTS ban
DROP COLUMN IF EXISTS cidr;

ALTER TABLE IF EXISTS ban
DROP COLUMN IF EXISTS name;
