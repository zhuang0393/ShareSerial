.PHONY: build build-server build-client build-cli build-server-windows build-client-windows build-all-windows test test-unit test-e2e test-simulation clean fmt vet lint install run-server run-client package release install-systemd uninstall-systemd simulation-test

VERSION := 1.0.0
BUILD_DIR := bin
SCRIPTS_DIR := scripts
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Build targets (Linux)
build: build-server build-client build-cli

build-server:
	@echo "Building server (Linux)..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial-server ./cmd/server

build-client:
	@echo "Building client (Linux)..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial-client ./cmd/client

build-cli:
	@echo "Building CLI..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial ./cmd/cli

# Windows build targets
build-server-windows:
	@echo "Building Windows server..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial-server-windows.exe ./cmd/server-windows

build-client-windows:
	@echo "Building Windows client..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial-client-windows.exe ./cmd/client-windows

build-all-windows:
	@echo "Building all Windows binaries..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial-server-windows.exe ./cmd/server-windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial-client-windows.exe ./cmd/client-windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/shareserial-cli-windows.exe ./cmd/cli

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

# Full automated test suite (one-click testing)
automated-test:
	@echo "Running full automated test suite..."
	./scripts/automated-test.sh

# Quick automated test (unit + e2e only)
automated-test-quick:
	@echo "Running quick automated test..."
	./scripts/automated-test.sh --quick

# Full automated test (including long-run tests)
automated-test-full:
	@echo "Running full automated test with long-run tests..."
	./scripts/automated-test.sh --full

# Generate test report
generate-report:
	@echo "Generating test report..."
	go run ./cmd/report-generator .

# Build report generator
build-report-generator:
	@echo "Building report generator..."
	go build -o $(BUILD_DIR)/report-generator ./cmd/report-generator

# Complete test pipeline
test-pipeline: build automated-test generate-report
	@echo "=== Test pipeline complete ==="
	@echo "Check test-reports/ for detailed reports"

# Windows package
package-windows:
	@echo "Packaging Windows release..."
	mkdir -p release/windows
	cp $(BUILD_DIR)/shareserial-server-windows.exe release/windows/
	cp $(BUILD_DIR)/shareserial-client-windows.exe release/windows/
	cp configs/server-windows.yaml release/windows/
	cp configs/client.yaml release/windows/client-windows.yaml
	cp README.md release/windows/
	echo "ShareSerial Windows Version $(VERSION)" > release/windows/README.txt
	echo "" >> release/windows/README.txt
	echo "=== Server Usage ===" >> release/windows/README.txt
	echo "  shareserial-server-windows.exe --serial COM1 --port 7700" >> release/windows/README.txt
	echo "  shareserial-server-windows.exe --scan  (scan available COM ports)" >> release/windows/README.txt
	echo "" >> release/windows/README.txt
	echo "=== Client Usage ===" >> release/windows/README.txt
	echo "  shareserial-client-windows.exe --server IP:7700 --local-port 8888" >> release/windows/README.txt
	echo "" >> release/windows/README.txt
	echo "=== Connect with Putty ===" >> release/windows/README.txt
	echo "  Connection type: Raw" >> release/windows/README.txt
	echo "  Host: localhost" >> release/windows/README.txt
	echo "  Port: 8888" >> release/windows/README.txt
	@echo "Windows package created in release/windows/"

# Full Windows release (server + client)
release-windows: clean build-all-windows package-windows
	@echo "=== Windows release complete ==="
	@ls -la release/windows/

# Windows server only
release-windows-server: clean build-server-windows
	@echo "=== Windows server release complete ==="
	@ls -la $(BUILD_DIR)/shareserial-server-windows.exe