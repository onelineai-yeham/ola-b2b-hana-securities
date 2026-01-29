.PHONY: build run test clean swagger docker-build docker-push deploy

# Variables
APP_NAME := hana-news-api
DOCKER_REGISTRY := asia-northeast3-docker.pkg.dev/finola-global/ola-b2b
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build
build:
	go build -ldflags="-w -s -X main.version=$(VERSION)" -o bin/$(APP_NAME) ./cmd/server

# Run locally
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf docs/

# Generate swagger docs
swagger:
	swag init -g cmd/server/main.go -o docs

# Install tools
tools:
	go install github.com/swaggo/swag/cmd/swag@latest

# Docker build
docker-build:
	docker build -t $(DOCKER_REGISTRY)/$(APP_NAME):$(VERSION) .
	docker tag $(DOCKER_REGISTRY)/$(APP_NAME):$(VERSION) $(DOCKER_REGISTRY)/$(APP_NAME):latest

# Docker push
docker-push:
	docker push $(DOCKER_REGISTRY)/$(APP_NAME):$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(APP_NAME):latest

# Deploy to k8s
deploy:
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/secret.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml

# Run migration on gold DB
migrate:
	@echo "Run: psql -h <host> -U <user> -d hana_securities -f migrations/001_create_gold_tables.sql"

# Tidy dependencies
tidy:
	go mod tidy

# Full build pipeline
all: tidy swagger build
