# FlowKit

JSON olarak tanımlanan doğrusal AI workflow'larını kaydeden, doğrulayan ve arka planda çalıştıran küçük bir backend platformu. Go API + ayrı River worker + PostgreSQL; React dashboard yalnızca HTTP API kullanır.

**Durum:** uygulama, mock testler ve canlı üretim kabulü tamamlandı. Doğrudan OpenRouter → fal denemesi, FlowKit Schnell run'ı, API'den yeni Dev modeline geçiş ve ikinci workflow'un Dev run'ı başarılı. Yeni model/workflow eklerken kod değişmedi, servisler yeniden başlatılmadı, deploy yapılmadı. Fal billing okuması mevcut anahtarın yetkisi nedeniyle 403; bilinmeyen gerçek tutarlar null kalıyor. [Kabul kanıtları ve billing sınırı](docs/ACCEPTANCE.md). Çıktı kalitesi değerlendirmesine yönelik ek üretim yapılmaz.

Repository: [CanhK-2103/trypix-flowkitCase](https://github.com/CanhK-2103/trypix-flowkitCase). Gerçek çıktılarla dashboard: [ürün / Schnell](docs/dashboard-schnell-live.jpg), [etkinlik / Dev](docs/dashboard-event-live.jpg).

## Çalıştırma

```sh
cp .env.example .env
# OPENROUTER_API_KEY ve FAL_KEY alanlarını kendi anahtarlarınızla doldurun.
docker compose up
```

Dashboard ve API: **http://localhost:8080**. PowerShell: `Copy-Item .env.example .env`. Anahtarlar olmadan servisler ve editör açılır; üretim isteği `PROVIDER_NOT_CONFIGURED` ile reddedilir. Provider anahtarları frontend'e verilmez, Git'e girmez.

Servislerin çalışması için Docker Compose yeterlidir. Yerel geliştirme ve Go demo komutları Go 1.27.2; frontend komutları Node 24 LTS gerektirir.

Init uygulama/River migration'larını uygular. Bootstrap örnek JSON'ları **API üzerinden** yükler: ürün fotoğrafı / Schnell ve etkinlik afişi / Dev. Startup herhangi bir üretim veya fiyat isteği yapmaz. Tekrar yüklemede mevcut kayıtlar, düzenlemeler, disabled modeller ve silinmiş workflow kimlikleri korunur. Boş kabul ortamı için `.env` içinde `LOAD_EXAMPLES=false`, farklı Compose proje adı ve dashboard portu kullanın:

```sh
DASHBOARD_PORT=8081 LOAD_EXAMPLES=false docker compose -p flowkit-acceptance up
```

Normal kurulumu yeniden başlatmanız veya verilerini silmeniz gerekmez. Yerel gateway yalnızca 127.0.0.1'e bağlıdır. İsteğe bağlı VPS kurulumu için [deployment yönergesi](deploy/README.md) ve `compose.production.yaml` bulunur: Caddy HTTPS, kullanıcı adı gerektirmeyen tek şifreli Go gate, özel DB parolası ve açıkça başlatılan worker. Gate dashboard/API/görselleri 12 saatlik imzalı oturumla korur; yerel Compose giriş ekranı gerektirmez.

Parola korumalı yönetim paneli: **https://flowkit.canhurkurtoglu.com**. Cloudflare DNS, public Let’s Encrypt sertifikası/hostname doğrulaması, HTTPS yönlendirmesi, tek şifreli login, oturumsuz API/asset 401, yetkili dashboard/API 200, CSRF reddi/logout ve iki seed workflow [deployment kanıtında](docs/evidence/deployment.json). [Giriş ekranı](docs/flowkit-login.jpg) yayından alınmıştır. Üretim worker'ı cloud'da kapalı, provider anahtarları boş; bu kurulum ek ücretli üretim yapmaz. Değerlendirici site şifresi ayrıca paylaşılır, repoda bulunmaz. Önceki gerçek run'lar/medya yerel kabul DB'sindedir; cloud yeni DB ile açılmıştır.

Docker Desktop yeniden açıldığında veritabanı, API ve dashboard `unless-stopped` politikasıyla geri gelir; elle durdurulmuş servisler elle açılır. Her sağlayıcı çağrısını ayrıca onaylayacağınız bir oturumda worker'ı kapalı tutarak yalnızca yönetim servislerini açın:

```sh
docker compose stop worker
docker compose up -d --wait db init api bootstrap dashboard
```

Bu durumda workflow düzenleme, listeleme ve maliyet tahmini kullanılabilir; başlatılan run'lar worker açılana kadar kuyrukta bekler. Worker'ı anahtarlarla açmak kuyruktaki run'ları ve bekleyen faturalama sorgularını otomatik yürütebilir; kapsam onaylanmadan `docker compose up` ile worker'ı başlatmayın.

Her gerçek çağrıya ayrı onay gereken bu oturum için [kontrollü kabul demosu](docs/APPROVED_DEMO.md) ve `compose.approval.yaml` hazırlanmıştır. Araç aynı River/Engine/DB akışını kullanır; normal worker kapalıyken açıkça onaylanan tek isteği yürütür. Model/workflow eklerken API ve kontrollü worker yeniden başlatılmaz.

## Mimari ve çalıştırma

```mermaid
flowchart LR
  UI[React / TanStack Query / CodeMirror] --> API[Go + chi API]
  API --> DB[(PostgreSQL / sqlc)]
  DB --> W[River worker]
  W --> OR[OpenRouter chat]
  W --> F[fal submit → status → result]
  W --> V[(Artifact volume)]
  API --> V
```

API run, immutable workflow/model snapshot'ları ve River job'ını **aynı PostgreSQL transaction'ında** kaydeder; hemen 202 döner. Worker grafı edge sırasıyla yorumlar. Model kimlikleri iş mantığında hardcode edilmez; node katalog alias'ına referans verir. sqlc elle yazılmış SQL'den pgx kodu üretir; yeni model/workflow verisi için generate veya restart gerekmez.

Worker `sending → waiting_provider → fetch_result → result_saved → succeeded` fazlarını transaction ile kaydeder. Fal request ID ve dönen status/result URL'leri **submit'ten hemen sonra, polling/fiyat/indirme öncesinde** commit edilir. `JobSnooze` polling/retry beklemelerinde worker slotunu serbest bırakır. Üretilen görseller 20 MiB sınırı, MIME kontrolü ve atomik dosya taşıma ile kalıcı volume'a alınır; provider URL'sinin ömrüne bağımlı değildir.

TanStack Query aktif run detayını 2 saniyede bir yeniler; üretim bittikten sonra ücret kaydı bekleniyorsa 15 saniyede bir maliyeti yeniler, ücret doğrulanınca veya erişilemiyorsa durur. Mutation otomatik retry yapmaz. CodeMirror sözdizimi hatalarını gösterir; semantik kurallar her kaydetmede backend'de uygulanır.

## API

| Method | Endpoint | İşlem |
|---|---|---|
| POST / GET | `/models` | Model kaydı / katalog |
| GET / PATCH | `/models/{alias}` | Detay / defaults, pricing, enabled güncelleme |
| POST / GET | `/workflows` | Doğrulanmış oluşturma / liste |
| GET / PUT / DELETE | `/workflows/{id}` | Detay / doğrulanmış güncelleme / soft-delete |
| POST | `/workflows/validate` | Kaydetmeden `{valid, findings}` |
| POST | `/workflows/{id}/runs` | Asenkron run; optional Idempotency-Key |
| POST | `/workflows/{id}/estimate` | Üretim çağırmadan node ve toplam başlangıç tahmini |
| GET | `/runs` | Geçmiş; workflow_id/status filtreleri |
| GET | `/runs/{id}/status` | Hafif durum |
| GET | `/runs/{id}` | Snapshot, node/attempt izi, hata, çıktı, maliyet |
| POST | `/runs/{id}/costs/refresh` | Bitmiş run'ın eksik ücret kayıtlarını yeniden sorgulamayı planla |
| GET | `/artifacts/{id}` | Kalıcı görsel |
| GET | `/health/live`, `/health/ready` | Canlılık / DB ve migration hazırlığı |

Listeleme `limit` (varsayılan 25, max 100) ve `cursor` kullanır; yanıt `{items, next_cursor}`. OpenAPI: [docs/openapi.json](docs/openapi.json). Workflow/katalog camelCase; run/log/hata metadata snake_case.

```sh
curl http://localhost:8080/models
curl -X POST http://localhost:8080/workflows/validate \
  -H 'Content-Type: application/json' --data-binary @examples/workflows/product.json
curl -X POST http://localhost:8080/workflows/10000000-0000-4000-8000-000000000001/runs \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: product-demo-1' \
  -d '{"input":{"description":"Matte blue ceramic coffee mug with a wooden handle"}}'
curl http://localhost:8080/runs/RUN_ID
```

PowerShell'de `curl.exe` kullanın. Hata: `{error:{code,message,request_id,findings?}}`. Bozuk JSON 400, validation 422, yok 404, conflict 409. Standalone validate geçerli JSON için 200 ve tüm bağımsız findings döndürür. Geçersiz create/update eski kaydı değiştirmez.

## Graf ve doğrulama

[JSON Schema](schemas/workflow.schema.json), [ürün workflow'u](examples/workflows/product.json), [etkinlik workflow'u](examples/workflows/event.json).

`schemaVersion:1`, node `id/kind/modelRef/params/promptTemplate`, edge `from/fromPort/to/toPort`. Bir input, bir output ve bağlı tek zincir; cycle, fan-out, fan-in, eksik/uyumsuz port, duplicate ID, unknown/disabled model ve çözülemeyen şablon değişkenleri reddedilir. Findings `code/message/path` ve gerektiğinde node_id taşır.

Portlar: input `text`; llm giriş `text` veya `prompt`, çıkış `prompt`; image giriş `prompt`, çıkış `image`; output bu üç tipten birini kabul eder. Input schema düz string/number/integer/boolean alanları tanımlar. Şablonlar `{{input.description}}` ve `{{nodes.prompt_writer.prompt}}`; yalnızca önceki node'ların scalar çıktılarını kullanır. Kod/fonksiyon değerlendirme yoktur. Image template yoksa bağlı prompt doğrudan kullanılır.

## Hata, retry ve bonuslar

- HTTP 408/504/524 → `PROVIDER_TIMEOUT`; 5xx/529 → `PROVIDER_5XX`; 429 → `PROVIDER_RATE_LIMIT`; bağlantı hatası → `PROVIDER_NETWORK_ERROR`. Yalnızca güvenli fazda geçici hatalar tekrar edilir: **en fazla 3 toplam node attempt**, 2 sn / 4 sn + 0–250 ms jitter; Retry-After gözetilir.
- Auth, ödeme, invalid input, model bulunamaması, içerik reddi ve bozuk completion retry edilmez. OpenRouter 200 refusal/content_filter, boş ve kesilmiş çıktı fal'a gönderilmez. Fal COMPLETED içindeki error başarı sayılmaz. Provider kodu/mesajı, HTTP status, failed node ve skipped node'lar saklanır.
- HTTP timeout 30 sn; kalıcı node deadline LLM 120 sn / image 600 sn; retry/restart süreyi sıfırlamaz. Fal'ın kendi retry ve model fallback'i kapalıdır. Run'ın River altyapı retry'si domain attempt'ten ayrıdır; terminal queue hatası uzlaştırılır.
- **Restart:** kayıtlı fal ID varsa aynı istek poll edilir; sonuç kaydedilmişse artifact fazı devam eder. Provider kabulü ile DB commit atomik olamaz: `sending` var fakat durable ID/cevap yoksa `PROVIDER_SUBMISSION_UNKNOWN`, kör resubmit yok. [Zorla process kapatma testi](internal/integration/integration_test.go) bu kritik commit sonrası aralığı doğrular.
- **Idempotency:** aynı key + workflow/input → aynı run (200); farklı body → 409. Key global scope ve run ömrü boyunca saklanır. Replay, güncel workflow doğrulamasından önce gelir; sonradan silinse bile eski run döner. 10 eşzamanlı istek için bir run/job testi vardır.
- **Maliyet:** request ledger ve fiyat snapshot'ı; NUMERIC(24,12) / decimal string. Başlangıç tahmini, çalıştırma tahmini ve gerçek ücret ayrı tutulur. Polling/restart ikinci maliyet üretmez; yeni üretim retry'si ayrı kayıt olur. Ayrıntılar aşağıda.

## Başlangıç tahmini ve gerçek maliyet

Dashboard girdi formunda, başlatmadan önce node'ların fiyatları ve **başlangıç tahmini toplam** görünür. `/estimate` ücretli üretim veya sağlayıcı fiyat çağrısı yapmaz: katalog fiyatı, seçilen görsel boyutu ve LLM için yaklaşık 4 karakter/token + 16 mesaj tokenı ile `max_completion_tokens` çıktı rezervi kullanır. Bu kesin bir üst sınır değildir; retry ve önbellek indirimleri tahmine dahil değildir. Eksik girdi/fiyat `null/partial` döner. Run kabul edilirken o anki snapshot'tan tekrar hesaplanan `initial_estimate` kalıcıdır; sonradan fiyat veya workflow değişince eski tahmin değişmez.

Run detayında her node ve sağlayıcı isteği için tahmini ve gerçek maliyet ayrı gösterilir. Çalıştırma tahmini gerçek token kullanımı / fal historical estimate ile güncellenebilir; başlangıç tahmini korunur. **Gerçek toplam**, tüm sağlayıcı isteklerinin ücretleri bilindiğinde `actual_cost_usd` olarak hesaplanır. Eksik ücret varken bu alan `null`, `actual_cost_status=pending/unavailable` olur; kısmi ücretler gerçek toplam diye gösterilmez. `reported_cost_usd` yalnızca eldeki raporların ara toplamıdır. Yerel input/output ücretsizdir; belirsiz gönderim veya ücreti doğrulanamayan istek sıfır sayılmaz.

OpenRouter `usage.cost` gerçek ücret olarak kaydedilir; yoksa aynı generation ID için [Generation API](https://openrouter.ai/docs/api/api-reference/generations/get-generation) `total_cost` okunur. Fal için [Billing Events API](https://fal.ai/docs/platform-apis/v1/models/billing-events) `source=billed`, `request_id` ve `endpoint_id` ile sorgulanır; indirim sonrası `cost_total` kullanılır. Liste sayfaları tamamlanmadan toplam kesinleştirilmez. Workspace toplamı, birim fiyat × billable units veya tahmin hiçbir zaman gerçek ücret yerine kullanılmaz. Sağlayıcının o isteğe ait ücret kaydı saklanır; bu tutar vergi içeren aylık fatura değildir.

Worker üretim tamamlandıktan sonra eksik ücretleri arka planda kontrol eder: 30 sn → 2 dk → 10 dk → 1 saat bekleme; en fazla 24 saat. Kontrol sayısı/zamanı, hata ve ücret yanıtı DB'de saklanır, restart devam eder. Billing hatası başarılı üretimi başarısız yapmaz. Yetki hatasında sürekli sorgu yerine “Doğrulanamadı” görünür; izin düzeltildikten sonra dashboard/API üzerinden maliyet yenilenebilir. Fal billing için `billing:usage:read`, `auth:keys:read`, `account:users:read` yetkileri gerekir (`BILLING` veya `FULL` preset). İstenirse `.env` içindeki **FAL_BILLING_KEY** ayrı tutulur; boşsa FAL_KEY kullanılır. Anahtarlar frontend'e gönderilmez.

```sh
curl -X POST http://localhost:8080/workflows/10000000-0000-4000-8000-000000000001/estimate \
  -H 'Content-Type: application/json' -d '{"input":{"description":"Blue ceramic cup"}}'
curl -X POST http://localhost:8080/runs/RUN_ID/costs/refresh
```

## İlk doğrudan akış → FlowKit

```sh
go run ./cmd/direct
# Daha önce fal ticket kaydedildiyse: go run ./cmd/direct -resume
```

Komut örnek workflow/model dosyalarını kullanır; her aşamayı ignored `evidence/local/direct.json` dosyasına kaydeder. Mevcut evidence dosyası varken normal komut yeni ücretli üretim başlatmaz. Anahtar yoksa çağrı yapılmadan durur.

11 Ekim 2026'da kullanıcı her anahtarlı çağrıyı ayrı onayladı; mevcut sağlayıcı adaptörleriyle dört çağrılık doğrudan deneme tamamlandı. `openai/gpt-6-luna` 58 input / 71 output token ve `usage.cost=0.0000413 USD` döndürdü. `fal-ai/flux/schnell` isteği `01a127ed-1583-7c50-8eb7-0688dddc4466`, COMPLETED ve 512×512 görsel döndürdü. [Anahtarsız kanıt kaydı](docs/evidence/direct.json), [görsel](https://v3b.fal.media/files/b/0aaddf91/LSonJO8IbWMVUppDPmUY7.jpg). Sonraki ayrı onaylı tek billing-events sorgusu 403 döndü: kullanılan anahtarın faturalama okuma yetkisi yok. Fal için 0.003 USD yalnızca tahmindir; fal gerçek ücret ve gerçek toplam null/unavailable. Bu standalone deney FlowKit DB run'ı değildir. Anahtarların bulunması izin sayılmaz; otomatik ek çağrı yapan demo komutları kapsamları ayrıca onaylanmadan çalıştırılmaz.

| Doğrudan deneme | FlowKit karşılığı |
|---|---|
| Ürün açıklaması | input.description / input node textField |
| English studio prompt talimatı | product.json prompt_writer.promptTemplate |
| `openai/gpt-6-luna`, reasoning none, max_completion_tokens 300 | prompt-writer katalog kaydı + llm params |
| LLM'den dönen prompt | prompt_writer.prompt → render.prompt |
| `fal-ai/flux/schnell`, 4 steps, 512×512, 1 image | flux-schnell katalog defaults + render.params |
| Queue ID / status / result | provider_requests kalıcı fazları |
| Görsel URL, gerçek model | artifact + run output/provenance |

İlk gerçek FlowKit run'ı `a733fb1e-686e-4d78-8386-09ce5953a36a` succeeded: birer LLM/image attempt, fal request `01a12b1f-c88d-73a2-85f1-cce572dc075b`, 512×512 JPEG (133147 byte), producer model `fal-ai/flux/schnell`. Artifact `/artifacts/70266b82-d304-4ef6-80c1-9696e044f5d1` API'den HTTP 200 ile okundu. Başlangıç tahmini 0.003157700000 USD korundu; çalıştırma tahmini 0.003039800000 USD, OpenRouter gerçek maliyeti 0.000039800000 USD. Fal actual ve actual total null/pending; billing çağrısı yapılmadı. [Canlı run/API kanıtı](docs/evidence/flowkit-approved.json).

Çağrı başına onaylı demoda insan bekleyişi ilk görsel deadline'ını aştı. Demo aracı düzeltildi; özgün deadline ve bekleme süresi DB'de korunarak aynı request ID'den, ek üretim göndermeden devam edildi. Bu düzeltme API/worker restart içerir ve runtime model ekleme kanıtı olarak sunulmaz. Yeni model demosunun baseline'ı ilk başarılı run sonrasında alındı.

Runtime model kabulünde `flux-dev` POST /models ile eklendi, mevcut ürün workflow'unun render.modelRef'i PUT ile değiştirildi (revision 2). Run `217fb931-3af8-4313-89b2-436985798449` succeeded: `fal-ai/flux/dev`, 28 steps, 512×512, birer LLM/image attempt ve tek submit. Artifact `f0f242b1-2b6d-4d1a-8925-6e136605d037` HTTP 200/JPEG/147577 byte. Baseline/sonuç Git `4e140e0`, dört servis container ID/StartedAt/image kimlikleri aynı; HTTP mutasyonları ve run snapshot'ları [canlı kanıtta](docs/evidence/flowkit-approved.json). Eski Schnell run'ı revision 1/model snapshot'ını korudu.

Dev başlangıç tahmini 0.025157700000 USD korundu. OpenRouter gerçek tutarı 0.000043300000; fal historical pricing sorgusu 0.050000000000 tahmin döndürdü, güncel run tahmini 0.050043300000. [Historical API price](https://fal.ai/docs/platform-apis/v1/models/pricing/estimate) geçmiş çağrı kullanımına dayanır; input boyutuna göre katalog fiyatı ve exact request billing tutarı ile aynı değildir. Public Dev fiyatı hâlâ [0.025 USD/üst megapiksel](https://fal.ai/models/fal-ai/flux/dev); tahmin farkı gerçek kesinti olarak yorumlanmaz. Fal actual ve gerçek toplam null/pending.

İkinci workflow etkinlik fikrinden dikey afiş üretir; `fal-ai/flux/dev`, 28 steps, 512×768. Ürün tanımı API'den Schnell'e geri alındı (revision 3), etkinlik workflow'u POST /workflows ile eklendi. Run `614b472e-e6c8-4e17-b07d-995f4307b321` succeeded: birer LLM/image attempt, actual OpenRouter 0.000071800000 USD, tek fal submit/request `01a12b5c-96ba-71e2-8aae-8d8b6b4bf933`. ID 764 ms içinde DB'de kalıcılaştı. Ayrı onaylı pricing/status/result sorguları tamamlandı; artifact `53c51066-5d9f-4c39-b758-15a831a44ac7`, 512×768 JPEG/549927 byte, API HTTP 200. Başlangıç tahmini 0.025158800000 korundu; çalıştırma tahmini 0.050071800000 USD. Fal actual ve gerçek toplam null/pending; billing checks 0. Dev kaydı ve etkinlik ekleme/run boyunca Git `4e140e0`, dört container'ın ID/StartedAt/image değerleri değişmedi. Uyumlu iki modelin farklı defaults'u katalog verisidir. Dashboard'da her iki gerçek çıktının producer modeli ve ayrı maliyet alanları kontrol edildi. [Tam API/DB/runtime kanıtı](docs/evidence/flowkit-approved.json).

PowerShell ile doğrudan denemeden sonra `./tools/live-demo.ps1`; manuel CLI: `go run ./cmd/acceptance -url http://localhost:8081 -project flowkit-acceptance`. Komut üç FlowKit run'ını (Schnell, model değişimi Dev, ikinci workflow Dev), bütün mutasyon HTTP istek/yanıtlarını ve önce/sonra container ID/StartedAt/image/Git SHA bilgisini `evidence/local/acceptance.json` dosyasına kaydeder. İlk çalıştırma boş ayrı DB ister. Yarıda kalırsa `-resume` aynı idempotency key'leri kullanır; tamamlanmış demo tekrar üretim yapmaz. Bütçe kontrolü doğrudan denemeyi de toplamda sayar ve sonraki çağrının üç denemesini rezerve eder.

## Yeni model / workflow ekleme

Boş kabul ortamında önce prompt-writer ve Schnell'i, ardından product workflow'unu API ile kaydedin. Yeni model için yalnızca:

```sh
curl -X POST http://localhost:8081/models -H 'Content-Type: application/json' \
  --data-binary @examples/models/flux-dev.json
# product.json içindeki render.modelRef'i flux-dev yapın; Schnell'e özgü step override eklemeyin.
curl -X PUT http://localhost:8081/workflows/10000000-0000-4000-8000-000000000001 \
  -H 'Content-Type: application/json' --data-binary @product-dev.json
curl -X POST http://localhost:8081/workflows -H 'Content-Type: application/json' \
  --data-binary @examples/workflows/event.json
```

Ardından run başlatın; çıktı `actual_model_id` yeni endpoint'i göstermelidir. Bu senaryo canlı tamamlandı: Dev katalog POST 201, workflow PUT 200/revision 2, run POST 202 ve yukarıdaki Dev run succeeded. Tam istek gövdeleri ve yanıtlar [kanıt JSON'unda](docs/evidence/flowkit-approved.json). İki nihai workflow farklı model kullansın diye ürün tanımı ardından Schnell'e geri alınır; geçmiş Dev run değişmez. `cmd/acceptance` toplu onaylı ortamda aynı senaryoyu kaydeder; çağrı başına onay için [kontrollü araç](docs/APPROVED_DEMO.md) kullanılır. Servis restart/deploy/kod değişikliği gerekmez. Destek sınırı: aynı `openrouter-chat-text-v1` veya `fal-text-to-image-v1` profili; yeni provider, image-list girdisi veya yeni parametre sözleşmesi adaptör/profile geliştirmesi gerektirir. Online profil `:batch` kabul etmez; eksik model tahmin edilmez.

## Test ve loglar

```sh
go test ./...
go vet ./...
docker compose -f compose.test.yaml up -d --wait
TEST_DATABASE_URL='postgres://flowkit:flowkit@localhost:55432/flowkit_test?sslmode=disable' \
  go test ./... -count=1 -timeout=6m
cd web && npm ci && npm test && npm run build
```

PowerShell: `$env:TEST_DATABASE_URL='postgres://flowkit:flowkit@localhost:55432/flowkit_test?sslmode=disable'`. Integration test DB adı `_test` ile bitmelidir; testler bu ayrı DB'yi temizler. Ortam değişkeni yoksa integration testler açıkça skip edilir. CI gerçek provider anahtarı içermez. sqlc 1.31.1 ile `sqlc generate`; generated kod repodadır ve CI fark kontrolü yapar.

CI ayrıca temiz Docker Compose kurulumu, örnek yükleme, gateway statik dosyası ve gerçek worker üzerinde sağlayıcısız `input → output` run'ını kontrol eder. Bu kontrol ücretli istek göndermez; kendi oluşturduğu test workflow'unu sonunda soft-delete eder. Yerelde çalışan temiz örnek ortam için `node tools/check-compose.mjs http://localhost:8080`. OpenAPI yeniden üretimi de CI'da fark kontrolüne dahildir.

`node tools/check-production.mjs` ayrı ve geçici bir Compose projesinde production override'ını kontrol eder: DB/API/dashboard/gate için kapalı host portları, varsayılan kapalı worker, HTTP → HTTPS yönlendirmesi, doğrulanan yerel CA, tek şifreli form, yetkisiz API/asset 401, yetkili dashboard/API erişimi, CSRF reddi, logout, sağlayıcısız asenkron run/idempotency, gate/proxy restart ve JSON log bağlamı. Sağlayıcı anahtarlarını boşaltır; yalnız kendi test volume'larını temizler. Gate unit testleri şifre/anahtar değişiminde eski oturum reddini, token tahrifini, expiry ve rate limit'i de kapsar. Bu kontrol canlı domain sertifikasının doğrulanmasının yerine geçmez.

API/worker/init/bootstrap/River uygulama logları JSONL: run_id, node_id, attempt (bağlam yoksa null), event, request/model/provider kimlikleri, süre, retry ve hata. DEBUG demo varsayılanıdır; `LOG_LEVEL` ile değişir. Provider hataları ERROR. Tam input/prompt/output DB izinde; credential/header/base64 loglanmaz. Nginx access log JSON; PostgreSQL motor ve Nginx native error logları altyapı loglarıdır. Docker log rotation 10 MB × 3.

## Model seçimi, bütçe ve AI kullanımı

Luna kısa metin için düşük maliyet; Schnell hızlı/ucuz; Dev aynı parametre sözleşmesiyle runtime model eklemeyi göstermek için seçildi. Dört gerçek zincir tamamlandı: direct Schnell, FlowKit Schnell, API ile Dev'e geçiş, ikinci workflow Dev. Her birinde tek LLM completion ve tek image submit var; FlowKit'te 15, direct dahil toplam 20 onaylı anahtarlı HTTP çağrısı (sorgular dahil). Katalog fiyatıyla dört görsel planı fal yaklaşık **$0.056**; canlı historical sorgular sonrasında fal tahmin toplamı **$0.106**. Doğrulanmış OpenRouter toplamı **$0.0001962**; fal gerçek toplamı ve birleşik gerçek toplam bilinmiyor. Sağlayıcı bakiye/sınır tüketimi billing yetkisi olmadan doğrulanmış sayılmaz. Fal $5 / OpenRouter $2 üst sınırları ve demo inceleme eşikleri $0.50 / $0.05 ayrı tutuldu; planlanan kontrol tutarları eşiklerin altında. Testler ve startup provider çağırmaz; test harcaması $0. Ek üretim/retry/sorgu yalnızca açık kullanıcı onayıyla yapılır. [Son bütçe ve çağrı özeti](docs/evidence/flowkit-approved.json) `final_budget_summary` alanında.

Codex planlama, uygulama, test ve dokümantasyonda kullanıldı. [AGENTS.md](AGENTS.md) değişmez kuralları, [AI_LOG.md](docs/AI_LOG.md) gerçek karar/hata/düzeltmeleri içerir. Örneğin ilk max_tokens örneği API dokümanında deprecated olduğu görülerek düzeltildi; River default checkpoint'i tek başına yeterli sanılmayıp DB commit sınırı incelendi. Kodun kritik transaction/retry/recovery davranışları mock + PostgreSQL ve gerçek worker process testleriyle doğrulandı.

Bir hafta daha olsaydı çoklu host için object storage, provider submission belirsizliğini azaltacak sağlayıcı destekli idempotency, otomatik fiyat güncelleme ve hesap ölçeğinde maliyet raporları, operasyonel metrics/traces ve genişletilebilir profile şemalarına öncelik verirdim. Yerel volume, tek host ve sağlayıcı ücret kaydının erişimine/zamanına bağımlılık bu teslimin bilinçli sınırlarıdır.
