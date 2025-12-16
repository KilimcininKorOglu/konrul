# 🔥 Konrul

**Terminal Based System Monitor**

Named after the mythological Turkish phoenix-like creature (Konrul/Zümrüdü Anka), 
this lightweight htop-like system monitor is written in Go.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Platform-Linux%20|%20macOS%20|%20Windows%20|%20FreeBSD-blue" alt="Platform">
  <img src="https://img.shields.io/badge/License-MIT-green" alt="License">
</p>

---

## ✨ Features

| Category | Features |
|:---------|:---------|
| **📊 System Monitoring** | CPU (total + per-core), Memory, Swap |
| **🌐 Network** | RX/TX bytes, per-second rates |
| **💾 Disk** | Read/Write I/O, usage percentage |
| **🎮 GPU** | NVIDIA (nvidia-smi), AMD (rocm-smi) |
| **🐳 Docker** | Container stats, ports, IP addresses |
| **📋 Processes** | List, sort, search, tree view, kill |
| **🎨 Themes** | 4 built-in color schemes |
| **⚙️ Config** | YAML configuration file |

---

## 🆚 Konrul vs htop

| Feature | Konrul | htop |
|:--------|:------:|:----:|
| **Windows Support** | ✅ Native | ❌ |
| **Network I/O** | ✅ Built-in | ❌ |
| **Disk I/O** | ✅ Built-in | ❌ |
| **GPU Monitoring** | ✅ NVIDIA + AMD | ❌ |
| **Docker Containers** | ✅ Stats + Ports | ❌ |
| **Single Binary** | ✅ Zero deps | ❌ ncurses |
| **Cross-platform** | ✅ 4 platforms | ⚠️ 3 platforms |
| **Per-core CPU** | ✅ | ✅ |
| **Process Tree** | ✅ | ✅ |
| **Process Search** | ✅ | ✅ |
| **Themes** | ✅ 4 themes | ✅ |
| **Config File** | ✅ YAML | ✅ htoprc |
| **Memory** | ~15 MB | ~5 MB |
| **Language** | Go | C |

---

## 🖥️ Supported Platforms

| Platform | Architecture | Status | Notes |
|:---------|:-------------|:------:|:------|
| 🐧 **Linux** | amd64, arm64, arm | ✅ | Full support |
| 🍎 **macOS** | amd64 (Intel), arm64 (Apple Silicon) | ✅ | Full support |
| 🪟 **Windows** | amd64, arm64 | ✅ | CPU Queue Length instead of load avg |
| 😈 **FreeBSD** | amd64 | ✅ | Full support |

---

## 📦 Installation

### Quick Install (Go)

```bash
go install github.com/user/konrul@latest
```

### From Source

```bash
git clone https://github.com/user/konrul.git
cd konrul
go mod tidy
go build -o konrul
./konrul
```

### Build Scripts

<table>
<tr>
<td width="50%">

**🐧 Linux / 🍎 macOS (Makefile)**
```bash
make build        # Current platform
make build-all    # All platforms
make install      # Install to GOPATH
```

</td>
<td width="50%">

**🪟 Windows (build.bat)**
```cmd
build.bat build     # Windows only
build.bat build-all # All platforms
```

</td>
</tr>
</table>

### Pre-built Binaries

| Platform | Binary |
|:---------|:-------|
| Linux (amd64) | `konrul-linux-amd64` |
| Linux (arm64) | `konrul-linux-arm64` |
| macOS (Intel) | `konrul-darwin-amd64` |
| macOS (Apple Silicon) | `konrul-darwin-arm64` |
| Windows | `konrul-windows-amd64.exe` |
| FreeBSD | `konrul-freebsd-amd64` |

---

## ⌨️ Keyboard Controls

### Function Keys

| Key | Action |
|:---:|:-------|
| `F1` | Show help screen |
| `F8` | Cycle sort mode |
| `F9` | Kill process / Stop container |
| `F10` | Quit application |

### Navigation

| Key | Action |
|:---:|:-------|
| `↑` `k` | Scroll up |
| `↓` `j` | Scroll down |
| `Home` | Jump to top |
| `End` | Jump to bottom |

### Sorting

| Key | Normal View | GPU View |
|:---:|:------------|:---------|
| `c` | Sort by CPU% | - |
| `m` | Sort by MEM% | - |
| `p` | Sort by PID | Sort by PID |
| `n` | Sort by NAME | Sort by NAME |
| `r` | Reverse order | Reverse order |
| `g` | - | Sort by GPU% |
| `G` | - | Sort by GPU Memory |

### Views & Modes

| Key | Action |
|:---:|:-------|
| `d` | Toggle view: **Normal** → **GPU** → **Docker** |
| `t` | Toggle tree view (Normal view only) |
| `T` | Cycle themes |
| `/` | Search/filter processes |
| `Esc` | Clear search filter |
| `?` `h` | Show help |
| `q` `Ctrl+C` | Quit |

### Process Management

| Key | Normal/GPU View | Docker View |
|:---:|:----------------|:------------|
| `K` `Delete` `F9` | Kill process | Stop container |

---

## 🎯 View Modes

### 📊 Normal View (Default)
Standard process list with CPU, Memory, State information.

```
┌─ Processes [Sort:CPU%] ─────────────────────────────────┐
│ PID    USER    CPU%   MEM%   STATE   COMMAND            │
│ 1234   root    45.2   3.2    R       python train.py    │
│ 5678   www     12.1   1.8    S       nginx: worker      │
└─────────────────────────────────────────────────────────┘
```

### 🎮 GPU View (Press `d`)
Shows only GPU-using processes with GPU metrics.

```
┌─ GPU Processes [Sort:GPU_MEM] ──────────────────────────┐
│ PID    USER    GPU%   GPU_MEM   TYPE   COMMAND          │
│ 1234   root    45%    2.1 GB    C      python train.py  │
│ 5678   user    12%    512 MB    G      blender          │
└─────────────────────────────────────────────────────────┘
```

### 🐳 Docker View (Press `d` again)
Shows Docker containers with ports and IP addresses.

```
┌─ Docker Containers [Sort:CPU%] ─────────────────────────┐
│ CONTAINER  IMAGE         CPU%   MEM     IP          PORTS       │
│ web-app    nginx:latest  2.5%   45 MB   172.17.0.2  8080:80     │
│ database   postgres:15   5.1%   256 MB  172.17.0.3  5432:5432   │
└─────────────────────────────────────────────────────────┘
```

---

## 🖼️ UI Layout

```
┌─ CPU ──────────────┐┌─ Memory ───────────┐┌─ Swap ─────────────┐
│ [████████░░] 78.5% ││ [█████░░░] 4.2G/8G ││ [░░░░░░░░] 0/2G    │
└────────────────────┘└────────────────────┘└────────────────────┘
┌─ CPU Cores ─────────────┐┌─ Net ───┐┌─ Disk ──┐┌─ GPU ───┐┌─ System ─┐
│ 0██ 1████ 2███ 3█       ││ RX:1.2G ││ 65%     ││ [NV]    ││ Up: 5d   │
│ 4████ 5██ 6█████ 7██    ││ TX:856M ││ R:12M/s ││ 45%     ││ Load:1.2 │
└─────────────────────────┘└─────────┘└─────────┘└─────────┘└──────────┘
┌─ Processes [Sort:CPU%] ─────────────────────────────────────────────┐
│ PID     USER     CPU%    MEM%    STATE   COMMAND                    │
│ 1234    root     45.2    3.2     S       python train.py            │
│ 5678    www      12.1    1.8     S       nginx: worker process      │
└─────────────────────────────────────────────────────────────────────┘
 F1:Help F8:Sort F9:Kill F10:Quit | /:Search t:Tree d:GPU/Docker
```

---

## ⚙️ Command-line Arguments

```bash
konrul [options]
```

| Option | Short | Description | Default |
|:-------|:-----:|:------------|:--------|
| `--version` | `-v` | Show version information | - |
| `--interval` | `-i` | Refresh interval (1-10 sec) | 1 |
| `--sort` | `-s` | Default sort (cpu/mem/pid/name) | cpu |
| `--tree` | `-t` | Start in tree view | false |
| `--config` | `-c` | Config file path | auto |

**Examples:**
```bash
konrul --version              # Show version
konrul -i 2                   # 2-second refresh
konrul --tree --sort mem      # Tree view, sort by memory
konrul -c ~/.konrul.yaml      # Custom config file
```

---

## 📁 Configuration

Config file locations:
- **Linux/macOS:** `~/.config/konrul/config.yaml`
- **Windows:** `%APPDATA%\konrul\config.yaml`

```yaml
# Example config.yaml
refresh_interval: 1
default_sort: cpu
tree_view: false
theme: default  # default, dark, light, monokai
```

---

## 🎨 Themes

| Theme | Description |
|:------|:------------|
| `default` | Blue/Green on dark background |
| `dark` | Muted colors, easy on eyes |
| `light` | For light terminal backgrounds |
| `monokai` | Monokai-inspired colors |

Press `T` to cycle through themes at runtime.

---

## 📊 Technical Details

### Dependencies

| Package | Purpose |
|:--------|:--------|
| `github.com/gizak/termui/v3` | Terminal UI framework |
| `github.com/shirou/gopsutil/v3` | Cross-platform system info |
| `gopkg.in/yaml.v3` | YAML config parsing |

### Performance

| Metric | Target | Actual |
|:-------|:------:|:------:|
| Startup time | < 100ms | ✅ |
| CPU usage (idle) | < 1% | ✅ |
| Memory usage | < 20 MB | ✅ ~15 MB |
| Refresh latency | < 50ms | ✅ |

---

## 🗺️ Roadmap

| Phase | Status | Features |
|:------|:------:|:---------|
| **Phase 1: MVP** | ✅ | CPU, Memory, Swap, Process list |
| **Phase 2: Extended** | ✅ | Per-core CPU, Network I/O, Disk I/O, Sorting, Tree view |
| **Phase 3: Advanced** | ✅ | Search, Help, Config, Themes, CLI args |
| **Phase 4: Monitoring** | ✅ | GPU (NVIDIA/AMD), Docker containers |
| **Phase 5: Views** | ✅ | Context-aware process list (GPU/Docker views) |
| **Future** | 🔄 | Kubernetes pods, Plugin system |

---

## 📜 License

MIT License

---

## 🦅 About the Name

**Konrul** (also known as **Zümrüdü Anka**) is a mythological Turkish creature similar to the phoenix. It is born from fire and rises from its ashes.

Just like systems that need continuous monitoring and processes that need to be restarted... 🔥

---

<p align="center">
  Made with ❤️ in Go
</p>
