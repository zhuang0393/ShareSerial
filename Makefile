.PHONY: build build-server build-client build-cli test test-unit test-e2e test-simulation clean fmt vet lint install run-server run-client package release install-systemd uninstall-systemd simulation-test

VERSION := 1.0.0
BUILD_DIR := bin
SCRIPTS_DIR := scripts
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Build targets
build: build-server build-client build-cli

build-server:
	@echo "Building server..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial-server ./cmd/server

build-client:
	@echo "Building client..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial-client ./cmd/client

build-cli:
	@echo "Building CLI..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial ./cmd/cli

# Test targets
test: test-unit test-e2e

test-unit:
	@echo "Running unit tests..."
	go test -v ./pkg/... ./internal/... ./cmd/cli/...

test-e2e:
	@echo "Running E2E tests..."
	go test -v ./tests/e2e/...

# Code quality
fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

lint:
	@echo "Running golint..."
	@which golint > /dev/null || go install golang.org/x/lint/golint@latest
	golint ./...

# Clean
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)/
	rm -rf release/
	go clean

# Development
dev-server:
	go run ./cmd/server

dev-client:
	go run ./cmd/client

# Run scripts
run-server:
	./$(SCRIPTS_DIR)/start-server.sh

run-client:
	./$(SCRIPTS_DIR)/start-client.sh

# Install dependencies
deps:
	go mod download
	go mod tidy

# Package for release
package:
	@echo "Packaging for release..."
	mkdir -p release
	cp $(BUILD_DIR)/shareserial-server release/
	cp $(BUILD_DIR)/shareserial-client release/
	cp $(BUILD_DIR)/shareserial release/
	cp README.md release/
	cp DEPLOY.md release/
	cp -r configs release/
	cp -r scripts release/
	tar -czvf shareserial-$(VERSION).tar.gz release/
	rm -rf release/
	@echo "Package created: shareserial-$(VERSION).tar.gz"

# Full release build
release: clean build test package
	@echo "=== Release complete ==="
	@ls -la shareserial-$(VERSION).tar.gz

# Install to system
install:
	@echo "Installing to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/shareserial-server /usr/local/bin/
	sudo cp $(BUILD_DIR)/shareserial-client /usr/local/bin/
	sudo cp $(BUILD_DIR)/shareserial /usr/local/bin/
	sudo mkdir -p /etc/shareserial
	sudo cp configs/server.yaml /etc/shareserial/
	sudo cp configs/client.yaml /etc/shareserial/
	@echo "Installed:"
	@echo "  /usr/local/bin/shareserial-server"
	@echo "  /usr/local/bin/shareserial-client"
	@echo "  /usr/local/bin/shareserial"
	@echo "  /etc/shareserial/"

# Install systemd service
install-systemd: install
	@echo "Installing systemd service..."
	sudo cp $(SCRIPTS_DIR)/shareserial-server.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable shareserial-server
	@echo "Systemd service installed."
	@echo "Start with: sudo systemctl start shareserial-server"
	@echo "Status with: sudo systemctl status shareserial-server"

# Uninstall systemd service
uninstall-systemd:
	@echo "Uninstalling systemd service..."
	sudo systemctl stop shareserial-server 2>/dev/null || true
	sudo systemctl disable shareserial-server 2>/dev/null || true
	sudo rm -f /etc/systemd/system/shareserial-server.service
	sudo systemctl daemon-reload
	@echo "Systemd service uninstalled."

# Uninstall from system
uninstall: uninstall-systemd
	@echo "Uninstalling from /usr/local/bin..."
	sudo rm -f /usr/local/bin/shareserial-server
	sudo rm -f /usr/local/bin/shareserial-client
	sudo rm -f /usr/local/bin/shareserial
	sudo rm -rf /etc/shareserial
	@echo "Uninstalled."

# Stability test
stability-test:
	@echo "Running stability test..."
	./scripts/stability-test.sh

# Verify serial port
verify-serial:
	@echo "Verifying serial port..."
	./scripts/verify-serial.sh

# Quick test
quick-test:
	@echo "Running quick test..."
	./scripts/deploy.sh test

# Simulation test (automated virtual serial environment)
simulation-test:
	@echo "Running simulation tests..."
	go test -v ./tests/simulation/...

# Simulation test with short mode
simulation-test-short:
	@echo "Running simulation tests (short mode)..."
	go test -v -short ./tests/simulation/...