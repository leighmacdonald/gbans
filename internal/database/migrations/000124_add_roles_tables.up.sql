do
$do$
BEGIN
    CREATE TYPE permission AS ENUM (
        'PERMISSION_ANTICHEAT_READ',
        'PERMISSION_APPEAL_READ',
        'PERMISSION_APPEAL_WRITE',
        'PERMISSION_ASSET_WRITE',
        'PERMISSION_ASSET_DELETE',
        'PERMISSION_BAN_READ',
        'PERMISSION_BAN_WRITE',
        'PERMISSION_BAN_CREATE',
        'PERMISSION_BLOCKLIST_READ',
        'PERMISSION_BLOCKLIST_WRITE',
        'PERMISSION_BLOCKLIST_DELETE',
        'PERMISSION_CONFIG_READ',
        'PERMISSION_CONFIG_WRITE',
        'PERMISSION_CONTEST_READ',
        'PERMISSION_CONTEST_WRITE',
        'PERMISSION_CONTEST_DELETE',
        'PERMISSION_FORUM_READ',
        'PERMISSION_FORUM_WRITE',
        'PERMISSION_FORUM_EDIT',
        'PERMISSION_GAMEADMIN_READ',
        'PERMISSION_GAMEADMIN_WRITE',
        'PERMISSION_NETWORK_READ',
        'PERMISSION_NETWORK_ADMIN',
        'PERMISSION_NEWS_READ',
        'PERMISSION_NEWS_WRITE',
        'PERMISSION_NEWS_DELETE',
        'PERMISSION_PERSON_READ',
        'PERMISSION_PERSON_WRITE',
        'PERMISSION_REPORT_READ',
        'PERMISSION_REPORT_WRITE',
        'PERMISSION_REPORT_CREATE',
        'PERMISSION_SERVER_READ',
        'PERMISSION_SERVER_WRITE',
        'PERMISSION_SPEEDRUN_READ',
        'PERMISSION_SPEEDRUN_WRITE',
        'PERMISSION_STATS_READ',
        'PERMISSION_WIKI_READ',
        'PERMISSION_WIKI_EDIT',
        'PERMISSION_VOTE_READ',
        'PERMISSION_WORDFILTER_READ',
        'PERMISSION_WORDFILTER_WRITE',
        'PERMISSION_WORDFILTER_DELETE'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END
$do$;

CREATE TABLE IF NOT EXISTS roles (
    role_id SERIAL PRIMARY KEY,
    role_name TEXT NOT NULL,
    created_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_on TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

create unique index if not exists "roles_role_name_uidx" on roles using btree (role_name);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INTEGER NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
    permission permission NOT NULL,
    created_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission)
);

CREATE TABLE IF NOT EXISTS role_assignments (
    steam_id BIGINT NOT NULL REFERENCES person(steam_id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
    created_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (steam_id, role_id)
);

insert into roles (role_name) values ('admin') on conflict do nothing;
insert into roles (role_name) values ('moderator') on conflict do nothing;
insert into roles (role_name) values ('steamer') on conflict do nothing;
insert into roles (role_name) values ('banned') on conflict do nothing;
insert into roles (role_name) values ('user') on conflict do nothing;


INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'admin') r
CROSS JOIN (VALUES
    ('PERMISSION_ANTICHEAT_READ'::permission),
    ('PERMISSION_APPEAL_READ'::permission),
    ('PERMISSION_APPEAL_WRITE'::permission),
    ('PERMISSION_ASSET_WRITE'::permission),
    ('PERMISSION_ASSET_DELETE'::permission),
    ('PERMISSION_BAN_READ'::permission),
    ('PERMISSION_BAN_WRITE'::permission),
    ('PERMISSION_BAN_CREATE'::permission),
    ('PERMISSION_BLOCKLIST_READ'::permission),
    ('PERMISSION_BLOCKLIST_WRITE'::permission),
    ('PERMISSION_BLOCKLIST_DELETE'::permission),
    ('PERMISSION_CONFIG_READ'::permission),
    ('PERMISSION_CONFIG_WRITE'::permission),
    ('PERMISSION_CONTEST_READ'::permission),
    ('PERMISSION_CONTEST_WRITE'::permission),
    ('PERMISSION_CONTEST_DELETE'::permission),
    ('PERMISSION_FORUM_READ'::permission),
    ('PERMISSION_FORUM_WRITE'::permission),
    ('PERMISSION_FORUM_EDIT'::permission),
    ('PERMISSION_GAMEADMIN_READ'::permission),
    ('PERMISSION_GAMEADMIN_WRITE'::permission),
    ('PERMISSION_NETWORK_READ'::permission),
    ('PERMISSION_NETWORK_ADMIN'::permission),
    ('PERMISSION_NEWS_READ'::permission),
    ('PERMISSION_NEWS_WRITE'::permission),
    ('PERMISSION_NEWS_DELETE'::permission),
    ('PERMISSION_PERSON_READ'::permission),
    ('PERMISSION_PERSON_WRITE'::permission),
    ('PERMISSION_REPORT_READ'::permission),
    ('PERMISSION_REPORT_WRITE'::permission),
    ('PERMISSION_REPORT_CREATE'::permission),
    ('PERMISSION_SERVER_READ'::permission),
    ('PERMISSION_SERVER_WRITE'::permission),
    ('PERMISSION_SPEEDRUN_READ'::permission),
    ('PERMISSION_SPEEDRUN_WRITE'::permission),
    ('PERMISSION_STATS_READ'::permission),
    ('PERMISSION_WIKI_READ'::permission),
    ('PERMISSION_WIKI_EDIT'::permission),
    ('PERMISSION_VOTE_READ'::permission),
    ('PERMISSION_WORDFILTER_READ'::permission),
    ('PERMISSION_WORDFILTER_WRITE'::permission),
    ('PERMISSION_WORDFILTER_DELETE'::permission)
) v(permission);

INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'moderator') r
CROSS JOIN (VALUES
    ('PERMISSION_ANTICHEAT_READ'::permission),
    ('PERMISSION_APPEAL_READ'::permission),
    ('PERMISSION_APPEAL_WRITE'::permission),
    ('PERMISSION_BAN_READ'::permission),
    ('PERMISSION_BAN_WRITE'::permission),
    ('PERMISSION_BLOCKLIST_READ'::permission),
    ('PERMISSION_CONTEST_WRITE'::permission),
    ('PERMISSION_CONTEST_DELETE'::permission),
    ('PERMISSION_FORUM_EDIT'::permission),
    ('PERMISSION_GAMEADMIN_READ'::permission),
    ('PERMISSION_GAMEADMIN_WRITE'::permission),
    ('PERMISSION_NETWORK_READ'::permission),
    ('PERMISSION_NETWORK_ADMIN'::permission),
    ('PERMISSION_NEWS_WRITE'::permission),
    ('PERMISSION_NEWS_DELETE'::permission),
    ('PERMISSION_PERSON_READ'::permission),
    ('PERMISSION_REPORT_READ'::permission),
    ('PERMISSION_REPORT_WRITE'::permission),
    ('PERMISSION_SPEEDRUN_WRITE'::permission),
    ('PERMISSION_WIKI_EDIT'::permission),
    ('PERMISSION_VOTE_READ'::permission),
    ('PERMISSION_WORDFILTER_READ'::permission),
    ('PERMISSION_WORDFILTER_WRITE'::permission),
    ('PERMISSION_WORDFILTER_DELETE'::permission)
) v(permission);

INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'streamer') r
CROSS JOIN (VALUES
    ('PERMISSION_BAN_WRITE'::permission)
) v(permission);

INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT r.role_id, v.permission, NOW(), NOW()
FROM (SELECT role_id FROM roles WHERE role_name = 'user') r
CROSS JOIN (VALUES
    ('PERMISSION_ASSET_WRITE'::permission),
    ('PERMISSION_ASSET_DELETE'::permission),
    ('PERMISSION_BAN_CREATE'::permission),
    ('PERMISSION_CONTEST_READ'::permission),
    ('PERMISSION_CONTEST_WRITE'::permission),
    ('PERMISSION_FORUM_READ'::permission),
    ('PERMISSION_FORUM_WRITE'::permission),
    ('PERMISSION_NEWS_READ'::permission),
    ('PERMISSION_REPORT_CREATE'::permission),
    ('PERMISSION_SPEEDRUN_READ'::permission),
    ('PERMISSION_STATS_READ'::permission),
    ('PERMISSION_WIKI_READ'::permission)
) v(permission);
