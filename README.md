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
- Single binary, no external dependencies
- Low resource consumption

## Requirements

- Go 1.21 or later
- Linux operating system (requires procfs)
- Terminal with 256 color support (recommended)

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
make build          # Build for current platform
make build-linux    # Build for Linux
make install        # Install to GOPATH/bin
make help           # Show all available targets
```

**Windows (build.bat):**
```cmd
build.bat build     # Build for Windows
build.bat build-all # Build for all platforms
build.bat help      # Show all available targets
```

### System-wide Installation

```bash
sudo cp konrul /usr/local/bin/
```

## Usage

Simply run the binary:

```bash
./konrul
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
                                 +----------------------------------+
+-- Processes (Up/Down: scroll, q: quit, K: kill) -----------------+
| PID     USER     CPU%    MEM%    STATE   COMMAND                 |
| 1234    root     45.2    3.2     S       /usr/bin/python3...     |
| 5678    www      12.1    1.8     S       nginx: worker proc...   |
| 9012    mysql    8.5     5.1     S       /usr/sbin/mysqld        |
+------------------------------------------------------------------+
```

## Technical Details

### Data Sources

| File | Data | Usage |
|------|------|-------|
| /proc/stat | CPU timing | CPU percentage calculation |
| /proc/meminfo | Memory details | RAM and Swap info |
| /proc/[pid]/* | Process info | Process list |
| /proc/uptime | System uptime | Uptime display |
| /proc/loadavg | Load average | System load |

### Performance Targets

| Metric | Target |
|--------|--------|
| Startup time | < 100ms |
| CPU usage (idle) | < 1% |
| Memory usage | < 20 MB RSS |
| Refresh latency | < 50ms |

### Dependencies

- github.com/gizak/termui/v3 - Terminal UI framework

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

### Phase 1: MVP (Current)
- CPU, Memory, Swap monitoring
- Process list with kill feature
- Basic system information

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

### Phase 4: Multi-platform
- macOS support
- FreeBSD support
- Windows support (WMI)

## License

MIT License

## About the Name

Konrul (also known as Zumrudu Anka) is a mythological Turkish creature similar to the phoenix.
It is born from fire and rises from its ashes. Just like systems that need continuous 
monitoring and processes that need to be restarted...
