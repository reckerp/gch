.PHONY: build install deps

BINARY_NAME=gch
BUILD_DIR=./build
INSTALL_DIR=$(HOME)/bin

build: deps
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) -v

deps:
	@echo "Downloading dependencies..."
	@go mod tidy
	@go mod download

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@rm -rf $(BUILD_DIR)
	@echo "Installation complete. Make sure $(INSTALL_DIR) is in your PATH."

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean
