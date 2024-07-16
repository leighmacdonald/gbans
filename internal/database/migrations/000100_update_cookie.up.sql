BEGIN;

CREATE OR REPLACE FUNCTION add_or_update_cookie(in_player VARCHAR(65), in_cookie INT, in_value VARCHAR(100), in_time INT) RETURNS VOID AS
$$
BEGIN
    LOOP
        -- first try to update the it.
        UPDATE sm_cookie_cache SET value = in_value, timestamp = in_time WHERE player = in_player AND cookie_id = in_cookie;
        IF found THEN
            RETURN;
        END IF;
        -- not there, so try to insert.
        -- if someone else inserts the same key concurrently, we could get a unique-key failure.
        BEGIN
            INSERT INTO sm_cookie_cache (player, cookie_id, value, timestamp) VALUES (in_player, in_cookie, in_value, in_time);
            RETURN;
        EXCEPTION WHEN unique_violation THEN
        -- do nothing...  loop again, and we'll update.
        END;
    END LOOP;
END;
$$
    LANGUAGE plpgsql;

COMMIT;
