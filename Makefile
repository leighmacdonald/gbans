.PHONY: all test clean build install frontend sourcemod
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
TAGGED_IMAGE = ghcr.io/leighmacdonald/gbans:$(BRANCH)

GO_CMD=go
GO_BUILD=$(GO_CMD) build
DEBUG_FLAGS = -gcflags "all=-N -l"

all: frontend sourcemod build

vet:
	@go vet . ./...

fmt:
	@go fmt . ./...

build_debug:
	@go build $(DEBUG_FLAGS) $(GO_FLAGS) -o gbans

bump_deps:
	go get -u ./...
	cd frontend && yarn upgrade-interactive --latest

build: fmt vet linux64

frontend:
	cd frontend && yarn && yarn run build

linux64:
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o build/linux64/gbans main.go

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
	@./sourcemod/build/package/addons/sourcemod/scripting/spcomp -i./sourcemod/include sourcemod/gbans.sp -ogbans.smx

install:
	@go install $(GO_FLAGS) ./...

test: test-go test-ts

test-ts:
	@cd frontend && yarn && yarn run test --passWithNoTests

test-go:
	@go test $(GO_FLAGS) -race -cover . ./...

testcover:
	@go test -race -coverprofile c.out $(GO_FLAGS) ./...

lint-ts:
	cd frontend && eslint . --ext .ts,.tsx

lint:
	@golangci-lint run

bench:
	@go test -run=NONE -bench=. $(GO_FLAGS) ./...

clean:
	@go clean $(GO_FLAGS) -i

docker_test_postgres:
	@docker-compose -f docker/docker-compose-test.yml down
	@docker-compose -f docker/docker-compose-test.yml up --exit-code-from postgres-test --remove-orphans postgres-test

docker_test:
	@docker-compose -f docker/docker-compose-test.yml pull
	@docker-compose -f docker/docker-compose-test.yml down
	@docker-compose -f docker/docker-compose-test.yml build --no-cache
	@docker-compose -f docker/docker-compose-test.yml up --renew-anon-volumes --exit-code-from gbans-test --remove-orphans gbans-test

image_latest:
	@docker build -t leighmacdonald/gbans:latest .

publish_latest: image_latest
	@docker push leighmacdonald/gbans:latest

image_tag:
	docker build -t leighmacdonald/gbans:$$(git describe --abbrev=0 --tags) .

docker_run:
	docker run -it --rm -v "$(pwd)"/gbans.yml:/app/gbans.yml:ro leighmacdonald/gbans:latest

up_postgres:
	docker-compose -f docker/docker-compose.yml up -d postgres

up:
	docker-compose -f docker/docker-compose.yml up --build --remove-orphans --abort-on-container-exit --exit-code-from gbans

docker_dump:
	docker exec gbans-postgres pg_dump -U gbans > gbans.sql

docker_restore:
	cat gbans.sql | docker exec -i docker-postgres-1 psql -U gbans

# This will dump the current local tf2 server into a local directory.
# This allows he following:
# - Uses the actual configs generated from ansible deployment to local host
# - Faster iterations of small changes
# - Simple use of vscode sourcepawn plugin to automatically write the plugin to the correct dir and reload via rcon
# - Use the identical sourcemod versions to prod
# Alternatively you can create a build script to copy the plugin into the running container and reload via rcon
docker_generate_local_server:
	rm -rf tf2server || true
	docker cp -a srcds-localhost-1:/home/steam tf2server
	local_server

local_server:
	cd tf2server && ../scripts/test_game_server.sh
