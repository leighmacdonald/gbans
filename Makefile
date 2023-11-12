.PHONY: all test clean build install frontend sourcemod
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
TAGGED_IMAGE = ghcr.io/leighmacdonald/gbans:$(BRANCH)
VERSION=v0.5.1
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_FLAGS = -trimpath -ldflags="-s -w -X github.com/leighmacdonald/gbans/internal/app.BuildVersion=$(VERSION)"
DEBUG_FLAGS = -gcflags "all=-N -l"

all: frontend sourcemod build

vet:
	@go vet . ./...

fmt:
	gci write . --skip-generated -s standard -s default
	gofumpt -l -w .
	cd frontend && yarn prettier src/ --write

build_debug:
	@go build $(DEBUG_FLAGS) $(GO_FLAGS) -o gbans

bump_deps:
	go get -u ./...
	cd frontend && yarn upgrade-interactive

build: linux64

frontend:
	cd frontend && yarn && yarn run build

linux64:
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o build/linux64/gbans  main.go

windows64:
	GOOS=windows GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o build/win64/gbans.exe main.go

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

install:
	@go install $(GO_FLAGS) ./...

test: test-go test-ts

test-ts:
	@cd frontend && yarn && yarn run test --passWithNoTests

test-go:
	@go test $(GO_FLAGS) -race -cover . ./...

testcover:
	@go test -race -coverprofile c.out $(GO_FLAGS) ./...

check_deps:
	go install github.com/daixiang0/gci@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
	go install honnef.co/go/tools/cmd/staticcheck@latest

check: lint_golangci static lint_ts

lint_golangci:
	golangci-lint run --timeout 3m ./...

fix: fmt
	golangci-lint run --fix

lint_ts:
	cd frontend && yarn run eslint:check && yarn prettier src/ --check

static:
	staticcheck -go 1.21 ./...

bench:
	@go test -run=NONE -bench=. $(GO_FLAGS) ./...

clean:
	@go clean $(GO_FLAGS) -i

docker_test_postgres:
	@docker-compose --project-name testing -f docker/docker-compose-test.yml down -v
	@docker-compose --project-name testing -f docker/docker-compose-test.yml up --pull --exit-code-from postgres-test --remove-orphans postgres-test

docker_test:
	@docker-compose --project-name testing -f docker/docker-compose-test.yml down -v
	@docker-compose --project-name testing -f docker/docker-compose-test.yml up --build --force-recreate --renew-anon-volumes --abort-on-container-exit --exit-code-from gbans-test --remove-orphans gbans-test

image_latest:
	@docker build -t leighmacdonald/gbans:latest .

publish_latest: image_latest
	@docker push leighmacdonald/gbans:latest

image_tag:
	docker build -t leighmacdonald/gbans:$$(git describe --abbrev=0 --tags) .

docker_run:
	docker run -it --rm -v "$(pwd)"/gbans.yml:/app/gbans.yml:ro leighmacdonald/gbans:latest

up_postgres:
	docker-compose --project-name dev -f docker/docker-compose-dev.yml down -v
	docker-compose --project-name dev -f docker/docker-compose-dev.yml up postgres --remove-orphans --renew-anon-volumes

up:
	docker-compose --project-name dev -f docker/docker-compose-dev.yml down -v
	docker-compose --project-name dev -f docker/docker-compose-dev.yml up --build --remove-orphans --abort-on-container-exit --exit-code-from gbans

docker_dump:
	docker exec gbans-postgres pg_dump -U gbans > gbans.sql

docker_restore:
	cat gbans.sql | docker exec -i docker-postgres-1 psql -U gbans

docker_update_plugin:
	docker cp sourcemod/plugins/gbans.smx srcds-localhost-1:/home/tf2server/tf-dedicated/tf/addons/sourcemod/plugins/gbans.smx
	rcon -H 192.168.0.57 -p dev_pass sm plugins reload gbans
	docker logs -f srcds-localhost-1

copy_ut:
	cp -rv sourcemod/scripting/* ../uncletopia/roles/sourcemod/files/addons/sourcemod/scripting/

