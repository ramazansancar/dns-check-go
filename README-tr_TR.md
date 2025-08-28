# DNS Sunucu Denetleyicisi

Birden fazla DNS sunucusunu çeşitli alan adlarıyla test eden ve gerçek zamanlı ilerleme takibi ile detaylı kategorize raporlar sunan kapsamlı bir DNS test aracı.

## Özellikler

- **Eşzamanlı DNS Testi**: Optimal performans için birden fazla DNS sunucusunu aynı anda test eder
- **Alan Adı Kategorilendirmesi**: Alan adlarını otomatik olarak kategorize eder (Genel, Reklam-sunucusu, Diğer, Yetişkin)
- **Gerçek Zamanlı İlerleme**: Zaman tahminleri ve tamamlanma takibi ile etkileşimli ilerleme çubuğu
- **Çoklu Çıktı Formatları**: JSON ve metin çıktısı desteği
- **Kapsamlı Raporlama**: Kategori bazlı başarı oranları ile detaylı istatistikler
- **Yapılandırılabilir Parametreler**: Özelleştirilebilir zaman aşımı, worker sayısı ve çıktı formatı
- **İkili Özet Gösterimi**: Sonuçların başında ve sonunda özet görüntülenir

## Yapılandırma

Araç, kolayca değiştirilebilir önceden tanımlanmış yapılandırma sabitleri içerir:

```go
// Yapılandırma Sabitleri
const (
    DefaultTimeout     = 15          // DNS sorgu zaman aşımı (saniye)
    DefaultWorkerCount = 50          // Eşzamanlı worker sayısı
    DefaultFormat     = "text"      // Varsayılan çıktı formatı
    ProgressBarWidth  = 40          // İlerleme çubuğu genişliği (karakter)
    ProgressUpdateRate = 100 * time.Millisecond // İlerleme güncelleme sıklığı
)

// Kategori Sabitleri
const (
    CategoryGeneral   = "General"
    CategoryAdServer  = "Ad-server"
    CategoryOther     = "Other"
    CategoryAdult     = "Adult"
)
```

## Kullanım

### Temel Kullanım

```bash
# Go ile doğrudan çalıştırma (geliştirme)
go run .

# Veya derlenmiş binary kullanarak
dns-check-go
```

### Komut Satırı Parametreleri (CLI)

```bash
# Özel DNS sunucuları dosyası belirtme
go run . --list ozel-sunucular.txt
dns-check-go --list ozel-sunucular.txt

# Özel alan adları dosyası belirtme
go run . --domains ozel-alanlar.txt
dns-check-go --domains ozel-alanlar.txt

# Çıktı formatını JSON olarak ayarlama
go run . --format json
dns-check-go --format json

# Zaman aşımı ve worker sayısını özelleştirme
go run . --timeout 20 --workers 100
dns-check-go --timeout 20 --workers 100

# Çıktıyı dosyaya kaydetme
go run . --output sonuclar.txt
dns-check-go --output sonuclar.txt

# Yardım gösterme
go run . --help
dns-check-go --help

# Birden fazla seçeneği birleştirme
go run . --list sunucular.txt --domains alanlar.txt --format json --timeout 10 --workers 75 --output sonuclar.json
dns-check-go --list sunucular.txt --domains alanlar.txt --format json --timeout 10 --workers 75 --output sonuclar.json
```

## Komut Satırı Parametreleri

| Parametre | Varsayılan | Açıklama |
|-----------|------------|----------|
| `--list` | Yerleşik DNS sunucuları | DNS sunucuları liste dosyasının yolu |
| `--domains` | Yerleşik alan adları | Alan adları liste dosyasının yolu |
| `--format` | `text` | Çıktı formatı (`text` veya `json`) |
| `--timeout` | `15` | DNS sorgu zaman aşımı (saniye) |
| `--workers` | `50` | Eşzamanlı worker sayısı |
| `--output` | - | Çıktı dosyası yolu (isteğe bağlı, belirtilmezse stdout'a yazdırır) |

## Dosya Formatları

### DNS Sunucuları Dosyası (`dns-servers.txt`)

```text
# Format: IP_ADRESI AÇIKLAMA (isteğe bağlı)
# # ile başlayan satırlar yorumdur
8.8.8.8 Google Public DNS
1.1.1.1 Cloudflare DNS
208.67.222.222 OpenDNS
9.9.9.9 Quad9 DNS
```

### Alan Adları Dosyası (`domains.txt`)

```text
# Format: ALAN_ADI KATEGORI
# # ile başlayan satırlar yorumdur
google.com general
facebook.com general
youtube.com general
doubleclick.net ad-server
googlesyndication.com ad-server
yetiskin-site.xxx adult
bilinmeyen-kategori.com other
```

## Alan Adı Kategorileri

- **General**: Yaygın web siteleri ve hizmetler (google.com, facebook.com, vb.)
- **Ad-server**: Reklam ve takip alan adları (doubleclick.net, vb.)
- **Other**: Kategorize edilmemiş veya çeşitli alan adları
- **Adult**: Yetişkin içerik web siteleri

## İlerleme Takibi

Araç gerçek zamanlı ilerleme bilgisi sağlar:

```text
DNS sunucuları test ediliyor... ████████████████████████████████████████ 100% (320/320) ETA: 0s
12.5 saniyede tamamlandı
```

İlerleme çubuğu özellikleri:

- Unicode blokları ile görsel ilerleme göstergesi
- Mevcut/toplam test sayısı
- Yüzde tamamlanma
- Tahmini tamamlanma süresi (ETA)
- Toplam geçen süre

## Kurulum

### Önkoşullar

- Go 1.21 veya üzeri
- DNS sorguları için internet bağlantısı

### Yöntem 1: Kopyala ve Kaynaktan Derle

```bash
# Depoyu klonlayın
git clone <repository-url>
cd dns-check-go

# Bağımlılıkları yükleyin
go mod tidy

# Go ile doğrudan çalıştırın (geliştirme)
go run .
```

### Yöntem 2: Binary Derle

```bash
# Depoyu klonlayın
git clone <repository-url>
cd dns-check-go

# Go modülünü başlatın (gerekirse)
go mod init dns-check-go

# Bağımlılıkları yükleyin
go get github.com/miekg/dns

# Çalıştırılabilir dosyayı derleyin
go build -o dns-check-go main.go

# Binary'yi çalıştırın
./dns-check-go
```

### Bağımlılıklar

```bash
go get github.com/miekg/dns
```

## Derleme

```bash
# Mevcut platform için derleme
go build -o dns-check-go main.go

# Windows için derleme
GOOS=windows GOARCH=amd64 go build -o dns-check-go.exe main.go

# Linux için derleme
GOOS=linux GOARCH=amd64 go build -o dns-check-go-linux main.go

# macOS için derleme
GOOS=darwin GOARCH=amd64 go build -o dns-check-go-mac main.go
```

## Performans Notları

- **Eşzamanlı İşleme**: Yapılandırılabilir eşzamanlı worker'lar ile worker havuzu deseni kullanır
- **İlerleme Güncellemeleri**: Yükü minimize etmek için ilerleme çubuğu her 100ms'de güncellenir
- **Bellek Verimliliği**: Akışlı sonuçlar ile optimize edilmiş bellek kullanımı
- **Ölçeklenebilir**: Varsayılan yapılandırma 50'ye kadar eşzamanlı worker'ı destekler
- **Hızlı Yürütme**: Tipik olarak 4 DNS sunucusu × 20 alan adı testi 5-15 saniyede tamamlanır

## Lisans

Bu proje açık kaynaklıdır. İhtiyaçlarınıza göre serbestçe kullanın, değiştirin ve dağıtın.

## Katkıda Bulunma

DNS Sunucu Denetleyicisi'ni geliştirmek için katkılarınızı bekliyoruz! İşte nasıl katkıda bulunabileceğiniz:

### Başlangıç

1. **Depoyu Fork Edin**: Projenin kendi fork'unuzu oluşturun
2. **Fork'unuzu Klonlayın**: `git clone https://github.com/kullanici-adiniz/dns-check-go.git`
3. **Branch Oluşturun**: `git checkout -b feature/ozellik-adiniz`

### Geliştirme Kurulumu

```bash
# Fork'unuzu klonlayın
git clone https://github.com/kullanici-adiniz/dns-check-go.git
cd dns-check-go

# Bağımlılıkları yükleyin
go mod tidy

# Testleri çalıştırın (mevcut olduğunda)
go test ./...

# Test için aracı çalıştırın
go run . -dns-file dns-servers.txt -domains-file domains.txt
```

### Katkı Türleri

- **🐛 Hata Raporları**: Hata buldunuz mu? Lütfen detaylı bilgiyle issue oluşturun
- **💡 Özellik İstekleri**: Yeni özellikler için fikirleriniz var mı? Duymak isteriz!
- **📖 Dokümantasyon**: README, yorumları iyileştirin veya örnekler ekleyin
- **🔧 Kod İyileştirmeleri**: Performans optimizasyonları, kod refaktörü
- **🌍 Çeviriler**: Daha fazla dil desteği ekleyin
- **🧪 Test**: Birim testleri, entegrasyon testleri ekleyin veya farklı konfigürasyonlarla test edin

### Kod Rehberleri

- Go en iyi uygulamalarını ve konvansiyonlarını takip edin
- Fonksiyonları odaklı ve iyi dokümante edilmiş tutun
- Anlamlı değişken ve fonksiyon isimleri kullanın
- Karmaşık mantık için yorumlar ekleyin
- Mümkün olduğunda geriye uyumluluğu sağlayın

### Değişiklikleri Gönderme

1. **Değişikliklerinizi Test Edin**: Kodunuzun çeşitli DNS sunucuları ve domain listeleriyle çalıştığından emin olun
2. **Dokümantasyonu Güncelleyin**: Yeni özellikler eklediyseniz README'yi güncelleyin
3. **Değişikliklerinizi Commit Edin**: Açıklayıcı commit mesajları kullanın
   ```bash
   git add .
   git commit -m "Özellik ekle: DNS over HTTPS desteği"
   ```
4. **Fork'unuza Push Edin**: `git push origin feature/ozellik-adiniz`
5. **Pull Request Oluşturun**: Değişikliklerin detaylı açıklamasıyla PR gönderin

### Pull Request Rehberleri

- **Net Başlık**: Açıklayıcı başlıklar kullanın (örn: "Büyük domain listeleri için timeout işlemini düzelt")
- **Detaylı Açıklama**: Ne değiştirdiğinizi ve neden değiştirdiğinizi açıklayın
- **Test Sonuçları**: Uygunsa test sonuçları veya ekran görüntüleri ekleyin
- **Issue Bağlantıları**: İlgili issue'ları `#issue-numarası` ile referans edin

## Destek

Sorunlarla karşılaşırsanız veya sorularınız varsa:

### Sorun Bildirme

1. **Mevcut Sorunları Kontrol Edin**: Önce [mevcut issue'ları](https://github.com/username/dns-check-go/issues) arayın
2. **Detaylı Raporlar Oluşturun**: Yeni issue oluştururken lütfen şunları ekleyin:
   - **Sistem Bilgileri**: İşletim sistemi, Go sürümü, sistem mimarisi
   - **Kullanılan Komut**: Soruna neden olan tam komutu
   - **Beklenen Davranış**: Ne olmasını beklediğiniz
   - **Gerçek Davranış**: Gerçekte ne olduğu
   - **Hata Mesajları**: Herhangi bir hata mesajı veya log
   - **Örnek Dosyalar**: İlgiliyse, DNS sunucuları veya domain dosyalarınızı ekleyin

### Sorun Şablonları

**Hata Raporu Örneği:**
```text
**Sistem Bilgileri:**
- İS: Windows 11 / Ubuntu 22.04 / macOS 13
- Go Sürümü: 1.21.5
- Mimari: amd64

**Kullanılan Komut:**
`dns-check-go -dns-file ozel-sunucular.txt -timeout 5 -workers 100`

**Beklenen Davranış:**
Araç belirtilen timeout süresi içinde tüm domain testlerini tamamlamalı

**Gerçek Davranış:**
Araç %50 ilerlemede takılıyor ve tamamlanmıyor

**Hata Mesajları:**
[Buraya hata çıktısını ekleyin]

**Ek Bilgiler:**
- Özel DNS dosyası 10 sunucu içeriyor
- 50 domain ile test ediliyor
- Sorun sürekli oluşuyor
```

### Yardım Alma

- **📚 Dokümantasyon**: Kapsamlı kullanım bilgisi için bu README'yi kontrol edin
- **🐛 Hata Raporları**: Hatalar veya beklenmeyen davranış için issue oluşturun
- **💬 Özellik İstekleri**: GitHub issue'ları aracılığıyla yeni özellikler önerin
- **❓ Sorular**: Genel sorular için discussion veya issue oluşturun

### Topluluk Rehberleri

- Tüm etkileşimlerde saygılı ve yapıcı olun
- Sorun bildirirken detaylı bilgi sağlayın
- Bilginizi ve deneyiminizi paylaşarak diğerlerine yardım edin
- Katkı göndermeden önce kapsamlı test yapın
