ALTER TABLE wiki
ADD CONSTRAINT wiki_slug_not_empty CHECK (slug != ''),
ADD CONSTRAINT wiki_body_not_empty CHECK (body_md != '');
