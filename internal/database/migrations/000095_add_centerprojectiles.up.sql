BEGIN;

CREATE TABLE IF NOT EXISTS sm_cookies
(
    id serial,
    name varchar(30) NOT NULL UNIQUE,
    description varchar(255),
    access INTEGER,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS sm_cookie_cache
(
    player varchar(65) NOT NULL,
    cookie_id int NOT NULL,
    value varchar(100),
    timestamp int NOT NULL,
    PRIMARY KEY (player, cookie_id)
);


COMMIT;
