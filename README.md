# Konrul

Terminal Based System Monitor

Named after the mythological Turkish phoenix-like creature (Konrul/Zumrudu Anka), 
this lightweight htop-like system monitor is written in Go.

## Features

- Real-time CPU usage monitoring with history graph
- Memory and swap usage display
- Process list with sorting by CPU usage
- Process management (kill selected process)
- Responsive terminal UI
- Cross-platform support (Linux, macOS, Windows, FreeBSD)
- Single binary deployment
- Low resource consumption

## Supported Platforms

| Platform | Status | Notes |
|----------|--------|-------|
| Linux | Full support | All features available |
| macOS | Full support | All features available |
| Windows | Supported | Load average not available |
| FreeBSD | Full support | All features available |

## Requirements

- Go 1.21 or later
- Terminal with color support (recommended)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/user/konrul.git
cd konrul

# Download dependencies
go mod tidy

# Build
go build -o konrul

# Run
./konrul
```

### Using Build Scripts

**Linux/macOS (Makefile):**
```bash
make build              # Build for current platform
make build-linux        # Build for Linux (amd64)
make build-darwin       # Build for macOS (Intel)
make build-darwin-arm64 # Build for macOS (Apple Silicon)
make build-all-platforms # Build for all platforms
make install            # Install to GOPATH/bin
make help               # Show all available targets
```

**Windows (build.bat):**
```cmd
build.bat build     # Build for Windows
build.bat build-all # Build for all platforms
build.bat help      # Show all available targets
```

### Pre-built Binaries

Download from releases:
- konrul-linux-amd64
- konrul-linux-arm64
- konrul-darwin-amd64
- konrul-darwin-arm64
- konrul-windows-amd64.exe
- konrul-freebsd-amd64

### System-wide Installation

**Linux/macOS:**
```bash
sudo cp konrul /usr/local/bin/
```

**Windows:**
Copy konrul.exe to a directory in your PATH.

## Usage

Simply run the binary:

```bash
./konrul
```

On Windows:
```cmd
konrul.exe
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| q / Ctrl+C | Exit application |
| Up / k | Scroll up in process list |
| Down / j | Scroll down in process list |
| K / Delete | Kill selected process |
| Home | Jump to top of list |
| End | Jump to bottom of list |

## UI Layout

```
+-- CPU ---------------++-- Memory ------------++-- Swap --------------+
| [========  ] 78.5%   || [=====   ] 4.2G/8G   || [        ] 0/2G      |
+----------------------++----------------------++----------------------+
+-- CPU History ----------------++-- System -----------------------+
| ............................  || Hostname: server01              |
|                               || Uptime: 5d 12h 30m              |
+-------------------------------+| Load: 1.25 0.87 0.52            |
                                 | Processes: 142                  |
                                 | Platform: linux                 |
                                 +----------------------------------+
+-- Processes (Up/Down: scroll, q: quit, K: kill) -----------------+
| PID     USER     CPU%    MEM%    STATE   COMMAND                 |
| 1234    root     45.2    3.2     S       /usr/bin/python3...     |
| 5678    www      12.1    1.8     S       nginx: worker proc...   |
| 9012    mysql    8.5     5.1     S       /usr/sbin/mysqld        |
+------------------------------------------------------------------+
```

## Technical Details

### Dependencies

| Package | Purpose |
|---------|---------|
| github.com/gizak/termui/v3 | Terminal UI framework |
| github.com/shirou/gopsutil/v3 | Cross-platform system information |

### Performance Targets

| Metric | Target |
|--------|--------|
| Startup time | < 100ms |
| CPU usage (idle) | < 1% |
| Memory usage | < 20 MB RSS |
| Refresh latency | < 50ms |

### System Information Sources

gopsutil provides cross-platform abstractions for:
- CPU usage (cpu.Percent)
- Memory information (mem.VirtualMemory, mem.SwapMemory)
- Process information (process.Pids, process.NewProcess)
- Host information (host.Uptime)
- Load average (load.Avg) - Linux/macOS/FreeBSD only

## Project Structure

```
konrul/
├── main.go          # Main application and UI
├── go.mod           # Go module definition
├── go.sum           # Dependency checksums
├── Makefile         # Build automation (Linux/macOS)
├── build.bat        # Build automation (Windows)
└── README.md        # Documentation
```

## Build Information

Build with version information:

```bash
go build -ldflags "-s -w \
  -X main.version=$(git describe --tags --always) \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.date=$(date -u +%Y-%m-%d_%H:%M:%S)" \
  -o konrul
```

## Roadmap

### Phase 1: MVP (Complete)
- CPU, Memory, Swap monitoring
- Process list with kill feature
- Basic system information
- Cross-platform support

### Phase 2: Extended Features
- Per-core CPU usage
- Network I/O monitoring
- Disk I/O and usage
- Process tree view

### Phase 3: Advanced
- Process filtering and search
- Customizable layout
- GPU monitoring (NVIDIA/AMD)
- Docker container monitoring
- Configuration file support

## License

MIT License

## About the Name

Konrul (also known as Zumrudu Anka) is a mythological Turkish creature similar to the phoenix.
It is born from fire and rises from its ashes. Just like systems that need continuous 
monitoring and processes that need to be restarted...
