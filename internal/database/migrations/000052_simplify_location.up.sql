begin;
alter table server drop column location;

alter table server add latitude float4 default 0.0 not null;

alter table server add longitude float4 default 0.0 not null;

commit;
