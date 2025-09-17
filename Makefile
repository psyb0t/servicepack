MIN_TEST_COVERAGE := 90
SCRIPTS_DIR := scripts
APP_NAME := $(shell head -n 1 go.mod | awk '{print $$2}' | awk -F'/' '{print $$NF}')

.PHONY: all dep lint lint-fix test test-coverage build \
	service service-remove service-registration servicepack-update own \
	backup backup-restore backup-clear

all: dep lint test ## Run dep, lint and test

dep: ## Get project dependencies
	@./$(SCRIPTS_DIR)/make/dep.sh

lint: ## Lint all Golang files
	@./$(SCRIPTS_DIR)/make/lint.sh

lint-fix: ## Lint all Golang files and fix
	@./$(SCRIPTS_DIR)/make/lint_fix.sh

test: ## Run all tests
	@./$(SCRIPTS_DIR)/make/test.sh

test-coverage: ## Run tests with coverage check. Fails if coverage is below the threshold.
	@./$(SCRIPTS_DIR)/make/test_coverage.sh

build: ## Build the app binary using Docker
	@./$(SCRIPTS_DIR)/make/build.sh

docker-build-dev: ## Build the development Docker image
	@./$(SCRIPTS_DIR)/make/docker_build_dev.sh

run-dev: ## Run in the development Docker image
	@./$(SCRIPTS_DIR)/make/run_dev.sh

service: ## Create a new service skeleton. Usage: make service NAME=myservice
	@./$(SCRIPTS_DIR)/make/service.sh $(NAME)

service-remove: ## Remove a service. Usage: make service-remove NAME=myservice
	@./$(SCRIPTS_DIR)/make/service_remove.sh $(NAME)

service-registration: ## Regenerate service registration file
	@./$(SCRIPTS_DIR)/make/service_registration.sh

servicepack-update: ## Update servicepack framework to latest version
	@./$(SCRIPTS_DIR)/make/servicepack_update.sh

own: ## Make this project your own. Usage: make own MODNAME=github.com/foo/bar
	@./$(SCRIPTS_DIR)/make/own.sh $(MODNAME)

backup: ## Create backup of the current project
	@./$(SCRIPTS_DIR)/make/backup.sh

backup-restore: ## Restore from backup. Usage: make backup-restore [BACKUP=filename.tar.gz] (defaults to latest)
	@./$(SCRIPTS_DIR)/make/backup_restore.sh $(BACKUP)

backup-clear: ## Delete all backup files
	@./$(SCRIPTS_DIR)/make/backup_clear.sh

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
