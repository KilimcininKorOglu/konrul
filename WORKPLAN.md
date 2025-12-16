# Konrul - Is Plani

Bu dokuman, PRD'de belirtilen eksik ozelliklerin eklenmesi icin detayli is planini icerir.
Her adim bir commit ile tamamlanacaktir.

---

## Mevcut Durum

### Tamamlanan Ozellikler
- [x] CPU monitoring (toplam, gauge widget)
- [x] CPU history (sparkline, son 50 deger)
- [x] Memory monitoring (used/total, yuzde)
- [x] Swap monitoring (No Swap destegi)
- [x] Process listesi (PID, USER, CPU%, MEM%, STATE, COMMAND)
- [x] Process kill (K/Delete tuslari)
- [x] Sistem bilgisi (hostname, uptime, load, process count, platform)
- [x] Klavye kontrolleri (q, j/k, Up/Down, Home, End)
- [x] Responsive UI (terminal resize destegi)
- [x] Cross-platform (Linux, macOS, Windows, FreeBSD)
- [x] Dinamik kolon genislikleri
- [x] Windows CPU Queue Length

---

## Faz 1: MVP Tamamlama

### Adim 1.1: Per-core CPU Kullanimi
**Commit:** "Add per-core CPU usage display"

**Yapilacaklar:**
1. `cpu.Percent(0, true)` ile her core icin CPU yuzdesini al
2. CPU gauge'un altina veya yanina per-core gosterim ekle
3. Secenekler:
   - A) Her core icin mini gauge (cok yer kaplar)
   - B) Her core icin tek satirda bar (kompakt)
   - C) CPU panelinde core sayisi ve ortalama goster, detay icin ayri panel
4. Onerilen: Sparkline panelini "CPU Cores" olarak degistir, her core icin mini sparkline

**Dosyalar:**
- main.go (UI layout ve render fonksiyonu)

**Test:**
- Cok cekirdekli sistemde her core'un ayri gosterildigini dogrula
- Tek cekirdekli sistemde duzgun calistigini dogrula

---

## Faz 2: Genisletilmis Ozellikler

### Adim 2.1: Process Siralama Secenekleri
**Commit:** "Add process sorting options (CPU, MEM, PID, NAME)"

**Yapilacaklar:**
1. Siralama modu icin degisken ekle (cpu, mem, pid, name)
2. Klavye kisayollari ekle:
   - `c` - CPU'ya gore sirala (varsayilan)
   - `m` - Memory'ye gore sirala
   - `p` - PID'ye gore sirala
   - `n` - Name'e gore sirala
3. Process table basliginda aktif siralama modunu goster
4. Ters siralama icin `r` tusu (reverse)

**Dosyalar:**
- main.go

**Test:**
- Her siralama modunun dogru calistigini dogrula
- Ters siralamanin calistigini dogrula

---

### Adim 2.2: Network I/O Monitoring
**Commit:** "Add network I/O monitoring panel"

**Yapilacaklar:**
1. gopsutil/net paketini ekle
2. Network I/O icin yeni panel olustur
3. Gosterilecek bilgiler:
   - Total RX (received bytes)
   - Total TX (transmitted bytes)
   - RX/s (bytes per second)
   - TX/s (bytes per second)
4. UI layout'u guncelle (System panelini genislet veya yeni satir ekle)
5. Human-readable format (KB/s, MB/s)

**Dosyalar:**
- main.go
- go.mod (yeni import)

**Test:**
- Network aktivitesi sirasinda degerlerin degistigini dogrula
- Birden fazla network interface varsa toplam degerleri goster

---

### Adim 2.3: Disk I/O ve Kullanim
**Commit:** "Add disk I/O and usage monitoring"

**Yapilacaklar:**
1. gopsutil/disk paketini ekle
2. Disk bilgisi icin yeni panel veya mevcut System paneline ekle
3. Gosterilecek bilgiler:
   - Disk kullanimi (used/total, yuzde) - root partition
   - Read/s (bytes per second)
   - Write/s (bytes per second)
4. Birden fazla disk varsa toplam veya ana diski goster

**Dosyalar:**
- main.go
- go.mod

**Test:**
- Disk I/O sirasinda degerlerin degistigini dogrula
- Farkli platformlarda (Windows C:, Linux /) calistigini dogrula

---

### Adim 2.4: UI Layout Yeniden Duzenleme
**Commit:** "Reorganize UI layout for new panels"

**Yapilacaklar:**
1. Yeni layout tasarimi:
```
Row 1 (10%): [CPU Gauge] [Memory Gauge] [Swap Gauge]
Row 2 (12%): [CPU Cores - mini bars veya sparklines]
Row 3 (10%): [Network I/O] [Disk I/O] [System Info]
Row 4 (68%): [Process Table]
```
2. Grid oranlarini ayarla
3. Responsive tasarim icin minimum boyutlari kontrol et

**Dosyalar:**
- main.go

**Test:**
- Farkli terminal boyutlarinda duzgun gorunumu dogrula
- Minimum 80x24 terminalde calistigini dogrula

---

### Adim 2.5: Process Tree Gorunumu
**Commit:** "Add process tree view toggle"

**Yapilacaklar:**
1. Tree view modu icin degisken ekle
2. `t` tusu ile tree/flat view arasinda gecis
3. Tree view icin:
   - Parent PID bilgisini al (PPID)
   - Process'leri parent-child iliskisine gore sirala
   - Indent ile hierarchy goster (ornek: "  |- child_process")
4. Root process'leri (PPID=0 veya 1) en ustte goster

**Dosyalar:**
- main.go

**Test:**
- Tree view'da parent-child iliskisinin dogru gosterildigini dogrula
- Flat view'a geri donusun calistigini dogrula

---

## Faz 3: Ileri Duzey Ozellikler

### Adim 3.1: Process Filtreleme ve Arama
**Commit:** "Add process search and filter functionality"

**Yapilacaklar:**
1. Arama modu icin degisken ve input buffer ekle
2. `/` tusu ile arama moduna gir
3. Arama sirasinda:
   - Alt kisimda arama kutusu goster
   - Her karakter girisinde filtreleme yap
   - Enter ile aramadan cik, ESC ile iptal
4. Filtreleme kriterleri:
   - Process name
   - Command
   - PID
   - User
5. `Escape` ile filtreyi temizle

**Dosyalar:**
- main.go

**Test:**
- Arama sonuclarinin dogru filtrelendigini dogrula
- Buyuk/kucuk harf duyarsiz aramayi dogrula
- Ozel karakterlerin duzgun calistigini dogrula

---

### Adim 3.2: Config Dosyasi Destegi
**Commit:** "Add configuration file support"

**Yapilacaklar:**
1. Config struct olustur:
```go
type Config struct {
    RefreshInterval int    // saniye
    DefaultSort     string // cpu, mem, pid, name
    ShowTreeView    bool
    Theme           string // default, dark, light
    Columns         []string // gorunur kolonlar
}
```
2. Config dosyasi konumlari:
   - Linux/macOS: ~/.config/konrul/config.yaml
   - Windows: %APPDATA%\konrul\config.yaml
3. YAML parser ekle (gopkg.in/yaml.v3)
4. Varsayilan config olustur
5. Command-line flag ile config dosyasi belirt: `--config`

**Dosyalar:**
- main.go
- config.go (yeni)
- go.mod

**Test:**
- Config dosyasi yoksa varsayilan degerlerle calistigini dogrula
- Config dosyasindaki degerlerin uygulandigini dogrula

---

### Adim 3.3: Refresh Interval Ayari
**Commit:** "Add configurable refresh interval"

**Yapilacaklar:**
1. Config'den refresh interval oku
2. Command-line flag ekle: `--interval` veya `-i`
3. Varsayilan: 1 saniye
4. Minimum: 100ms, Maximum: 10 saniye
5. UI'da mevcut interval'i goster (System panelinde)

**Dosyalar:**
- main.go
- config.go

**Test:**
- Farkli interval degerlerinin calistigini dogrula
- Cok dusuk interval'de performans sorunlarini kontrol et

---

### Adim 3.4: Tema Destegi
**Commit:** "Add theme support (default, dark, light, custom)"

**Yapilacaklar:**
1. Theme struct olustur:
```go
type Theme struct {
    CPUColor     ui.Color
    MemoryColor  ui.Color
    SwapColor    ui.Color
    BorderColor  ui.Color
    TextColor    ui.Color
    SelectionBg  ui.Color
    SelectionFg  ui.Color
}
```
2. Varsayilan temalar: default, dark, light
3. Config dosyasindan tema sec
4. `T` tusu ile tema degistir (runtime)

**Dosyalar:**
- main.go
- theme.go (yeni)
- config.go

**Test:**
- Tum temalarin duzgun gorunumunu dogrula
- Tema degistirmenin aninda uygulandigini dogrula

---

### Adim 3.5: Help Ekrani
**Commit:** "Add help screen with keyboard shortcuts"

**Yapilacaklar:**
1. `?` veya `h` tusu ile help ekrani ac
2. Help ekraninda goster:
   - Tum klavye kisayollari
   - Siralama secenekleri
   - Mevcut versiyon
3. Herhangi bir tus ile help'ten cik
4. Help icin popup/overlay widget kullan

**Dosyalar:**
- main.go

**Test:**
- Help ekraninin duzgun acilip kapandigini dogrula
- Tum kisayollarin dogru listelendigini dogrula

---

### Adim 3.6: Command-line Argumanlar
**Commit:** "Add command-line argument support"

**Yapilacaklar:**
1. flag paketi ile argumanlar:
   - `--version`, `-v`: Versiyon bilgisi
   - `--help`: Kullanim bilgisi
   - `--config`, `-c`: Config dosyasi yolu
   - `--interval`, `-i`: Refresh interval
   - `--sort`, `-s`: Varsayilan siralama
   - `--tree`: Tree view ile baslat
2. Argumanlar config dosyasini override etsin

**Dosyalar:**
- main.go

**Test:**
- Tum argumanlarin calistigini dogrula
- --help ciktisinin dogru oldugunu dogrula
- --version ciktisinin build bilgilerini icerdigini dogrula

---

### Adim 3.7: Docker Container Monitoring (Opsiyonel)
**Commit:** "Add Docker container monitoring support"

**Yapilacaklar:**
1. Docker API client ekle (docker/docker/client)
2. Container listesi icin ayri panel veya tab
3. Gosterilecek bilgiler:
   - Container ID
   - Image
   - Status
   - CPU%
   - Memory
4. `d` tusu ile Docker view'a gec
5. Docker yoksa veya erisim yoksa hata mesaji goster

**Dosyalar:**
- main.go
- docker.go (yeni)
- go.mod

**Test:**
- Docker kurulu sistemde container listesini dogrula
- Docker kurulu olmayan sistemde graceful fallback

---

### Adim 3.8: GPU Monitoring (Opsiyonel)
**Commit:** "Add GPU monitoring support (NVIDIA)"

**Yapilacaklar:**
1. NVIDIA GPU icin nvidia-smi veya NVML binding
2. GPU bilgisi icin yeni panel
3. Gosterilecek bilgiler:
   - GPU kullanimi %
   - Memory kullanimi
   - Sicaklik
   - Fan hizi
4. GPU yoksa paneli gizle veya "No GPU" goster

**Dosyalar:**
- main.go
- gpu.go (yeni)
- go.mod

**Test:**
- NVIDIA GPU'lu sistemde bilgilerin dogru gosterildigini dogrula
- GPU olmayan sistemde graceful fallback

---

## Faz 4: Son Rotuslari

### Adim 4.1: Error Handling Iyilestirmeleri
**Commit:** "Improve error handling and logging"

**Yapilacaklar:**
1. Tum hata durumlarini kontrol et
2. Kullaniciya anlamli hata mesajlari goster
3. Debug modu icin log dosyasi destegi
4. `--debug` flagi ekle

**Dosyalar:**
- main.go
- Tum .go dosyalari

---

### Adim 4.2: Performance Optimizasyonlari
**Commit:** "Optimize performance and reduce resource usage"

**Yapilacaklar:**
1. Process listesi icin daha akilli caching
2. Gereksiz render cagrilarini azalt
3. Memory allocation'lari optimize et
4. Profiling ile bottleneck'leri bul

**Dosyalar:**
- main.go

**Test:**
- Idle durumda CPU kullanimi < 1% dogrula
- Memory kullanimi < 20MB dogrula

---

### Adim 4.3: README ve Dokumantasyon Guncellemesi
**Commit:** "Update documentation for all new features"

**Yapilacaklar:**
1. README.md'yi tum yeni ozelliklerle guncelle
2. Klavye kisayollari tablosunu guncelle
3. Config dosyasi ornegi ekle
4. Ekran goruntuleri ekle (opsiyonel)

**Dosyalar:**
- README.md
- CONFIG.md (yeni - config dokumantasyonu)

---

### Adim 4.4: Final Test ve Release Hazirligi
**Commit:** "Prepare for v1.0.0 release"

**Yapilacaklar:**
1. Tum platformlarda test et
2. CHANGELOG.md olustur
3. LICENSE dosyasini kontrol et
4. Git tag olustur: v1.0.0
5. Release binary'leri olustur

**Dosyalar:**
- CHANGELOG.md (yeni)
- Tum binary'ler (bin/)

---

## Commit Sirasi Ozeti

| # | Commit | Faz |
|---|--------|-----|
| 1 | Add per-core CPU usage display | 1.1 |
| 2 | Add process sorting options | 2.1 |
| 3 | Add network I/O monitoring panel | 2.2 |
| 4 | Add disk I/O and usage monitoring | 2.3 |
| 5 | Reorganize UI layout for new panels | 2.4 |
| 6 | Add process tree view toggle | 2.5 |
| 7 | Add process search and filter functionality | 3.1 |
| 8 | Add configuration file support | 3.2 |
| 9 | Add configurable refresh interval | 3.3 |
| 10 | Add theme support | 3.4 |
| 11 | Add help screen with keyboard shortcuts | 3.5 |
| 12 | Add command-line argument support | 3.6 |
| 13 | Add Docker container monitoring support | 3.7 (opsiyonel) |
| 14 | Add GPU monitoring support | 3.8 (opsiyonel) |
| 15 | Improve error handling and logging | 4.1 |
| 16 | Optimize performance | 4.2 |
| 17 | Update documentation | 4.3 |
| 18 | Prepare for v1.0.0 release | 4.4 |

---

## Notlar

- Her commit sonrasi `build.bat build-all` ile tum platformlarda build alinacak
- Her commit sonrasi manual test yapilacak
- Buyuk degisiklikler icin branch olusturulabilir
- Opsiyonel ozellikler (Docker, GPU) proje sahibinin tercihine bagli

---

## Tahmini Sure

| Faz | Sure |
|-----|------|
| Faz 1 (MVP Tamamlama) | 1-2 saat |
| Faz 2 (Genisletilmis) | 3-4 saat |
| Faz 3 (Ileri Duzey) | 4-6 saat |
| Faz 4 (Son Rotuslar) | 2-3 saat |
| **Toplam** | **10-15 saat** |

---

*Bu dokuman, gelistirme surecinde guncellenecektir.*
