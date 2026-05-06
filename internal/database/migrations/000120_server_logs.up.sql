CREATE TABLE IF NOT EXISTS server_logs (
  server_log_id bigserial PRIMARY KEY,
  server_id INT NOT NULL REFERENCES server (server_id) ON DELETE CASCADE ON UPDATE CASCADE,
  body TEXT NOT NULL,
  created_on TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_server_logs_server_id ON server_logs (server_id);

CREATE INDEX IF NOT EXISTS idx_server_logs_created_on ON server_logs (created_on);
