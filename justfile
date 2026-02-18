all: frontend sourcemod buildp

fmt:
    golangci-lint fmt
    just -f frontend/justfile fmt

bump_deps:
    go get -u ./...
    just -f frontend/justfile update

buildp: frontend
    goreleaser release --clean

builds: frontend
    goreleaser release --clean --snapshot

watch:
    just -f frontend/justfile watch

generate:
    go generate ./...

serve:
    just -f frontend/justfile serve

frontend:
    just -f frontend/justfile

run:
    go run -race main.go

sourcemod:
    just -f sourcemod/justfile sourcemod

sourcemod_devel: sourcemod
    docker cp sourcemod/plugins/gbans.smx srcds-localhost-1:/home/tf2server/tf-dedicated/tf/addons/sourcemod/plugins/
    docker restart srcds-localhost-1

test: test-go test-ts

test-ts:
    just -f frontend/justfile test

test-go:
    go test -race ./...

check: lint_golangci vulncheck lint_ts typecheck_ts

vulncheck:
    govulncheck

lint_nil:
    nilaway -include-pkgs="github.com/leighmacdonald/gbans" -exclude-pkgs="github.com/jackc/pgx/v5" -test=false ./...

lint_golangci:
    golangci-lint run --timeout 3m ./...

fix: fmt
    golangci-lint run --fix

lint_ts:
    just -f frontend/justfile lint

typecheck_ts:
    just -f frontend/justfile typecheck

clean:
    go clean $(GO_FLAGS) -i
    rm -rf ./build/
    just -f frontend/justfile clean
    rm -rf ./sourcemod/plugins/gbans.smx

docker_test:
    docker compose -f docker/docker-compose-test.yml up --force-recreate -V --remove-orphans
    docker compose -f docker/docker-compose-test.yml rm -f

up_postgres:
    docker-compose --project-name dev -f docker/docker-compose-dev.yml down -v
    docker-compose --project-name dev -f docker/docker-compose-dev.yml up postgres --remove-orphans --renew-anon-volumes

docker_dump:
    docker exec gbans-postgres pg_dump -U gbans > gbans.sql

docker_restore:
    cat gbans.sql | docker exec -i docker-postgres-1 psql -U gbans

run_docker_snapshot: builds
    docker build . --no-cache -t gbans:snapshot
    docker run -it -v ./gbans.yml:/app/gbans.yml -v ./.cache:/app/.cache -p 6006:6006  gbans:snapshot

docs_install:
    just -f docs/justfile install

docs_start:
    just -f docs/justfile start

docs_deploy:
    just -f docs/justfile deploy

docs_build:
    just -f docs/justfile build
