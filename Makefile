.PHONY: build install run clean test

BINARY := kestral
BUILD_DIR := .
INSTALL_DIR := $(HOME)/.local/bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/kestral

install: build
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(INSTALL_DIR)/$(BINARY)"

run: build
	./$(BINARY)

clean:
	rm -f $(BUILD_DIR)/$(BINARY)

test:
	go test ./...
