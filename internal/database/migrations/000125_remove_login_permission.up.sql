-- Login no longer requires a specific permission: any authenticated user can
-- log in. Remove every login grant from the role tables.
--
-- The PERMISSION_LOGIN value cannot be dropped from the `permission` enum
-- type: PostgreSQL does not implement `ALTER TYPE ... DROP VALUE`. The value
-- is left in place but is now unused (it can no longer be granted via the API
-- and is checked by no route).
DELETE FROM role_permissions
WHERE permission = 'PERMISSION_LOGIN';
