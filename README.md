# FlowKit

JSON olarak tanımlanan doğrusal AI workflow'larını kaydeden, doğrulayan ve arka planda çalıştıran küçük bir backend platformu. Go API + ayrı River worker + PostgreSQL; React dashboard yalnızca HTTP API kullanır.

**Durum:** uygulama ve mock testler mevcut. Gerçek sağlayıcı kabul denemeleri yerel API anahtarları bekliyor; [kanıt ve kalan kontroller](docs/ACCEPTANCE.md). Çıktı kalitesi değerlendirmesine yönelik ek üretim yapılmaz.

Repository: [CanhK-2103/trypix-flowkitCase](https://github.com/CanhK-2103/trypix-flowkitCase). [Dashboard görünümü](docs/dashboard.jpg).

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

Normal kurulumu yeniden başlatmanız veya verilerini silmeniz gerekmez. Bulut deploy bu teslimin parçası değildir; yerel gateway yalnızca 127.0.0.1'e bağlıdır.

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

TanStack Query aktif run detayını 2 saniyede bir yeniler, terminal durumda durur. Mutation otomatik retry yapmaz. CodeMirror sözdizimi hatalarını gösterir; semantik kurallar her kaydetmede backend'de uygulanır.

## API

| Method | Endpoint | İşlem |
|---|---|---|
| POST / GET | `/models` | Model kaydı / katalog |
| GET / PATCH | `/models/{alias}` | Detay / defaults, pricing, enabled güncelleme |
| POST / GET | `/workflows` | Doğrulanmış oluşturma / liste |
| GET / PUT / DELETE | `/workflows/{id}` | Detay / doğrulanmış güncelleme / soft-delete |
| POST | `/workflows/validate` | Kaydetmeden `{valid, findings}` |
| POST | `/workflows/{id}/runs` | Asenkron run; optional Idempotency-Key |
| GET | `/runs` | Geçmiş; workflow_id/status filtreleri |
| GET | `/runs/{id}/status` | Hafif durum |
| GET | `/runs/{id}` | Snapshot, node/attempt izi, hata, çıktı, maliyet |
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
- **Maliyet:** request ledger ve fiyat snapshot'ı; NUMERIC(24,12) / decimal string. OpenRouter token kullanımından tahmin ve `usage.cost` ayrı saklanır; upstream cost eklenmez. Fal historical estimate API bir endpoint / bir çağrı tahmini verir; alınamazsa katalog fiyatı ve gerçek boyutla hesap yapılır. Bilinmeyen tutar null/partial; bilinen ara toplam gösterilir. Polling/restart ikinci maliyet üretmez; yeni gerçek üretim retry'si ayrı kayıt olur. Fal tutarı fatura değil tahmindir.

## İlk doğrudan akış → FlowKit

```sh
go run ./cmd/direct
# Daha önce fal ticket kaydedildiyse: go run ./cmd/direct -resume
```

Komut örnek workflow/model dosyalarını kullanır; her aşamayı ignored `evidence/local/direct.json` dosyasına kaydeder. Mevcut evidence dosyası varken normal komut yeni ücretli üretim başlatmaz. Anahtar yoksa çağrı yapılmadan durur.

| Doğrudan deneme | FlowKit karşılığı |
|---|---|
| Ürün açıklaması | input.description / input node textField |
| English studio prompt talimatı | product.json prompt_writer.promptTemplate |
| `openai/gpt-6-luna`, reasoning none, max_completion_tokens 300 | prompt-writer katalog kaydı + llm params |
| LLM'den dönen prompt | prompt_writer.prompt → render.prompt |
| `fal-ai/flux/schnell`, 4 steps, 512×512, 1 image | flux-schnell katalog defaults + render.params |
| Queue ID / status / result | provider_requests kalıcı fazları |
| Görsel URL, gerçek model | artifact + run output/provenance |

İkinci workflow etkinlik fikrinden dikey afiş üretir; `fal-ai/flux/dev`, 28 steps, 512×768. Uyumlu iki modelin farklı defaults'u katalog verisidir. Gerçek örnek run kimlikleri ve harcamalar henüz üretilmedi; [kabul durumu](docs/ACCEPTANCE.md).

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

Ardından run başlatın; çıktı `actual_model_id` yeni endpoint'i göstermelidir. `cmd/acceptance` bu senaryonun HTTP isteklerini/run sonuçlarını kaydeder. Servis restart/deploy/kod değişikliği gerekmez. Destek sınırı: aynı `openrouter-chat-text-v1` veya `fal-text-to-image-v1` profili; yeni provider, image-list girdisi veya yeni parametre sözleşmesi adaptör/profile geliştirmesi gerektirir. Online profil `:batch` kabul etmez; eksik model tahmin edilmez.

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

API/worker/init/bootstrap/River uygulama logları JSONL: run_id, node_id, attempt (bağlam yoksa null), event, request/model/provider kimlikleri, süre, retry ve hata. DEBUG demo varsayılanıdır; `LOG_LEVEL` ile değişir. Provider hataları ERROR. Tam input/prompt/output DB izinde; credential/header/base64 loglanmaz. Nginx access log JSON; PostgreSQL motor ve Nginx native error logları altyapı loglarıdır. Docker log rotation 10 MB × 3.

## Model seçimi, bütçe ve AI kullanımı

Luna kısa metin için düşük maliyet; Schnell hızlı/ucuz; Dev aynı parametre sözleşmesiyle runtime model eklemeyi göstermek için seçildi. Dört temel gerçek zincir planlandı: direct Schnell, FlowKit Schnell, API ile Dev'e geçiş, ikinci workflow Dev. Public fiyatlarla fal yaklaşık **$0.056**; gerçek token kullanımına göre OpenRouter maliyeti ayrı kaydedilecek. Fal $5 / OpenRouter $2 üst sınırları; demo inceleme eşikleri $0.50 / $0.05. Testler ve startup provider çağırmaz; bugüne kadarki test harcaması $0. Gerçek fiyat ve hesap erişimi direct denemede tekrar kontrol edilir.

Codex planlama, uygulama, test ve dokümantasyonda kullanıldı. [AGENTS.md](AGENTS.md) değişmez kuralları, [AI_LOG.md](docs/AI_LOG.md) gerçek karar/hata/düzeltmeleri içerir. Örneğin ilk max_tokens örneği API dokümanında deprecated olduğu görülerek düzeltildi; River default checkpoint'i tek başına yeterli sanılmayıp DB commit sınırı incelendi. Kodun kritik transaction/retry/recovery davranışları mock + PostgreSQL ve gerçek worker process testleriyle doğrulandı.

Bir hafta daha olsaydı çoklu host için object storage, provider submission belirsizliğini azaltacak sağlayıcı destekli idempotency/ücret mutabakatı, operasyonel metrics/traces ve genişletilebilir profile şemalarına öncelik verirdim. Yerel volume, tek host ve kapsamı sınırlı maliyet tahmini bu teslimin bilinçli sınırlarıdır.
