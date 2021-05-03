.PHONY: all test clean build install frontend
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GIT_TAG =
GO_FLAGS = -ldflags "-X 'github.com/leighmacdonald/gbans/service.BuildVersion=`git describe --abbrev=0`'"
DEBUG_FLAGS = -gcflags "all=-N -l"

current_dir :=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

all: frontend build

vet:
	@go vet . ./...

fmt:
	@go fmt . ./...

build_debug:
	@go build $(DEBUG_FLAGS) $(GO_FLAGS) -o gbans

build: fmt vet linux64 windows64

frontend:
	cd frontend && yarn && yarn run build && yarn run copy

linux64:
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o build/linux64/gbans main.go

windows64:
	GOOS=windows GOARCH=amd64 $(GO_BUILD) $(GO_FLAGS) -o build/win64/gbans.exe main.go

dist: frontend build
	zip -j gbans-`git describe --abbrev=0`-win64.zip build/win64/gbans.exe LICENSE README.md
	zip -r gbans-`git describe --abbrev=0`-win64.zip migrations/ docs/
	zip -j gbans-`git describe --abbrev=0`-lin64.zip build/linux64/gbans LICENSE README.md
	zip -r gbans-`git describe --abbrev=0`-lin64.zip migrations/ docs/

dist-master: frontend build
	zip -j gbans-master-win64.zip build/win64/gbans.exe LICENSE README.md
	zip -r gbans-master-win64.zip migrations/ docs/
	zip -j gbans-master-lin64.zip build/linux64/gbans LICENSE README.md
	zip -r gbans-master-lin64.zip migrations/ docs/

run:
	@go run $(GO_FLAGS) -race main.go

install:
	@go install $(GO_FLAGS) ./...

test-ts:
	@cd frontend && yarn run test

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

image_tag:
	docker build -t leighmacdonald/gbans:$$(git describe --abbrev=0 --tags) .

docker_run:
	docker run -it --rm -v "$(current_dir)"/gbans.yml:/app/gbans.yml:ro leighmacdonald/gbans:latest

up:
	docker-compose -f docker/docker-compose.yml up --build --remove-orphans --abort-on-container-exit --exit-code-from gbans
