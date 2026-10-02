-- Restore the login permission grant on every non-banned role (the state
-- before the up migration removed it).
INSERT INTO role_permissions (role_id, permission, created_on, updated_on)
SELECT
  r.role_id,
  'PERMISSION_LOGIN'::permission,
  NOW(),
  NOW()
FROM
  roles AS r
WHERE
  r.role_name <> 'banned'
  AND NOT EXISTS (
    SELECT
      1
    FROM
      role_permissions AS rp
    WHERE
      rp.role_id = r.role_id
      AND rp.permission = 'PERMISSION_LOGIN'::permission
  )
ON CONFLICT DO NOTHING;
