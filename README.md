# FlowKit

Trypix case çalışması: veri olarak tanımlanan AI üretim workflow'larını çalıştıran Go backend.

**Şu anki durum:** doğrudan sağlayıcı deneme aracı hazırlanmıştır. Workflow motoru, API, PostgreSQL worker, dashboard ve Docker Compose henüz uygulanmamıştır. Gerçek sağlayıcı denemesi henüz yapılmamıştır.

## İlk adım: doğrudan akış

1. `.env.example` dosyasındaki iki anahtarı yerel `.env` dosyasına girin. Anahtarları repoya eklemeyin.
2. Ödeme gerektirmeyen payload önizlemesi:

```powershell
go run ./cmd/direct -dry-run
```

3. Gerçek OpenRouter -> fal queue denemesi (sağlayıcı bakiyesinden harcar):

```powershell
go run ./cmd/direct -input "Matte black ceramic coffee mug with a wooden handle"
```

PATH üzerinde Go yoksa bu makinede komutun başını şöyle kullanın:

```powershell
& .\.tools\go1.27.2\go\bin\go.exe run ./cmd/direct -dry-run
```

Araç OpenRouter'dan İngilizce bir ürün fotoğrafı prompt'u üretir; fal'a bir adet 512x512 görsel isteği gönderir. Sabit model seçimleri `.env` içinde açıkça yazılıdır. Model tahmini veya başka modele otomatik geçiş yapılmaz. Denemede otomatik yeniden gönderim yapılmaz.

Her deneme `artifacts/direct/<timestamp>/evidence.json` dosyasına anahtarsız istek/yanıt, model bilgisi, prompt ve fal request ID kaydeder. fal ID ve queue URL'leri alındığı anda dosyaya yazılır. Bekleme sırasında hata olursa aynı isteği kontrol etmeye devam etmek için:

```powershell
go run ./cmd/direct -resume artifacts/direct/<timestamp>/evidence.json
```

Resume yeni OpenRouter/fal submit çağrısı yapmaz. Normal komutu yeniden çalıştırmak yeni ücretli bir üretim başlatır. Görsel URL'leri sağlayıcının saklama politikasına tabidir; bu aşama görselleri indirmez.

`.env` okuyucu basit `KEY=value` satırlarını ve tırnaklı değerleri destekler; shell değişken genişletmesi veya satır sonu yorumları yapmaz. Ortamda zaten tanımlı değerler önceliklidir.

## Planlanan mimari

- Go HTTP API: model kataloğu, workflow CRUD/doğrulama ve run başlatma/detay.
- PostgreSQL: katalog, workflow ve değişmez run snapshot'ları, adım/deneme izleri; aynı veritabanında iş kuyruğu.
- Ayrı Go worker: işleri kilitleyerek alma, sıralı yürütme, tiplendirilmiş hata ve node bazlı retry.
- Basit dashboard: API üzerinden JSON düzenleme, doğrulama, çalıştırma ve iz görüntüleme. İlk tercih ek frontend build zinciri gerektirmeyen HTML/CSS/JavaScript.
- Sonuç aktarımı: polling. Kısa süre ve sınırlı kapsam için Redis, WebSocket ve görsel canvas gerekmiyor.

Bu kararlar uygulama öncesi öneridir. Sağlayıcı denemesinden sonra veri sözleşmesini birlikte netleştireceğiz.

## Sağlayıcı kaynakları

- [OpenRouter chat completions](https://openrouter.ai/docs/api/api-reference/chat/send-chat-completion-request)
- [Gemini 2.5 Flash Lite](https://openrouter.ai/google/gemini-2.5-flash-lite)
- [fal queue REST API](https://fal.ai/docs/documentation/model-apis/inference/queue)
- [FLUX.1 schnell parametreleri](https://fal.ai/models/fal-ai/flux/schnell/api)
- [FLUX.1 dev parametreleri](https://fal.ai/models/fal-ai/flux/dev/api)

Deneme modeli için Gemini Flash Lite ve FLUX schnell seçildi; kısa prompt ve tek düşük çözünürlüklü çıktı kapsamı sınırlar. Kesin maliyet gerçek sağlayıcı yanıtı/hesabı ile kontrol edilecek; şu an doğrulanmış harcama yok.

## Doğrulama

```powershell
go test ./...
go vet ./...
```

Testler yerel HTTP sunucuları kullanır; gerçek sağlayıcı çağrısı yapmaz. Kabul takibi: [docs/ACCEPTANCE.md](docs/ACCEPTANCE.md). AI çalışma kaydı: [docs/AI_LOG.md](docs/AI_LOG.md).
