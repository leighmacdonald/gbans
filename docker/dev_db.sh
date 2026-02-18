docker volume create gbans-db-data
docker build -t gbans-db:latest -f postgres-ip4r.Dockerfile .
docker stop gbans-db || true
docker rm gbans-db || true
docker run -t \
    --name=gbans-db \
    --restart unless-stopped \
    -p 0.0.0.0:5432:5432 \
    -v gbans-db-data:/var/lib/postgresql/data \
    -e POSTGRES_USER=gbans \
    -e POSTGRES_PASSWORD=gbans \
    -e POSTGRES_DB=gbans \
    -e POSTGRES_HOST_AUTH_METHOD=md5 \
    -e POSTGRES_INITDB_ARGS=--auth-host=md5 \
    gbans-db:latest
