# Konrul

Terminal Based System Monitor

Named after the mythological Turkish phoenix-like creature (Konrul/Zumrudu Anka), 
this lightweight htop-like system monitor is written in Go.

## Features

- Real-time CPU usage monitoring (total and per-core)
- Memory and swap usage display
- Network I/O monitoring (RX/TX with per-second rates)
- Disk I/O and usage monitoring
- GPU monitoring (NVIDIA via nvidia-smi)
- Docker container monitoring
- Process list with multiple sorting options (CPU, MEM, PID, NAME)
- Process management (kill selected process)
- Responsive terminal UI
- Cross-platform support (Linux, macOS, Windows, FreeBSD)
- Single binary deployment
- Low resource consumption

## Konrul vs htop

| Feature          | Konrul                          | htop                        |
|:-----------------|:--------------------------------|:----------------------------|
| Windows Support  | ✅ Native support               | ❌ Not available            |
| Network I/O      | ✅ Built-in panel               | ❌ Not available            |
| Disk I/O         | ✅ Built-in panel               | ❌ Not available            |
| GPU Monitoring   | ✅ NVIDIA (nvidia-smi)          | ❌ Not available            |
| Docker Monitor   | ✅ Container stats              | ❌ Not available            |
| Single Binary    | ✅ Zero dependencies            | ❌ Requires ncurses         |
| Cross-platform   | ✅ Linux, macOS, Win, FreeBSD   | ⚠️ Linux, macOS, FreeBSD    |
| Per-core CPU     | ✅ Bar chart                    | ✅ Bar graph                |
| Process Tree     | ✅ Toggle with `t`              | ✅ Toggle with `t`          |
| Process Search   | ✅ Real-time filter             | ✅ Incremental search       |
| Kill Process     | ✅ `K` or `Delete`              | ✅ `F9`                     |
| Themes           | ✅ 4 built-in                   | ✅ Color schemes            |
| Config File      | ✅ YAML                         | ✅ htoprc                   |
| Memory Usage     | ~15 MB                          | ~5 MB                       |
| Language         | Go                              | C                           |
| Installation     | Single binary / `go install`    | Package manager             |

## Supported Platforms

| Platform | Status | Notes |
|----------|--------|-------|
| Linux | Full support | All features available |
| macOS | Full support | All features available |
| Windows | Full support | CPU Queue Length instead of load average |
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
| **Function Keys** | |
| F1 | Show help screen |
| F8 | Cycle sort mode (CPU→MEM→PID→NAME) |
| F9 | Kill selected process |
| F10 | Quit application |
| **Navigation** | |
| Up / k | Scroll up in process list |
| Down / j | Scroll down in process list |
| Home | Jump to top of list |
| End | Jump to bottom of list |
| **Sorting** | |
| c | Sort by CPU usage |
| m | Sort by Memory usage |
| p | Sort by PID |
| n | Sort by Name |
| r | Reverse sort order |
| **Views** | |
| t | Toggle tree view |
| T | Cycle themes (default, dark, light, monokai) |
| d | Toggle GPU/Docker panel |
| / | Search/filter processes |
| Esc | Clear search filter |
| **Other** | |
| K / Delete | Kill selected process |
| ? / h | Show help screen |
| q / Ctrl+C | Exit application |

### Command-line Arguments

```bash
konrul [options]

Options:
  -v, --version     Show version information
  -i, --interval N  Refresh interval in seconds (1-10, default: 1)
  -s, --sort MODE   Default sort mode (cpu, mem, pid, name)
  -t, --tree        Start in tree view mode
  -c, --config FILE Path to config file
```

Examples:
```bash
# Show version
konrul --version

# Start with 2-second refresh interval
konrul -i 2

# Start in tree view, sorted by memory
konrul --tree --sort mem
```

## UI Layout

```
+-- CPU ---------------++-- Memory ------------++-- Swap --------------+
| [========  ] 78.5%   || [=====   ] 4.2G/8G   || [        ] 0/2G      |
+----------------------++----------------------++----------------------+
+-- CPU Cores -------++-- Network -++-- Disk ----++-- System ---+
| 0: [===    ] 45%   || RX: 1.2G   || Used: 65%  || Up: 5d12h   |
| 1: [======  ] 72%  || TX: 856M   || R: 12M/s   || Load: 1.25  |
| 2: [====    ] 55%  || RX/s: 125K || W: 8M/s    || Procs: 142  |
| 3: [=       ] 12%  || TX/s: 45K  |+------------++-------------+
+--------------------++------------+
+-- Processes [c:CPU m:MEM p:PID n:NAME r:Rev] Sort:CPU% -------------+
| PID     USER     CPU%    MEM%    STATE   COMMAND                    |
| 1234    root     45.2    3.2     S       /usr/bin/python3...        |
| 5678    www      12.1    1.8     S       nginx: worker proc...      |
| 9012    mysql    8.5     5.1     S       /usr/sbin/mysqld           |
+---------------------------------------------------------------------+
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
- CPU usage (cpu.Percent) - total and per-core
- Memory information (mem.VirtualMemory, mem.SwapMemory)
- Network I/O (net.IOCounters) - bytes sent/received
- Disk I/O (disk.IOCounters, disk.Usage) - read/write and usage
- Process information (process.Pids, process.NewProcess)
- Host information (host.Uptime)
- Load average (load.Avg) - Linux/macOS/FreeBSD
- CPU Queue Length (PDH) - Windows

## Project Structure

```
konrul/
├── main.go          # Main application and UI
├── load_unix.go     # Load average for Unix systems
├── load_windows.go  # CPU Queue Length for Windows
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
- [x] CPU, Memory, Swap monitoring
- [x] Process list with kill feature
- [x] Basic system information
- [x] Cross-platform support

### Phase 2: Extended Features (Complete)
- [x] Per-core CPU usage display
- [x] Network I/O monitoring (RX/TX rates)
- [x] Disk I/O and usage monitoring
- [x] Process sorting options (CPU/MEM/PID/NAME)
- [x] Process tree view

### Phase 3: Advanced (Complete)
- [x] Process filtering and search
- [x] Help screen (keyboard shortcuts)
- [x] Command-line arguments
- [x] Configurable refresh interval
- [x] Configuration file support (YAML)
- [x] Theme support (4 built-in themes)

### Phase 4: Extended Monitoring (Complete)
- [x] GPU monitoring (NVIDIA via nvidia-smi)
- [x] Docker container monitoring

### Future Plans
- [ ] AMD GPU monitoring (rocm-smi)
- [ ] Kubernetes pod monitoring
- [ ] Custom plugin system

## License

MIT License

## About the Name

Konrul (also known as Zumrudu Anka) is a mythological Turkish creature similar to the phoenix.
It is born from fire and rises from its ashes. Just like systems that need continuous 
monitoring and processes that need to be restarted...
