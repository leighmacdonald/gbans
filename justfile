set dotenv-load

alias c := check
alias f := fmt
alias d := dev

SOURCEMOD_INCLUDE_DIR := shell('dirname $(which spcomp64)') + "/include"

all: build-backend build-frontend build-sourcemod build-docs

test: test-backend test-frontend

fmt: fmt-proto fmt-backend fmt-md fmt-frontend fmt-sourcemod

check: lint-proto lint-go vulncheck lint-frontend typecheck-frontend lint-md

update: update-backend update-frontend

lint-nil:
    @nilaway -include-pkgs="github.com/leighmacdonald/gbans" -exclude-pkgs="github.com/jackc/pgx/v5" -test=false ./...

lint-go:
    @golangci-lint run --timeout 3m ./...

generate:
    @go generate ./...
    @just fmt

run:
    go run -race . serve

fmt-proto:
    @buf format -w

fmt-md:
    @markdownlint-cli2 --fix

lint-md:
    @markdownlint-cli2

fmt-backend:
    @golangci-lint fmt

test-backend:
    @go test -race ./...

build-backend:
    @goreleaser release --clean

build-backend-snapshot:
    @goreleaser release --clean --snapshot

frontend:
    @just -f frontend/justfile

run-forever:
    while true; do go run -race . serve; sleep 1; done

air:
    while true; do air -c .air.toml -- serve; sleep 1; done

update-backend:
    go get -u ./...

upload-plugin:
    @just sourcemod
    @scp sourcemod/plugins/gbans.smx $SOURCEMOD_SCP_URI
    @rcon-cli --host $SOURCEMOD_HOST --port $SOURCEMOD_PORT --password $SOURCEMOD_RCON "$SOURCEMOD_RELOAD_COMMAND"

vulncheck:
    govulncheck -show verbose ./...

lint-proto:
    @buf lint

fix: fmt
    @golangci-lint run --fix

clean: clean-backend clean-frontend clean-docs clean-sourcemod

clean-backend:
    go clean -i
    rm -rf ./dist/

docker-dump:
    docker exec gbans-postgres pg_dump -U gbans > gbans.sql

docker-restore:
    cat gbans.sql | docker exec -i docker-postgres-1 psql -U gbans

run-docker-snapshot: build-backend-snapshot
    docker run -it -v ./gbans.yml:/app/gbans.yml -v ./.cache:/app/.cache -p 6006:6006 ghcr.io/leighmacdonald/gbans:latest-amd64

[working-directory('docker')]
db:
    ./dev_db.sh

demostats-serve:
    $DEMOSTATS_BIN update --api-key $STEAM_KEY
    $DEMOSTATS_BIN serve

dev:
    @zellij --layout .zellij.kdl attach --create gbans

pgcli host=`sed -n 's/^database_dsn: //p' gbans.yml`:
    @pgcli "{{ host }}"

[working-directory('frontend')]
install-frontend:
    pnpm install --frozen-lockfile

[working-directory('frontend')]
build-frontend:
    pnpm run build

[working-directory('frontend')]
watch-frontend:
    pnpm run watch

[working-directory('frontend')]
fmt-frontend:
    @pnpm run fmt

[working-directory('frontend')]
serve-frontend:
    pnpm run serve

[working-directory('frontend')]
update-frontend:
    pnpm update -i

[working-directory('frontend')]
test-frontend:
    pnpm run test

[working-directory('frontend')]
typecheck-frontend:
    pnpm run typecheck

[working-directory('frontend')]
lint-frontend:
    pnpm run check

[working-directory('frontend')]
clean-frontend:
    rm -rf dist
    rm -rf node_modules

[working-directory('docs')]
install-docs:
    pnpm install

[working-directory('docs')]
start-docs:
    pnpm start

[working-directory('docs')]
deploy-docs:
    pnpm deploywhat i

[working-directory('docs')]
build-docs:
    pnpm build

[working-directory('docs')]
clean-docs:
    rm -rf build
    rm -rf node_modules

[working-directory('sourcemod')]
clean-sourcemod:
    rm -rf ./plugins/gbans.smx

[working-directory('sourcemod')]
sourcemod-devel: build-sourcemod
    docker cp plugins/gbans.smx srcds-localhost-1:/home/tf2server/tf-dedicated/tf/addons/sourcemod/plugins/
    docker restart srcds-localhost-1

[working-directory('sourcemod')]
fmt-sourcemod:
    find scripting/ -not -path "./include/*" -iname gbans.inc -o -iname *.sp -type f -exec clang-format -style=file -i {} \;

[working-directory('sourcemod')]
build-sourcemod:
    spcomp64 scripting/gbans.sp -o plugins/gbans.smx -i{{ SOURCEMOD_INCLUDE_DIR }} -i scripting/include -v2

[working-directory('sourcemod')]
copy-sourcemod:
    cp -rv scripting/* $(HOME)/projects/uncletopia/roles/sourcemod/files/addons/sourcemod/scripting/

[working-directory('sourcemod')]
upload-sourcemod host="tst-1.internal.uncletopia.com" rootDir="~/srcds-tst-1/tf/addons/sourcemod" port="27015" password="testtest": build-sourcemod
    scp -r plugins configs {{ host }}:{{ rootDir }}/
    rcon-cli --host {{ host }} --port {{ port }} --password {{ password }} sm plugins refresh
