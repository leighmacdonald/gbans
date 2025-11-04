ALTER TABLE wiki
DROP CONSTRAINT IF EXISTS wiki_slug_not_empty;

ALTER TABLE wiki
DROP CONSTRAINT IF EXISTS wiki_body_not_empty;
