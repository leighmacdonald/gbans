set dotenv-load := true

alias c := check
alias f := fmt
alias d := dev

all: frontend sourcemod buildp

fmt: fmt-proto fmt-go fmt-md

fmt-go:
    golangci-lint fmt
    just -f frontend/justfile fmt

test-go:
    go test -race ./...

lint-nil:
    nilaway -include-pkgs="github.com/leighmacdonald/gbans" -exclude-pkgs="github.com/jackc/pgx/v5" -test=false ./...

lint-go:
    golangci-lint run --timeout 3m ./...

generate:
    go generate ./...

run:
    go run -race . serve

fmt-proto:
    buf format -w

fmt-md:
    markdownlint-cli2 --fix

lint-md:
    markdownlint-cli2

bump-deps:
    go get -u ./...
    just -f frontend/justfile update

buildp:
    goreleaser release --clean

builds:
    goreleaser release --clean --snapshot

serve:
    just -f frontend/justfile serve

frontend:
    just -f frontend/justfile

run-forever:
    while true; do go run -race . serve; sleep 1; done

air:
    while true; do air -c .air.toml -- serve; sleep 1; done

sourcemod:
    just -f sourcemod/justfile

sourcemod-devel: sourcemod
    docker cp sourcemod/plugins/gbans.smx srcds-localhost-1:/home/tf2server/tf-dedicated/tf/addons/sourcemod/plugins/
    docker restart srcds-localhost-1

test: test-go test-ts

test-ts:
    just -f frontend/justfile test

check: lint-proto lint-go vulncheck lint-ts typecheck-ts lint-md

vulncheck:
    govulncheck

lint-proto:
    @buf lint

fix: fmt
    golangci-lint run --fix

lint-ts:
    just -f frontend/justfile lint

typecheck-ts:
    just -f frontend/justfile typecheck

clean:
    go clean -i
    rm -rf ./dist/
    just -f frontend/justfile clean
    rm -rf ./sourcemod/plugins/gbans.smx

docker-dump:
    docker exec gbans-postgres pg_dump -U gbans > gbans.sql

docker-restore:
    cat gbans.sql | docker exec -i docker-postgres-1 psql -U gbans

run-docker-snapshot: builds
    docker run -it -v ./gbans.yml:/app/gbans.yml -v ./.cache:/app/.cache -p 6006:6006 ghcr.io/leighmacdonald/gbans:latest-amd64

docs-install:
    just -f docs/justfile install

docs-start:
    just -f docs/justfile start

docs-deploy:
    just -f docs/justfile deploy

docs-build:
    just -f docs/justfile build

db:
    pushd docker && ./dev_db.sh

demostats-serve:
    ../tf2_demostats/dist/cli_x86_64-unknown-linux-musl/tf2_demostats update --api-key $STEAM_KEY
    ../tf2_demostats/dist/cli_x86_64-unknown-linux-musl/tf2_demostats serve

dev:
    zellij --layout .zellij.kdl

psql host=`sed -n 's/^database_dsn: //p' gbans.yml`:
    @psql {{ host }}
