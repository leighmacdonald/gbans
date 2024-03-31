FROM postgres:15-bullseye

RUN apt-get update \
      && apt-cache showpkg postgresql-$PG_MAJOR-ip4r \
      && apt-get install -y --no-install-recommends \
           postgresql-$PG_MAJOR-ip4r \
      && rm -rf /var/lib/apt/lists/*