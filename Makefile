MIN_TEST_COVERAGE := 90
SCRIPTS_DIR := scripts
APP_NAME := $(shell head -n 1 go.mod | awk '{print $$2}' | awk -F'/' '{print $$NF}')

.PHONY: all dep lint lint-fix test test-coverage build \
	service service-remove service-registration servicepack-update own

all: dep lint test ## Run dep, lint and test

dep: ## Get project dependencies
	@echo "Getting project dependencies..."
	@go mod tidy
	@go mod vendor

lint: ## Lint all Golang files
	@echo "Linting all Go files..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./...
	@go tool golangci-lint run --timeout=30m0s ./...

lint-fix: ## Lint all Golang files and fix
	@echo "Linting all Go files..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...
	@go tool golangci-lint run --fix --timeout=30m0s ./...

test: ## Run all tests
	@echo "Running all tests..."
	@go test -race ./...

test-coverage: ## Run tests with coverage check. Fails if coverage is below the threshold.
	@echo "Running tests with coverage check..."
	@trap 'rm -f coverage.txt' EXIT; \
	go test -race -coverprofile=coverage.txt $$(go list ./... | grep -v hello-world | grep -v /cmd); \
	if [ $$? -ne 0 ]; then \
		echo "Test failed. Exiting."; \
		exit 1; \
	fi; \
	result=$$(go tool cover -func=coverage.txt | grep -oP 'total:\s+\(statements\)\s+\K\d+' || echo "0"); \
	if [ $$result -eq 0 ]; then \
		echo "No test coverage information available."; \
		exit 0; \
	elif [ $$result -lt $(MIN_TEST_COVERAGE) ]; then \
		echo "FAIL: Coverage $$result% is less than the minimum $(MIN_TEST_COVERAGE)%"; \
		exit 1; \
	fi

build: ## Build the app binary using Docker
	@echo "Building $(APP_NAME) binary..."
	@mkdir -p ./build
	@docker run --rm \
		-v $(PWD):/app \
		-w /app \
		-e USER_UID=$(shell id -u) \
		-e USER_GID=$(shell id -g) \
		golang:1.24.6-alpine \
		sh -c "apk add --no-cache gcc musl-dev && \
				CGO_ENABLED=0 go build -a \
				-ldflags '-extldflags \"-static\" -X main.appName=$(APP_NAME)' \
				-o ./build/$(APP_NAME) ./cmd/... && \
				chown \$$USER_UID:\$$USER_GID ./build/$(APP_NAME)"

docker-build-dev: ## Build the development Docker image
	@echo "Building the development Docker image..."
	@docker build -f Dockerfile.dev -t $(APP_NAME)-dev .

run-dev: docker-build-dev ## Run in the development Docker image
	@echo "Starting the containerized development environment..."
	@docker run -i --rm \
		--name $(APP_NAME)-dev \
		$(APP_NAME)-dev sh -c "CGO_ENABLED=1 go build -race -o ./build/$(APP_NAME) ./cmd/... && ./build/$(APP_NAME) run"

service: ## Create a new service skeleton. Usage: make service NAME=myservice
	@./$(SCRIPTS_DIR)/create_service.sh $(NAME)
	@$(MAKE) service-registration

service-remove: ## Remove a service. Usage: make service-remove NAME=myservice
	@./$(SCRIPTS_DIR)/remove_service.sh $(NAME)
	@$(MAKE) service-registration

service-registration: ## Regenerate service registration file
	@./$(SCRIPTS_DIR)/register_services.sh

servicepack-update: ## Update servicepack framework to latest version
	@./$(SCRIPTS_DIR)/servicepack_update.sh

own: ## Make this project your own. Usage: make own MODNAME=github.com/foo/bar
	@./$(SCRIPTS_DIR)/make_own.sh $(MODNAME)
	@$(MAKE) dep
	@git init

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
