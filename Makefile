.PHONY: all test clean build install
GIT_TAG =
GO_FLAGS = -ldflags "-X 'github.com/leighmacdonald/gbans/service.BuildVersion=`git describe --abbrev=0`'"
DEBUG_FLAGS = -gcflags "all=-N -l"

all: build

vet:
	@go vet . ./...

fmt:
	@go fmt . ./...

build_debug:
	@go build $(DEBUG_FLAGS) $(GO_FLAGS) -o gbans

build: fmt vet
	@go build $(GO_FLAGS)

run:
	@go run $(GO_FLAGS) -race main.go

install:
	@go install $(GO_FLAGS) ./...

test:
	@cd frontend && yarn run test
	@cd ..
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
	docker-compose -f docker/docker-compose-test.yml up --abort-on-container-exit --exit-code-from gbans-test --remove-orphans --build

image_latest:
	@docker build -t leighmacdonald/gbans:latest .

image_tag:
	docker build -t leighmacdonald/gbans:$$(git describe --abbrev=0 --tags) .

docker_run:
	docker-compose -f docker/docker-compose.yml up --build --abort-on-container-exit --exit-code-from gbans
