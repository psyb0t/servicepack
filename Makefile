# Project Makefile
# Add your custom targets here - they will override servicepack defaults

# Override framework variables (optional)
# MIN_TEST_COVERAGE := 95

# Include servicepack framework commands
include Makefile.servicepack

# Custom targets below this line
# Note: Override warnings are expected and can be ignored

# Example: Override framework build command
build: ## Custom build command
	@echo "Running custom build..."

# Add your custom targets below this line
