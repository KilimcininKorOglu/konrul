# Konrul - Is Plani

Bu dokuman, PRD'de belirtilen eksik ozelliklerin eklenmesi icin detayli is planini icerir.
Her adim bir commit ile tamamlanacaktir.

---

## Mevcut Durum

### Tamamlanan Ozellikler (Faz 1-4)
- [x] CPU monitoring (toplam, gauge widget)
- [x] Per-core CPU monitoring (bar chart)
- [x] Memory monitoring (used/total, yuzde)
- [x] Swap monitoring (No Swap destegi)
- [x] Network I/O monitoring (RX/TX, per-second rates)
- [x] Disk I/O monitoring (read/write rates, usage %)
- [x] Process listesi (PID, USER, CPU%, MEM%, STATE, COMMAND)
- [x] Process sorting (CPU, MEM, PID, NAME, reverse)
- [x] Process tree view (t tusu)
- [x] Process search/filter (/ tusu)
- [x] Process kill (K/Delete/F9 tuslari)
- [x] Sistem bilgisi (uptime, load, process count)
- [x] Klavye kontrolleri (q, j/k, Up/Down, Home, End, c/m/p/n/r, F1/F8/F9/F10)
- [x] Responsive UI (terminal resize destegi)
- [x] Cross-platform (Linux, macOS, Windows, FreeBSD)
- [x] Dinamik kolon genislikleri
- [x] Windows CPU Queue Length
- [x] Configuration file support (YAML)
- [x] Theme support (4 built-in themes)
- [x] Command-line arguments (-v, -i, -s, -t, -c)
- [x] Help screen (? veya h veya F1)
- [x] GPU monitoring (NVIDIA nvidia-smi, AMD rocm-smi)
- [x] Docker container monitoring (docker CLI)
- [x] Status bar (htop-style F-key shortcuts)
- [x] GPU/Docker panel toggle (d tusu)

---

## Faz 5: Kontekst-Duyarli Process Listesi (Yeni Ozellik)

Bu faz, GPU veya Docker gorunumundeyken process listesinin ilgili kaynaklara gore 
filtrelenmesini ve ozel kolonlarla gosterilmesini icermektedir.

### Genel Bakis

**Fikir:** 
- GPU paneli aktifken → Process listesinde sadece GPU kullanan process'ler
- Docker paneli aktifken → Process listesinde container'lar (veya container process'leri)

**Faydalar:**
- Daha odakli izleme
- GPU/Docker kaynak kullanimini daha iyi anlama
- htop'tan farklilasmis, benzersiz ozellik

---

### Adim 5.1: GPU Process Listesi

**Commit:** "Add GPU process filtering when GPU panel is active"

#### Arastirma Sonuclari

**NVIDIA GPU (nvidia-smi):**
```bash
# GPU kullanan process'leri listele
nvidia-smi --query-compute-apps=pid,process_name,used_memory --format=csv,noheader,nounits

# Ornek cikti:
# 1234, python, 2048
# 5678, chrome, 512

# Daha detayli (GPU kullanim yuzdesi dahil)
nvidia-smi pmon -c 1 -s um
# pid, type, sm%, mem%, enc, dec, command
```

**AMD GPU (rocm-smi):**
```bash
# AMD GPU process listesi
rocm-smi --showpidgpus
# veya
rocm-smi --showpids

# Ornek cikti:
# GPU[0] : PID 1234 is using 2048 MB
```

**Windows NVIDIA:**
- Ayni nvidia-smi komutlari Windows'ta da calisir
- CUDA yuklu olmali

#### Yapilacaklar

1. **gpu.go'ya yeni fonksiyon ekle:**
```go
type GPUProcess struct {
    PID         int32
    Name        string
    GPUMemory   uint64  // bytes
    GPUPercent  float64 // SM utilization %
    Type        string  // C=Compute, G=Graphics
}

func GetGPUProcesses() []GPUProcess {
    // NVIDIA icin nvidia-smi --query-compute-apps
    // AMD icin rocm-smi --showpidgpus
}
```

2. **main.go'da view mode ekle:**
```go
type ViewMode int
const (
    ViewModeNormal ViewMode = iota  // Tum process'ler
    ViewModeGPU                      // Sadece GPU process'leri
    ViewModeDocker                   // Sadece Docker container'lar
)
```

3. **Process tablosu kolonlarini degistir:**
   - Normal: PID, USER, CPU%, MEM%, STATE, COMMAND
   - GPU: PID, USER, GPU%, GPU_MEM, STATE, COMMAND
   - Docker: CONTAINER, IMAGE, CPU%, MEM%, STATUS

4. **Filtreleme mantigi:**
```go
func filterByGPU(processes []Process, gpuProcs []GPUProcess) []Process {
    gpuPIDs := make(map[int32]GPUProcess)
    for _, gp := range gpuProcs {
        gpuPIDs[gp.PID] = gp
    }
    
    var filtered []Process
    for _, p := range processes {
        if gp, ok := gpuPIDs[p.PID]; ok {
            p.GPUPercent = gp.GPUPercent
            p.GPUMemory = gp.GPUMemory
            filtered = append(filtered, p)
        }
    }
    return filtered
}
```

#### Test Senaryolari
- [ ] NVIDIA GPU'lu sistemde GPU process'lerin dogru listelenmesi
- [ ] AMD GPU'lu sistemde GPU process'lerin dogru listelenmesi
- [ ] GPU olmayan sistemde bos liste veya uyari mesaji
- [ ] GPU panelinden normal gorunume donuste tum process'lerin geri gelmesi

---

### Adim 5.2: Docker Container Listesi

**Commit:** "Add Docker container view when Docker panel is active"

#### Arastirma Sonuclari

**Docker CLI ile Container Listesi:**
```bash
# Container listesi (JSON)
docker ps -a --format '{{json .}}'

# Container stats (CPU, MEM - JSON)
docker stats --no-stream --format '{{json .}}'

# Ornek JSON:
# {"ID":"abc123","Names":"web-app","Image":"nginx:latest","Status":"Up 2h","CPUPerc":"2.5%","MemUsage":"45MiB / 2GiB"}
```

**Container Icindeki Process'ler:**
```bash
# Container'in ana PID'i (host namespace)
docker inspect --format '{{.State.Pid}}' <container_id>

# Container icindeki tum process'ler
docker top <container_id>
```

**Process'in Hangi Container'a Ait Oldugunu Bulma (Linux):**
```bash
# /proc/<pid>/cgroup dosyasindan container ID
cat /proc/1234/cgroup
# Cikti: 0::/docker/<container_id>
```

#### Iki Yaklasim Secenegi

**Secenek A: Container-Centric View (Onerilen)**
- Process listesi yerine container listesi goster
- Her satir bir container
- Kolonlar: CONTAINER, IMAGE, CPU%, MEM%, STATUS

**Secenek B: Process-Centric View**
- Mevcut process listesini filtrele
- Sadece container icindeki process'leri goster
- Her process'in hangi container'a ait oldugunu goster

#### Yapilacaklar (Secenek A)

1. **docker.go'yu genislet:**
```go
type DockerContainerView struct {
    ID        string
    Name      string
    Image     string
    Status    string
    State     string
    CPUPerc   float64
    MemUsage  string
    MemPerc   float64
    Ports     string   // "8080:80, 443:443"
    HostPorts []string // ["8080", "443"]
    ContPorts []string // ["80", "443"]
    IPAddress string   // Container IP (172.17.0.2)
    Networks  string   // Network adlari (bridge, custom_net)
}

func GetDockerContainersDetailed() []DockerContainerView {
    // docker ps + docker stats + docker inspect birlestir
}
```

**Port ve IP bilgisi alma komutlari:**
```bash
# Port bilgisi (docker ps ile)
docker ps --format '{{.Ports}}'
# Ornek cikti: 0.0.0.0:8080->80/tcp, :::443->443/tcp

# Container IP (docker inspect ile)
docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' <container>
# Ornek cikti: 172.17.0.2

# Tum bilgiler tek seferde (JSON)
docker inspect --format '{{json .NetworkSettings}}' <container>
```

2. **Yeni tablo formati (Port ve IP ile):**
```
┌──────────────────────────────────────────────────────────────────────────────────┐
│ Docker Containers [d:toggle] Sort:CPU                                            │
├────────────┬───────────────┬───────┬────────┬──────────────┬─────────────────────┤
│ CONTAINER  │ IMAGE         │ CPU%  │ MEM    │ IP           │ PORTS               │
├────────────┼───────────────┼───────┼────────┼──────────────┼─────────────────────┤
│ web-app    │ nginx:latest  │ 2.5%  │ 45 MB  │ 172.17.0.2   │ 8080:80, 443:443    │
│ database   │ postgres:15   │ 5.1%  │ 256 MB │ 172.17.0.3   │ 5432:5432           │
│ redis      │ redis:7       │ 0.3%  │ 12 MB  │ 172.17.0.4   │ 6379:6379           │
│ api-server │ node:18       │ 3.2%  │ 128 MB │ 172.18.0.2   │ 3000:3000, 9229:9229│
└────────────┴───────────────┴───────┴────────┴──────────────┴─────────────────────┘
```

**Alternatif kompakt format (dar terminaller icin):**
```
┌─────────────────────────────────────────────────────────────────┐
│ Docker Containers Sort:CPU                                      │
├────────────┬───────────────┬───────┬────────┬──────────────────┤
│ CONTAINER  │ IMAGE         │ CPU%  │ MEM    │ PORTS            │
├────────────┼───────────────┼───────┼────────┼──────────────────┤
│ web-app    │ nginx         │ 2.5%  │ 45 MB  │ :8080→80         │
│ database   │ postgres      │ 5.1%  │ 256 MB │ :5432→5432       │
└────────────┴───────────────┴───────┴────────┴──────────────────┘
```
- IP bilgisi STATUS yerine PORTS kolonunda hover/detay olarak
- Port formati: `:host→container` (kisaltilmis)

3. **Sorting secenekleri (Docker view):**
   - CPU%: Container CPU kullanimi
   - MEM: Container memory kullanimi
   - NAME: Container adi
   - STATUS: Container durumu

4. **Container islemleri:**
   - Kill (F9/K): `docker stop <container>`
   - Enter: Container detaylari (opsiyonel)

#### Test Senaryolari
- [ ] Docker kurulu sistemde container listesinin dogru gosterilmesi
- [ ] Container CPU/MEM degerlerinin guncellenmesi
- [ ] Docker kurulu olmayan sistemde uyari mesaji
- [ ] Bos container listesi durumu

---

### Adim 5.3: View Mode Gecisleri ve UI

**Commit:** "Add view mode switching and update process table headers"

#### Yapilacaklar

1. **View mode state yonetimi:**
```go
var currentViewMode ViewMode = ViewModeNormal

// d tusu davranisi guncelle
case "d":
    if currentViewMode == ViewModeNormal {
        if showDocker {
            currentViewMode = ViewModeDocker
        } else {
            currentViewMode = ViewModeGPU
        }
    }
    showDocker = !showDocker
    if !showDocker {
        currentViewMode = ViewModeNormal
    }
    updateGridLayout()
    render()
```

2. **Process table basligini guncelle:**
```go
func getTableTitle(mode ViewMode, sortMode SortMode) string {
    switch mode {
    case ViewModeGPU:
        return fmt.Sprintf(" GPU Processes Sort:%s ", sortMode)
    case ViewModeDocker:
        return fmt.Sprintf(" Docker Containers Sort:%s ", sortMode)
    default:
        return fmt.Sprintf(" Processes Sort:%s ", sortMode)
    }
}
```

3. **Kolon basliklarini degistir:**
```go
func getTableHeaders(mode ViewMode) []string {
    switch mode {
    case ViewModeGPU:
        return []string{"PID", "USER", "GPU%", "GPU_MEM", "STATE", "COMMAND"}
    case ViewModeDocker:
        return []string{"CONTAINER", "IMAGE", "CPU%", "MEM", "STATUS"}
    default:
        return []string{"PID", "USER", "CPU%", "MEM%", "STATE", "COMMAND"}
    }
}
```

4. **Status bar guncelle:**
```go
// GPU view aktifken
statusBar.Text = " F1:Help F8:Sort F9:Kill F10:Quit | d:Normal View | GPU Processes "

// Docker view aktifken
statusBar.Text = " F1:Help F8:Sort F9:Stop F10:Quit | d:Normal View | Docker Containers "
```

#### UI Mockup

**Normal View:**
```
┌─ CPU ──────┐┌─ Memory ───┐┌─ Swap ─────┐
│ ████░ 45%  ││ ███░░ 62%  ││ █░░░░ 15%  │
└────────────┘└────────────┘└────────────┘
┌─ CPU Cores ────────────┐┌─ Net ─┐┌─ Disk ┐┌─ GPU ──┐┌─ System ─┐
│ 0██ 1█░ 2███ 3█░ 4██░  ││ RX:.. ││ R:... ││ [NV]   ││ Host:... │
└────────────────────────┘└───────┘└───────┘└────────┘└──────────┘
┌─ Processes [Sort:CPU] ─────────────────────────────────────────┐
│ PID    USER    CPU%   MEM%   STATE   COMMAND                   │
│ 1234   root    45.2   12.3   R       python train.py           │
│ ...                                                            │
└────────────────────────────────────────────────────────────────┘
 F1:Help F8:Sort F9:Kill F10:Quit | /:Search t:Tree d:GPU/Docker
```

**GPU View (d ile gecis):**
```
┌─ CPU ──────┐┌─ Memory ───┐┌─ Swap ─────┐
│ ████░ 45%  ││ ███░░ 62%  ││ █░░░░ 15%  │
└────────────┘└────────────┘└────────────┘
┌─ CPU Cores ────────────┐┌─ Net ─┐┌─ Disk ┐┌─ GPU ──┐┌─ System ─┐
│ 0██ 1█░ 2███ 3█░ 4██░  ││ RX:.. ││ R:... ││ [NV]   ││ Host:... │
│                        ││       ││       ││ 45%    ││          │
└────────────────────────┘└───────┘└───────┘└────────┘└──────────┘
┌─ GPU Processes [nvidia-smi] Sort:GPU_MEM ──────────────────────┐
│ PID    USER    GPU%   GPU_MEM   STATE   COMMAND                │
│ 1234   root    45%    2.1 GB    R       python train.py        │
│ 5678   user    12%    512 MB    R       blender --gpu          │
└────────────────────────────────────────────────────────────────┘
 F1:Help F8:Sort F9:Kill F10:Quit | d:Normal View | GPU Processes
```

**Docker View (d ile gecis):**
```
┌─ CPU ──────┐┌─ Memory ───┐┌─ Swap ─────┐
│ ████░ 45%  ││ ███░░ 62%  ││ █░░░░ 15%  │
└────────────┘└────────────┘└────────────┘
┌─ CPU Cores ────────────┐┌─ Net ─┐┌─ Disk ┐┌─Docker─┐┌─ System ─┐
│ 0██ 1█░ 2███ 3█░ 4██░  ││ RX:.. ││ R:... ││ Run: 3 ││ Host:... │
│                        ││       ││       ││ Stop:1 ││          │
└────────────────────────┘└───────┘└───────┘└────────┘└──────────┘
┌─ Docker Containers Sort:CPU ───────────────────────────────────┐
│ CONTAINER    IMAGE              CPU%    MEM      STATUS        │
│ web-app      nginx:latest       2.5%    45 MB    Up 2 hours    │
│ database     postgres:15        5.1%    256 MB   Up 2 hours    │
│ redis        redis:7-alpine     0.3%    12 MB    Up 2 hours    │
└────────────────────────────────────────────────────────────────┘
 F1:Help F8:Sort F9:Stop F10:Quit | d:Normal View | Docker View
```

---

### Adim 5.4: Platform Uyumlulugu

**Commit:** "Add cross-platform support for GPU/Docker process views"

#### Platform Ozellikleri

| Ozellik | Linux | macOS | Windows | FreeBSD |
|---------|-------|-------|---------|---------|
| NVIDIA GPU Process | ✅ nvidia-smi | ✅ nvidia-smi | ✅ nvidia-smi | ❌ |
| AMD GPU Process | ✅ rocm-smi | ❌ | ❌ | ❌ |
| Docker Containers | ✅ | ✅ | ✅ | ✅ |
| Container PID Map | ✅ /proc/cgroup | ⚠️ Limited | ⚠️ Limited | ⚠️ Limited |

#### Yapilacaklar

1. **Build tag'leri ile platform-specific kod:**
```go
// gpu_nvidia.go
// +build linux windows darwin

// gpu_amd.go  
// +build linux

// docker_linux.go
// +build linux

// docker_windows.go
// +build windows
```

2. **Graceful fallback:**
```go
func GetGPUProcesses() ([]GPUProcess, error) {
    procs, err := getNVIDIAProcesses()
    if err == nil && len(procs) > 0 {
        return procs, nil
    }
    
    procs, err = getAMDProcesses()
    if err == nil && len(procs) > 0 {
        return procs, nil
    }
    
    return nil, fmt.Errorf("no GPU processes available")
}
```

---

### Adim 5.5: Sorting ve Navigasyon

**Commit:** "Add view-specific sorting and navigation"

#### Yapilacaklar

1. **GPU view sorting:**
   - g: Sort by GPU%
   - G: Sort by GPU Memory
   - c: Sort by CPU% (mevcut)
   - m: Sort by MEM% (mevcut)

2. **Docker view sorting:**
   - c: Sort by CPU%
   - m: Sort by Memory
   - n: Sort by Name
   - s: Sort by Status

3. **Kill/Stop davranisi:**
   - Normal view: `kill -9 <pid>`
   - GPU view: `kill -9 <pid>`
   - Docker view: `docker stop <container>`

---

## Implementasyon Sirasi

| # | Adim | Oncelik | Tahmini Sure |
|---|------|---------|--------------|
| 1 | GPU Process Listesi (NVIDIA) | Yuksek | 2-3 saat |
| 2 | GPU Process Listesi (AMD) | Orta | 1-2 saat |
| 3 | Docker Container View | Yuksek | 2-3 saat |
| 4 | View Mode UI Degisiklikleri | Yuksek | 1-2 saat |
| 5 | Platform Uyumlulugu | Orta | 1-2 saat |
| 6 | Sorting ve Navigasyon | Dusuk | 1 saat |
| **Toplam** | | | **8-13 saat** |

---

## Teknik Notlar

### nvidia-smi Komutlari

```bash
# Process listesi (basit)
nvidia-smi --query-compute-apps=pid,process_name,used_memory --format=csv,noheader,nounits

# Process listesi (detayli - GPU % dahil)
nvidia-smi pmon -c 1 -s um
# Cikti: idx, pid, type, sm, mem, enc, dec, command

# Sadece PID'ler
nvidia-smi --query-compute-apps=pid --format=csv,noheader
```

### rocm-smi Komutlari

```bash
# Process listesi
rocm-smi --showpidgpus

# Alternatif
rocm-smi --showpids
```

### Docker Komutlari

```bash
# Container listesi (JSON)
docker ps -a --format '{{json .}}'

# Container stats (JSON, tek seferlik)
docker stats --no-stream --format '{{json .}}'

# Container ana PID
docker inspect --format '{{.State.Pid}}' <container>
```

---

## Riskler ve Cozumler

| Risk | Etki | Cozum |
|------|------|-------|
| nvidia-smi yok | GPU view calismaz | Uyari mesaji goster, normal view'a don |
| rocm-smi yok | AMD GPU view calismaz | Uyari mesaji goster |
| Docker yok | Docker view calismaz | Uyari mesaji goster, normal view'a don |
| GPU process yok | Bos liste | "No GPU processes" mesaji |
| Container yok | Bos liste | "No containers running" mesaji |
| Izin hatasi | Komut calistiramaz | Uyari mesaji, graceful fallback |

---

## Eski Fazlar (Tamamlandi)

<details>
<summary>Faz 1-4 Detaylari (Tamamlandi)</summary>

### Faz 1: MVP Tamamlama ✅
- Per-core CPU bar chart

### Faz 2: Genisletilmis Ozellikler ✅
- Process sorting
- Network I/O
- Disk I/O
- UI Layout

### Faz 3: Ileri Duzey Ozellikler ✅
- Process tree view
- Process search/filter
- Config dosyasi
- Tema destegi
- Help ekrani
- Command-line argumanlar
- GPU monitoring
- Docker monitoring

### Faz 4: Son Rotuslar ✅
- Status bar
- F-key support
- AMD GPU support

</details>

---

*Son guncelleme: Faz 5 eklendi - Kontekst-Duyarli Process Listesi*
