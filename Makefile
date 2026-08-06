# Project Makefile
# Add your custom targets here - they will override servicepack defaults

# Override framework variables (optional)
# MIN_TEST_COVERAGE := 95

# Include servicepack framework commands
include Makefile.servicepack

# Custom targets below this line
# Note: Override warnings are expected and can be ignored

# Example: override a framework command by uncommenting and editing this.
#
# Left COMMENTED on purpose. As a live target it shadowed the framework's real
# `build`, so `make build` printed a line and produced no binary — which broke
# the README's own Quick Start (`make own` → `make build` → ./build/<name> run)
# for everyone who followed it.
#
# build: ## Custom build command
# 	@echo "Running custom build..."

# Add your custom targets below this line
