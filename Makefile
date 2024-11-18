# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=microservices

# Docker parameters
DOCKER_COMPOSE="docker compose"
DOCKER=docker

# Kubernetes parameters
KUBECTL=kubectl

# Service directories
USER_SERVICE_DIR=./user-service
ORDER_SERVICE_DIR=./order-service

# Docker image names
USER_SERVICE_IMAGE=user-service:latest
ORDER_SERVICE_IMAGE=order-service:latest

.PHONY: all build test clean deps docker-build docker-push k8s-deploy

all: clean deps build test

# Build commands
build: build-user build-order

build-user:
	cd $(USER_SERVICE_DIR) && $(GOBUILD) -o bin/user-service

build-order:
	cd $(ORDER_SERVICE_DIR) && $(GOBUILD) -o bin/order-service

# Test commands
test: test-user test-order

test-user:
	cd $(USER_SERVICE_DIR) && $(GOTEST) -v ./...

test-order:
	cd $(ORDER_SERVICE_DIR) && $(GOTEST) -v ./...

test-integration:
	cd $(USER_SERVICE_DIR)/tests && $(GOTEST) -tags=integration -v
	cd $(ORDER_SERVICE_DIR)/tests && $(GOTEST) -tags=integration -v

# Clean build artifacts
clean:
	rm -f $(USER_SERVICE_DIR)/bin/*
	rm -f $(ORDER_SERVICE_DIR)/bin/*

# Install dependencies
deps:
	$(GOMOD) download
	cd $(USER_SERVICE_DIR) && $(GOMOD) tidy
	cd $(ORDER_SERVICE_DIR) && $(GOMOD) tidy

# Docker commands
docker-build:
	$(DOCKER) build -t $(USER_SERVICE_IMAGE) $(USER_SERVICE_DIR)
	$(DOCKER) build -t $(ORDER_SERVICE_IMAGE) $(ORDER_SERVICE_DIR)

docker-push:
	$(DOCKER) push $(USER_SERVICE_IMAGE)
	$(DOCKER) push $(ORDER_SERVICE_IMAGE)

# Docker Compose commands
compose-up:
	$(DOCKER_COMPOSE) up --build -d

compose-down:
	$(DOCKER_COMPOSE) down

compose-logs:
	$(DOCKER_COMPOSE) logs -f

# Kubernetes commands
k8s-deploy:
	$(KUBECTL) apply -k deployments/kubernetes

k8s-delete:
	$(KUBECTL) delete -k deployments/kubernetes

k8s-status:
	$(KUBECTL) get pods -n microservices
	$(KUBECTL) get svc -n microservices

# Infrastructure commands
infra-up:
	$(DOCKER_COMPOSE) up -d mysql jaeger prometheus

infra-down:
	$(DOCKER_COMPOSE) down

# Run services locally
run-user:
	cd $(USER_SERVICE_DIR) && $(GOCMD) run main.go

run-order:
	cd $(ORDER_SERVICE_DIR) && $(GOCMD) run main.go

# Generate proto files
proto-gen:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/**/*.proto

# Linting and formatting
lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

# Database migrations
migrate-up:
	migrate -path db/migrations -database "mysql://root:password@tcp(localhost:3306)/users" up

migrate-down:
	migrate -path db/migrations -database "mysql://root:password@tcp(localhost:3306)/users" down

# Help command
help:
	@echo "Available commands:"
	@echo "  make build          - Build all services"
	@echo "  make test           - Run all tests"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make deps           - Install dependencies"
	@echo "  make docker-build   - Build Docker images"
	@echo "  make compose-up     - Start services with Docker Compose"
	@echo "  make k8s-deploy     - Deploy to Kubernetes"
	@echo "  make infra-up       - Start infrastructure services"
	@echo "  make run-user       - Run user service locally"
	@echo "  make run-order      - Run order service locally"
	@echo "  make proto-gen      - Generate proto files"
	@echo "  make lint           - Run linter"
	@echo "  make fmt            - Format code"
	@echo "  make migrate-up     - Run database migrations"