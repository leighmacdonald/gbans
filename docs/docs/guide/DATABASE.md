# Database Reference

## Creating a sourcemod user

It's recommended to create a secondary non-privileged user, especially when using servers remote of the
gbans instance. Below is an example of creating a read-only user that only has access to the tables, and functions, required
for operation.

```postgresql
CREATE ROLE sourcemod WITH LOGIN PASSWORD '<new-password>';
GRANT CONNECT ON DATABASE gbans TO sourcemod;
GRANT USAGE ON SCHEMA public TO sourcemod ;
GRANT SELECT ON 
    sm_config, sm_overrides, sm_group_overrides, sm_group_immunity, sm_groups,
    sm_admins_groups, sm_adminsTO sourcemod;
GRANT SELECT, INSERT, UPDATE, DELETE ON sm_cookie_cache, sm_cookies TO sourcemod;
GRANT EXECUTE ON FUNCTION check_ban TO sourcemod;

```
