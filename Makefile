.PHONY: all test clean build install frontend sourcemod build
VERSION=v0.7.14
GO_CMD=go
GO_BUILD=$(GO_CMD) build
DEBUG_FLAGS = -gcflags "all=-N -l"
ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

all: frontend sourcemod buildp

fmt:
	gci write . --skip-generated -s standard -s default
	gofumpt -l -w .
	cd frontend && pnpm prettier src/ --write

bump_deps:
	go get -u ./...
	cd frontend && pnpm update -i

buildp: frontend
	goreleaser release --clean

builds: frontend
	goreleaser release --clean --snapshot

watch:
	cd frontend && pnpm run watch

serve:
	cd frontend && pnpm run serve

frontend:
	cd frontend && pnpm install --frozen-lockfile && pnpm run build

dist: frontend build
	zip -j gbans-`git describe --abbrev=0`-win64.zip build/win64/gbans.exe LICENSE README.md gbans_example.yml
	zip -r gbans-`git describe --abbrev=0`-win64.zip docs/
	zip -j gbans-`git describe --abbrev=0`-lin64.zip build/linux64/gbans LICENSE README.md gbans_example.yml
	zip -r gbans-`git describe --abbrev=0`-lin64.zip docs/

dist-master: frontend build
	zip -j gbans-master-win64.zip build/win64/gbans.exe LICENSE README.md gbans_example.yml
	zip -r gbans-master-win64.zip docs/
	zip -j gbans-master-lin64.zip build/linux64/gbans LICENSE README.md gbans_example.yml
	zip -r gbans-master-lin64.zip docs/

run:
	@go run $(GO_FLAGS) -race main.go

sourcemod:
	make -C sourcemod

sourcemod_devel: sourcemod
	docker cp sourcemod/plugins/gbans.smx srcds-localhost-1:/home/tf2server/tf-dedicated/tf/addons/sourcemod/plugins/
	docker restart srcds-localhost-1

test: test-go test-ts

test-ts:
	@cd frontend && pnpm run test

test-go:
	@go test $(GO_FLAGS) -race -cover . ./...

install_deps:
	go install github.com/daixiang0/gci@v0.13.4
	go install mvdan.cc/gofumpt@v0.6.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1
	go install honnef.co/go/tools/cmd/staticcheck@v0.4.7

check: lint_golangci static lint_ts typecheck_ts

lint_golangci:
	golangci-lint run --timeout 3m ./...

fix: fmt
	golangci-lint run --fix

lint_ts:
	cd frontend && pnpm run eslint:check && pnpm prettier src/ --check

typecheck_ts:
	cd frontend && pnpm run typecheck

static:
	staticcheck -go 1.22 ./...

clean:
	@go clean $(GO_FLAGS) -i
	rm -rf ./build/
	rm -rf ./frontend/dist
	rm -rf ./frontend/node_modules
	rm -rf ./sourcemod/plugins/gbans.smx

docker_test:
	@docker compose -f docker/docker-compose-test.yml up --force-recreate -V --remove-orphans
	@docker compose -f docker/docker-compose-test.yml rm -f

up_postgres:
	docker-compose --project-name dev -f docker/docker-compose-dev.yml down -v
	docker-compose --project-name dev -f docker/docker-compose-dev.yml up postgres --remove-orphans --renew-anon-volumes

docker_dump:
	docker exec gbans-postgres pg_dump -U gbans > gbans.sql

docker_restore:
	cat gbans.sql | docker exec -i docker-postgres-1 psql -U gbans

run_docker_snapshot: builds
	docker build . --no-cache -t gbans:snapshot
	docker run -it -v ./gbans.yml:/app/gbans.yml -p 6006:6006  gbans:snapshot

docs_setup:
	cd docs && pnpm i

docs_start:
	cd docs && pnpm start

docs_deploy:
	cd docs && pnpm deploy

docs_build:
	cd docs && pnpm build