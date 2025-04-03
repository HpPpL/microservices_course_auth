include .env

# env
LOCAL_BIN:=$(CURDIR)/bin
LOCAL_MIGRATION_DIR=$(MIGRATION_DIR)
LOCAL_MIGRATION_DSN="host=localhost port=$(PG_PORT) dbname=$(PG_DATABASE_NAME) user=$(PG_USER) password=$(PG_PASSWORD) sslmode=disable"

# general
install-deps:
	make install-grpc-deps
	make install-goose-deps
	make install-golangci-lint-deps

# go

goimports-format:
	goimports -w cmd/grpc_server/main.go
# grpc
install-grpc-deps:
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	GOBIN=$(LOCAL_BIN) go install -mod=mod google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

get-deps:
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc

generate:
	make generate-auth-api

generate-auth-api:
	mkdir -p pkg/auth_v1
	protoc --proto_path api/auth_v1 \
	--go_out=pkg/auth_v1 --go_opt=paths=source_relative \
	--plugin=protoc-gen-go=bin/protoc-gen-go \
	--go-grpc_out=pkg/auth_v1 --go-grpc_opt=paths=source_relative \
	--plugin=protoc-gen-go-grpc=bin/protoc-gen-go-grpc \
	api/auth_v1/auth.proto

# linter
lint: 
	$(LOCAL_BIN)/golangci-lint run ./... -v --config .golangci.pipeline.yaml --max-issues-per-linter 200

install-golangci-lint-deps:
	GOBIN=$(LOCAL_BIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3

# docker regestry
docker-build-and-push:
	make docker-build
	docker login -u $(DOCKER_USER) -p $(DOCKER_PASSWORD) $(DOCKER_REGISTRY)
	docker push $(DOCKER_REGISTRY):v0.0.1

docker-build:
	docker build -t $(DOCKER_REGISTRY):v0.0.1 .

# goose
local-migration-status:
	${LOCAL_BIN}/goose -dir ${MIGRATION_DIR} postgres ${PG_DSN} status -v

local-migration-up:
	${LOCAL_BIN}/goose -dir ${MIGRATION_DIR} postgres ${PG_DSN} up -v

local-migration-down:
	${LOCAL_BIN}/goose -dir ${MIGRATION_DIR} postgres ${PG_DSN} down -v

goose-init:
	${LOCAL_BIN}/goose init

install-goose-deps:
	GOBIN=$(LOCAL_BIN) go install github.com/pressly/goose/v3/cmd/goose@v3.14.0
