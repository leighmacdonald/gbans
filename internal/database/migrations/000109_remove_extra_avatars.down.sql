BEGIN;

ALTER TABLE person
ADD COLUMN IF NOT EXISTS avatar text not null DEFAULT '';

ALTER TABLE person
ADD COLUMN IF NOT EXISTS avatarmedium text not null DEFAULT '';

ALTER TABLE person
ADD COLUMN IF NOT EXISTS avatarfull text not null DEFAULT '';

ALTER TABLE person
ADD COLUMN IF NOT EXISTS profileurl text not null DEFAULT '';

COMMIT;
