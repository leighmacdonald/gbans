FROM postgis/postgis:18-3.6

RUN apt-get update \
    && apt-cache showpkg postgresql-$PG_MAJOR-ip4r \
    && apt-cache showpkg postgresql-$PG_MAJOR-hypopg \
    && apt-get install -y --no-install-recommends \
    postgresql-$PG_MAJOR-ip4r postgresql-$PG_MAJOR-hypopg \
    && rm -rf /var/lib/apt/lists/*
