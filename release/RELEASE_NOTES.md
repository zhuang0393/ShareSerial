# ShareSerial Release Notes

## Version 1.0.0 (2026-06-03)

### 🎉 First Stable Release

ShareSerial is a cross-platform serial port sharing system that allows multiple engineers to access a physical serial port simultaneously over network.

### ✨ Features

- **Server Mode**: Share physical serial port over network
- **Client Mode**: Create virtual serial port on remote machines
- **CLI Tool**: AI-friendly command interface for log/status/send operations
- **Write Lock Arbitration**: Exclusive mode prevents concurrent input conflicts
- **Auto Reconnection**: Client automatically reconnects when server is available
- **High Baud Rate Support**: Up to 1500000 baud rate
- **Low Latency**: < 5ms transmission delay

### 📦 Supported Platforms

| Platform | Architecture | Files |
|----------|--------------|-------|
| Linux | x86_64 | shareserial-server, shareserial-client, shareserial |
| Windows | x86_64 | shareserial-server-windows.exe, shareserial-client-windows.exe, shareserial-cli-windows.exe |
| macOS | Intel (x86_64) | shareserial-server-darwin-amd64, shareserial-client-darwin-amd64, shareserial-darwin |
| macOS | Apple Silicon (ARM64) | shareserial-server-darwin-arm64, shareserial-client-darwin-arm64, shareserial-darwin |

### 🚀 Quick Start

#### Linux Server
```bash
./shareserial-server --serial /dev/ttyUSB0 --port 7700
```

#### Linux Client
```bash
./shareserial-client --server IP:7700 --pty /dev/ttyShare
```

#### Windows Server
```cmd
shareserial-server-windows.exe --serial COM1 --port 7700
```

#### Windows Client
```cmd
shareserial-client-windows.exe --server IP:7700 --local-port 8888
```

#### CLI Tool (AI Interface)
```bash
# Get log data
./shareserial log --server IP:7700

# Check connection status
./shareserial status --server IP:7700

# Send command (with automatic write lock)
./shareserial send --command "ls -la" --server IP:7700
```

### 🔧 Requirements

- **Linux**: Ubuntu 18.04+ or compatible distro
- **Windows**: Windows 10/11, no additional dependencies
- **macOS**: macOS 10.15+ (Intel or Apple Silicon)

### 📋 Known Limitations

- Phase 1: No encryption/authentication (for trusted network only)
- Phase 1: Fixed baud rate 115200 (configurable in Phase 2)
- Phase 1: Manual server IP configuration (mDNS in Phase 2)

### 🐛 Bug Fixes

- Fixed all golangci-lint warnings for clean build
- Fixed nil pointer dereference in config loader
- Fixed closure capturing loop variable issue
- Added proper error handling for all I/O operations

### 📝 License

MIT License - Open Source Project

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0 | 2026-06-03 | First stable release |