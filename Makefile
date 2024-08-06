-include local-tools/Makefile
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
current_dir := $(notdir $(patsubst %/,%,$(dir $(mkfile_path))))

BIN ?= $(shell pwd)/bin
GIT_TAG ?= $(shell git describe --always)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
PROJECT_NAME:=$(current_dir)
PROJECT_PATH ?= $(shell pwd -P)
SRC:=$(shell find cmd -iname main.go -type f)
export MOD_PATH := /go/src/$(shell grep module go.mod | awk '{print $$2}')
DOCKER_REPO ?= localhost:5000/slyngshot
REPOSITORY ?= ${DOCKER_REPO}/${PROJECT_NAME}
REPO_LOW=$(shell echo $(REPOSITORY) | tr A-Z a-z)
INTEGRATION_TEST ?= NO
DATABASE_DSN ?= postgres://postgres:mysecretpassword@localhost:5432/matchmaking?sslmode=disable

.PHONY: build $(SRC)
$(SRC):
	echo $(shell dirname $@)
	cd $(shell dirname $@) && CGO_ENABLED=0 go build -o $(BIN)/ -ldflags="-s -extldflags=-static -X main.Version=${GIT_TAG} -X istio.io/pkg/version.buildVersion=${GIT_TAG} -X istio.io/pkg/version.buildGitRevision=${GIT_COMMIT}" . && cd -

.PHONY: deps
deps:
	go mod tidy -compat=1.20
	go mod download

.PHONY: check-deps
check-deps:
	@go list -u -m -mod=mod -f '{{if not .Indirect}}{{if .Update}}{{.}}{{end}}{{end}}' all

.PHONY: build
build: $(SRC)

lint:
	golangci-lint run ./...

.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
	go install github.com/99designs/gqlgen@v0.17.46
	go install github.com/vektra/mockery/v2@v2.29.0
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2

.PHONY: graph
graph:
	gqlgen generate --config ./api/graph/gqlgen.yml

.PHONY: api
api:
	@mkdir -p ${PROJECT_PATH}/internal/server/webapi/api
	@mkdir -p ${PROJECT_PATH}/openapi
	oapi-codegen -config  ./api/openapi/api.gen.cfg.yaml ./api/openapi/api.yaml && \
		cp ./api/openapi/api.yaml ./openapi/api.yaml

.PHONY: gen
gen: tools
	go generate ./...

image: export GOOS=linux
image: export GOARCH=amd64
image: export CGO_ENABLED=0
image: $(SRC)
	GOOS=linux DOCKER_BUILDKIT=1 docker build -t "${REPO_LOW}:${IMAGE_TAG}" --build-arg PROJECT_NAME="${PROJECT_NAME}" -f deploy/Dockerfile .

push_image: image
	docker push "${REPO_LOW}:${IMAGE_TAG}"

test:
	go test ./...

test-integration:
	INTEGRATION_TEST=YES DATABASE_DSN=${DATABASE_DSN} go test ./...

test-dev-local:
	DEV_LOCAL_TEST=YES go test -v ./...

.PHONY: run-webapi
run-webapi: build
	go run ./cmd/app/. -c ./configs/config.yaml start webapi

# see https://developer.confluent.io/quickstart/kafka-docker/
.PHONY: start-local-kafka
start-local-kafka:
	docker-compose -f ./tools/docker-compose-kafka.yml up -d

.PHONY: stop-local-kafka
stop-local-kafka:
	docker-compose -f ./tools/docker-compose-kafka.yml down

.PHONY: start-local-db
start-local-db:
	docker-compose -f ./tools/docker-compose-postgres.yml up -d

.PHONY: stop-local-db
stop-local-db:
	docker-compose -f ./tools/docker-compose-postgres.yml down

.PHONY: reinit-local-db
reinit-local-db:
	make stop-local-db
	make start-local-db
	sleep 2
	make migrate

.PHONY: compose-up
compose-up: reinit-local-db start-local-kafka

.PHONY: compose-down
compose-down: stop-local-db

.PHONY: migrate
migrate:
	@docker run -v ${PROJECT_PATH}/migrations:/migrations --network host migrate/migrate \
		-path=/migrations/ -database ${DATABASE_DSN} up

.PHONY: migrate_down
migrate_down:
	@docker run -v ${PROJECT_PATH}/migrations:/migrations --network host migrate/migrate \
		-path=/migrations/ -database ${DATABASE_DSN} down

migrate-ci:
	cd cmd/app && go run . --config-file=../../configs/config_ci.yaml migrate up --location=file://../../migrations && cd -
