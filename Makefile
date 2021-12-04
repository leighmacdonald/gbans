.PHONY: all test clean build install frontend sourcemod
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GIT_TAG =
GO_FLAGS = -ldflags "-X 'github.com/leighmacdonald/gbans/service.BuildVersion=`git describe --abbrev=0`'"
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

build: fmt vet linux64 windows64

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
	@./sm.sh

install:
	@go install $(GO_FLAGS) ./...

test: test-go test-ts

test-ts:
	@cd frontend && yarn && yarn run test

test-go:
	@go test $(GO_FLAGS) -race -cover . ./...

testcover:
	@go test -race -coverprofile c.out $(GO_FLAGS) ./...

lint:
	@golangci-lint run

bench:
	@go test -run=NONE -bench=. $(GO_FLAGS) ./...

clean:
	@go clean $(GO_FLAGS) -i

pg_test_service:
	docker-compose -f docker/docker-compose.yml up --abort-on-container-exit --exit-code-from postgres --remove-orphans --build postgres

docker_test:
	docker-compose -f docker/docker-compose-test.yml up --renew-anon-volumes --abort-on-container-exit --exit-code-from gbans-test --remove-orphans --build

image_latest:
	@docker build -t leighmacdonald/gbans:latest .

publish_latest: image_latest
	@docker push leighmacdonald/gbans:latest

image_tag:
	docker build -t leighmacdonald/gbans:$$(git describe --abbrev=0 --tags) .

docker_run:
	docker run -it --rm -v "$(pwd)"/gbans.yml:/app/gbans.yml:ro leighmacdonald/gbans:latest

up:
	docker-compose -f docker/docker-compose.yml up --build --remove-orphans --abort-on-container-exit --exit-code-from gbans

docker_dump:
	docker exec gbans-postgres pg_dump -U gbans > gbans.sql

docker_restore:
	docker exec gbans-postgres pg_dump -U gbans -f gbans.sql
