# FlowKit

JSON olarak tanımlanan doğrusal AI workflow'larını kaydeden, doğrulayan ve arka planda çalıştıran küçük bir backend platformu. Go API + ayrı River worker + PostgreSQL; React dashboard yalnızca HTTP API kullanır.

**Durum:** uygulama, mock testler ve canlı üretim kabulü tamamlandı. Doğrudan OpenRouter → fal denemesi, FlowKit Schnell run'ı, API'den yeni Dev modeline geçiş ve ikinci workflow'un Dev run'ı başarılı. Yeni model/workflow eklerken kod değişmedi, servisler yeniden başlatılmadı, deploy yapılmadı. Fal billing okuması mevcut anahtarın yetkisi nedeniyle 403; bilinmeyen gerçek tutarlar null kalıyor. [Kabul kanıtları ve billing sınırı](docs/ACCEPTANCE.md). Çıktı kalitesi değerlendirmesine yönelik ek üretim yapılmaz.

Repository: [CanhK-2103/trypix-flowkitCase](https://github.com/CanhK-2103/trypix-flowkitCase). Gerçek çıktılarla dashboard: [ürün / Schnell](docs/dashboard-schnell-live.jpg), [etkinlik / Dev](docs/dashboard-event-live.jpg).

Case'i değerlendirmek için [paragraf paragraf orijinal metin, yapılan iş ve kontrol yolları](docs/CASE_PARAGRAPH_REVIEW.md) ile [22 adımlı teknik denetim](docs/CASE_AUDIT.md) bulunur. Orijinal `case.txt` değiştirilmeden korunmuştur; teknik test, önceki gerçek kabul kanıtı ve kişisel kod hâkimiyeti ayrı belirtilir.

## Gereksinim–uygulama–kontrol tablosu

Tablo, case'in temel şartlarını uygulamadaki karşılıkları ve inceleme yollarıyla eşleştirir. **Testli**, local mock/gerçek PostgreSQL testlerini; **gerçek kabul**, önceki açık onaylı OpenRouter/fal denemelerini ifade eder. Önceki üç gerçek run ve görselleri sunucuya taşınmıştır; cloud üzerinde yeniden ücretli üretim yapılmış gibi sunulmaz.

| Gereksinim | Uygulamada ne yaptık? | Nasıl kontrol edilir / durum |
|---|---|---|
| Veriyle tanımlanan workflow ve Go backend | Go API, ayrı River worker ve PostgreSQL; node/edge/model/prompt tanımları JSON verisi | **Tamamlandı.** [Mimari](#mimari-ve-çalıştırma), [motor](internal/engine/engine.go), [örnek workflow](examples/workflows/product.json) |
| HTTP API ve dashboard'un aynı API'yi kullanması | Katalog, workflow CRUD/validate, run başlatma/geçmiş/status/detail; dashboard HTTP istemcisi | **Testli.** [Endpoint tablosu](#api), [OpenAPI](docs/openapi.json), [CRUD testleri](internal/integration/integration_test.go); tarayıcı Network paneli |
| Katalog alias'ı ve kod/restart olmadan yeni model | Node modelRef katalog alias'ını kullanır; uyumlu Dev modeli API ile eklendi, ürün workflow'u güncellendi ve çalıştı | **Gerçek kabul + testli.** [API istek/yanıtları ve aynı runtime kimlikleri](docs/evidence/flowkit-approved.json), TestHotAddModelAndWorkflowWithoutRestart |
| JSON graf, doğrusal akış ve toplu doğrulama | input/llm/image/output, text/prompt/image; create/update döngü, dallanma, eksik port, tip, model ve template hatalarını bulgu listesiyle reddeder | **Testli.** [Şema](schemas/workflow.schema.json), [altı kural ve toplu bulgu testleri](internal/workflow/validate_test.go), [geçersiz write testi](internal/integration/validation_test.go) |
| Asenkron ve sıralı çalıştırma | HTTP 202/run_id; run/snapshot/River enqueue aynı transaction; ayrı worker önceki çıktıyı sonraki node'a aktarır | **Testli + gerçek kabul.** [API/transaction kodu](internal/api/server.go), TestAsyncWorkerSuccessRetryAndPermanentFailure; [ücretsiz cloud worker kontrolü](docs/evidence/deployment-fixes.json) |
| Fal queue ve hemen kalıcı request ID | Submit → status → result; ID/URL'ler poll/pricing/media öncesinde commit edilir | **Testli + gerçek kabul.** [Motor fazları](internal/engine/engine.go), TestKilledWorkerResumesDurableFalTicket; gerçek run provider_request_id izi |
| Tipli hata, retry ve kalıcı failure kaydı | Geçici hatalarda en fazla üç domain attempt; 2/4 sn+jitter; failed node/provider kodu/mesaj kalıcı, sonrakiler skipped | **Testli.** [Hata/retry politikası](#hata-retry-ve-bonuslar), [başarı/retry/kalıcı hata testleri](internal/engine/engine_test.go) |
| Doğrudan deneyin FlowKit'e taşınması | Gerçek OpenRouter → Schnell deneyi; aynı model/prompt/parametre karşılıkları README tablosunda | **Gerçek kabul.** [Doğrudan deney](docs/evidence/direct.json), [eşleştirme tablosu](#i%CC%87lk-doğrudan-akış--flowkit), ilk Schnell run |
| İki farklı workflow, farklı görsel modelleri | Ürün katalog görseli/Schnell ve etkinlik afişi/Dev; ikinci workflow yalnız API verisiyle eklendi | **Gerçek kabul.** [Ürün](examples/workflows/product.json), [etkinlik](examples/workflows/event.json), yayındaki iki workflow ve başarılı geçmişleri |
| Modelin çıktı yanında görünmesi ve adım izi | Gerçek producer model/provenance; girdi, çıktı, prompt, attempt, süre, request ID, maliyet ve hata | **Tamamlandı.** [Dashboard](https://flowkit.canhurkurtoglu.com/), `/runs/{id}`; [gerçek ekranlar](docs/dashboard-event-live.jpg) |
| JSON log ve error bağlamı | API/worker ve DB/Nginx/gate/proxy dahil bütün servislerde run_id/node_id/attempt; provider hata kod/mesajı ERROR seviyesinde | **Testli + cloud kontrolü.** [Telemetry testi](internal/telemetry/log_test.go), [native log normalizasyonu](cmd/logwrap/main.go), [sekiz servis log kanıtı](docs/evidence/deployment-fixes.json) |
| Dashboard'un temel ekranları | Workflow listesi, CodeMirror JSON editörü, Doğrula/Kaydet, şemadan girdi formu, run geçmişi/detayı | **Tamamlandı; frontend testli.** Yayındaki ekranlar; [form testleri](web/src/WorkflowInput.test.tsx), `npm test` ve `npm run build` |
| Hata yolları dahil mock testler | Sabit completion/görsel, timeout/5xx/retry/permanent hata, CRUD, gerçek PostgreSQL/River/process testleri; local mock pricing/media | **Testli.** [Test komutları](#test-ve-loglar), [GitHub Actions](https://github.com/CanhK-2103/trypix-flowkitCase/actions); integration testleri skip olmamalı |
| Tek komut kurulum, env örneği ve GitHub | `docker compose up`; migration/bootstrap; kalıcı DB/media volume; public repo ve `.env.example` | **Temiz kurulum testli.** [Compose](compose.yaml), [.env.example](.env.example), CI compose işi; aşağıdaki çalıştırma adımları |
| Opsiyonel parola korumalı deployment | HTTPS ve tek şifreli gate; dashboard/API/artifact korumalı; backend portları private | **Yapıldı, public erişim kontrol edildi.** [Site](https://flowkit.canhurkurtoglu.com/), [deployment yönergesi](deploy/README.md), [TLS/auth kanıtı](docs/evidence/deployment.json); site şifresi ayrıca paylaşılır |
| Bonus: worker restart dayanıklılığı | Bilinen fal ID'sine devam; yeni submit yok. ID'siz belirsiz gönderim güvenli fail | **Process-kill/mock/PostgreSQL testli.** TestKilledWorkerResumesDurableFalTicket; yeni ücretli fal işi sırasında VPS kill demosu yapılmadı |
| Bonus: idempotency | Aynı key/body tek run/job; farklı body 409; tarihsel snapshot ve replay korunur | **Testli + ücretsiz cloud replay.** TestConcurrentIdempotencyAndSnapshot; [aynı ID/200 kanıtı](docs/evidence/deployment-fixes.json) |
| Bonus: tahmini node/toplam maliyeti | Başlamadan node/toplam tahmin; immutable initial_estimate; exact decimal; run tahmini ve gerçek ücret ayrı | **Tamamlandı, testli.** [Maliyet açıklaması](#başlangıç-tahmini-ve-gerçek-maliyet), [quote/billing testleri](internal/engine/billing_test.go); run maliyet sütunları |
| Bütçe ve model gerekçesi | Az/onaylı gerçek çağrı, küçük çözünürlük, testlerde sıfır provider harcaması; tahmin ve actual ayrımı | **Kayıtlı.** [Model/bütçe bölümü](#model-seçimi-bütçe-ve-ai-kullanımı), kanıttaki final_budget_summary; fal gerçek bakiye doğrulanmış değil |
| AI şeffaflığı, kararlar ve gelecek planı | AGENTS.md, gerçek hata/düzeltme günlüğü, README gerekçeleri ve son tek paragraf | **Dokümanlar mevcut.** [AGENTS.md](AGENTS.md), [AI_LOG.md](docs/AI_LOG.md), README son bölümü; adayın kod hâkimiyeti görüşmede değerlendirilir |

**Gerçek ücret sınırı:** Case'in tahmini maliyet bonusu karşılanmıştır. Ek gerçek-maliyet alanları ve gecikmeli accounting davranışı mock testlerle doğrulanmıştır; önceki OpenRouter yanıtlarında gerçek `usage.cost` kaydedilmiştir. Fal anahtarının billing okuma yetkisi olmadığından, kullanıcının kararıyla cloud'da `BILLING_DISABLED=true` tutulur. Fal actual ve tüm ücretler bilinmeyen gerçek toplam `null` kalır; tahmin gerçek kesinti gibi gösterilmez.

## Önerilen inceleme sırası

1. **Kurulumu ve mimariyi okuyun.** Aşağıdaki [Çalıştırma](#çalıştırma), [Mimari](#mimari-ve-çalıştırma) ve [API](#api) bölümlerinden başlayın. Temiz yerel ortamda anahtarlar olmadan servis/editör açılabilir; startup ücretli üretim yapmaz.
2. **Yayındaki dashboard'u açın.** [flowkit.canhurkurtoglu.com](https://flowkit.canhurkurtoglu.com/) için ayrıca iletilen site şifresiyle giriş yapın. Ürün ve etkinlik workflow'larının JSON tanımlarını, farklı modelRef'lerini ve mevcut run geçmişlerini inceleyin.
3. **Mevcut gerçek sonuçları kontrol edin.** [İlk Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a), [model değişimi Dev run](https://flowkit.canhurkurtoglu.com/runs/217fb931-3af8-4313-89b2-436985798449) ve [ikinci workflow Dev run](https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321) JSON'unu giriş yaptıktan sonra açın. Dashboard'da node detayları, görsel/gerçek model, deneme/süre ve ayrı maliyet alanlarıyla karşılaştırın; eski Dev run güncel ürün tanımı Schnell'e dönse de Dev snapshot'ını korur.
4. **Doğrulamayı üretim yapmadan deneyin.** JSON editöründeki taslakta bilinmeyen modelRef, çözülemeyen template ve eksik edge oluşturup Doğrula kullanın; birden çok bulgu beklenir. Teslim workflow'unu kaydetmeden sayfayı yenileyin. CRUD/no-write ve hata yolları otomatik testlerde incelenebilir.
5. **Esneklik ve ilk akış kanıtını inceleyin.** [direct.json](docs/evidence/direct.json) ile [eşleştirme tablosunu](#i%CC%87lk-doğrudan-akış--flowkit); ardından [flowkit-approved.json](docs/evidence/flowkit-approved.json) içindeki gerçek model/workflow mutasyonlarını ve runtime_before/runtime_after_all kimliklerini karşılaştırın. Yeni model ve ikinci workflow kabulü boyunca kod/container kimlikleri aynıdır.
6. **Testleri ve bonusları doğrulayın.** [Actions](https://github.com/CanhK-2103/trypix-flowkitCase/actions) backend/dashboard/compose işlerini ve [test komutlarını](#test-ve-loglar) inceleyin. PostgreSQL integration testleri için yalnız ayrı `_test` DB kullanılır. Restart, concurrent idempotency, retry/permanent failure ve maliyet testleri local mock sağlayıcılarla çalışır.
7. **Sınırları ve teslim açıklamalarını okuyun.** [Maliyet](#başlangıç-tahmini-ve-gerçek-maliyet), [model/bütçe/AI](#model-seçimi-bütçe-ve-ai-kullanımı) ve [deployment](deploy/README.md) bölümlerine bakın. Madde bazında daha ayrıntılı kontrol için [paragraf raporu](docs/CASE_PARAGRAPH_REVIEW.md), [22 adımlı denetim](docs/CASE_AUDIT.md) ve [cloud taşıma/kontrol kanıtı](docs/evidence/deployment-fixes.json) bulunur.

Bu sıradaki mevcut kayıtları okumak ve mock testleri çalıştırmak yeni ücretli üretim gerektirmez. Gerçek provider demo komutları/yeniden run başlatma ayrıca onaylanan kullanımlardır. Teknik kanıtlar adayın her kod bölümünü açıklama sorumluluğunu ortadan kaldırmaz; paragraf raporunun E5 bölümü bu görüşme için kontrol soruları içerir.

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

Parola korumalı yönetim paneli: **https://flowkit.canhurkurtoglu.com**. Cloudflare DNS, public Let’s Encrypt sertifikası/hostname doğrulaması, HTTPS yönlendirmesi, tek şifreli login, oturumsuz API/asset 401, yetkili dashboard/API 200, CSRF reddi/logout [deployment kanıtında](docs/evidence/deployment.json). [Giriş ekranı](docs/flowkit-login.jpg) yayından alınmıştır. API/worker/DB/dashboard aktiftir, provider anahtarları ignored server `.env` içinde kuruludur. **BILLING_DISABLED=true**: otomatik accounting sorguları kapalıdır. İki workflow ve önceki onaylı üç gerçek run/görsel sunucudadır; bunlar yerel kabul DB'sinden tarih/snapshot/ücret bilgileri korunarak taşındı, cloud'da yeniden üretilmedi. [Tamamlama kanıtı](docs/evidence/deployment-fixes.json) üç görselin SHA-256 eşitliğini, ortak metadata/graf bulgularını ve sağlayıcısız worker/idempotency kontrolünü gösterir. Yeni ücretli cloud üretimi/billing çağrısı yapılmadı. Değerlendirici site şifresi ayrıca paylaşılır, repoda bulunmaz.

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
| GET | `/capabilities` | Ortamda sağlayıcı faturalama sorgularının açık olup olmadığı |

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

Workflow ID verilirse UUID olmalıdır; standalone validate, create ve update ortak metadata/graf bulgularını döndürür. Update body/path ID uyuşmazlığı diğer bulguları gizlemez. Dashboard girdi formu seçilen workflow'un şemasından üretilir; farklı alan isimleri, boolean/numeric değerleri ve enum seçenekleri desteklenir. Workflow/revision/şema değişiminde önceki alanlar gönderilmez.

Portlar: input `text`; llm giriş `text` veya `prompt`, çıkış `prompt`; image giriş `prompt`, çıkış `image`; output bu üç tipten birini kabul eder. Input schema düz string/number/integer/boolean alanları tanımlar. Şablonlar `{{input.description}}` ve `{{nodes.prompt_writer.prompt}}`; yalnızca önceki node'ların scalar çıktılarını kullanır. Kod/fonksiyon değerlendirme yoktur. Image template yoksa bağlı prompt doğrudan kullanılır.

## Hata, retry ve bonuslar

- HTTP 408/504/524 → `PROVIDER_TIMEOUT`; 5xx/529 → `PROVIDER_5XX`; 429 → `PROVIDER_RATE_LIMIT`; bağlantı hatası → `PROVIDER_NETWORK_ERROR`. Yalnızca güvenli fazda geçici hatalar tekrar edilir: **en fazla 3 toplam node attempt**, 2 sn / 4 sn + 0–250 ms jitter; Retry-After gözetilir.
- Auth, ödeme, invalid input, model bulunamaması, içerik reddi ve bozuk completion retry edilmez. OpenRouter 200 refusal/content_filter, boş ve kesilmiş çıktı fal'a gönderilmez. Fal COMPLETED içindeki error başarı sayılmaz. Provider kodu/mesajı, HTTP status, failed node ve skipped node'lar saklanır.
- HTTP timeout 30 sn; kalıcı node deadline LLM 120 sn / image 600 sn; retry/restart süreyi sıfırlamaz. Fal'ın kendi retry ve model fallback'i kapalıdır. Run'ın River altyapı retry'si domain attempt'ten ayrıdır; terminal queue hatası uzlaştırılır.
- **Restart:** kayıtlı fal ID varsa aynı istek poll edilir; sonuç kaydedilmişse artifact fazı devam eder. Provider kabulü ile DB commit atomik olamaz: `sending` var fakat durable ID/cevap yoksa `PROVIDER_SUBMISSION_UNKNOWN`, kör resubmit yok. [Zorla process kapatma testi](internal/integration/integration_test.go) bu kritik commit sonrası aralığı doğrular.
- **Idempotency:** aynı key + workflow/input → aynı run (200); farklı body → 409. Key global scope ve run ömrü boyunca saklanır. Replay, güncel workflow doğrulamasından önce gelir; sonradan silinse bile eski run döner. 10 eşzamanlı istek için bir run/job testi vardır.
- **Maliyet:** request ledger ve fiyat snapshot'ı; NUMERIC(24,12) / decimal string. Başlangıç tahmini, çalıştırma tahmini ve gerçek ücret ayrı tutulur. Polling/restart ikinci maliyet üretmez; yeni üretim retry'si ayrı kayıt olur. Ayrıntılar aşağıda.

## Başlangıç tahmini ve gerçek maliyet

Görsel node'larında isteğe bağlı **birincil → yedek model** sırası da desteklenir. Örneğin etkinlik afişinin görsel node'u şu alanlarla yapılandırılabilir:

```json
{
  "id": "poster",
  "kind": "image",
  "modelRef": "flux-krea",
  "fallbackModelRef": "flux-dev",
  "params": {"image_size": {"width": 512, "height": 768}, "num_images": 1},
  "promptTemplate": "{{nodes.art_director.prompt}}"
}
```

`modelRef` birincil, `fallbackModelRef` yedek katalog alias'ıdır. İkisi de kayıtlı/aktif, farklı alias'lar ve aynı provider/image profile ile uyumlu olmalıdır; create/update/run kabulünde ikisi de doğrulanır ve snapshot alınır. Temiz kurulum örneği Dev ile çalışmaya devam eder; Krea gibi ek model önce katalog API'sine kaydedilir, ardından mevcut workflow yalnız API/JSON ile güncellenir.

İlk **güvenli geçici üretim hatasında** sonraki deneme yedek modele geçer. İki model birlikte **en fazla 3 toplam node attempt** kullanır; 2/4 saniye+jitter/Retry-After ve mevcut node deadline korunur. Kalıcı input/auth/payment/model/content hatalarında veya belirsiz submission'da geçiş yapılmaz. Bilinen, sürmekte olan fal ID'sinin status/result hatası aynı ticket'la tekrar edilir; yalnız kesin terminal üretim hatası geçici sınıftaysa yeni model üretimine geçilebilir. Sağlayıcının gizli retry/fallback header'ları kapalı kalır; bu geçiş FlowKit'in açık workflow verisi ve log/iz kaydıdır.

Run'da immutable `model_snapshot` ve `fallback_model_snapshot`, seçimi gösteren `active_model_ref`, geçiş nedeni/zamanı `fallback` ve her provider request'in kendi model/ücret kaydı tutulur. Dashboard node detayında birincil/yedek ve geçiş nedeni görünür; görselin producer modeli gerçekten kullanılan request'ten gelir. Başlangıç tahmini birincil modelin tek üretim tahminidir; yedek geçişi/ek denemeler dahil değildir. İlk modelin bilinmeyen ücreti sıfırlanmaz; gerçek toplam bütün request ücretleri bilinince hesaplanır. [Motor/restart/üç deneme testleri](internal/engine/fallback_test.go), [validation](internal/workflow/fallback_test.go), [PostgreSQL/HTTP/worker testi](internal/integration/fallback_test.go).

15 Ekim'de canlı etkinlik afişi revision 4 için Krea → Dev ayarı yalnız API ile kaydedildi. [Deployment kaydı](docs/evidence/model-fallback-deployment.json) canlı konfigürasyon/sağlık kontrolünü ve mock failover testlerini ayrı gösterir; bu kontrolde gerçek sağlayıcı çağrısı veya yeni run oluşturulmadı.

Billing okuma yetkisi olmayan ortamda `BILLING_DISABLED=true` kullanılır. Worker otomatik ücret sorgusu yapmaz; maliyet yenileme endpoint'i `409/BILLING_DISABLED` döndürür, dashboard yenileme butonunu göstermez. Üretim, katalog tahmini ve OpenRouter'ın completion yanıtında bildirdiği gerçek ücret çalışır. Bilinmeyen fal/gerçek toplam null kalır; bu ayar üretim sağlayıcısının pricing sorgusunu veya normal queue polling'ini kapatmaz.

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

API/worker/init/bootstrap/River ve bütün deployment container logları JSONL: run_id, node_id, attempt (bağlam yoksa null), event, request/model/provider kimlikleri, süre, retry ve hata. DEBUG demo varsayılanıdır; `LOG_LEVEL` ile değişir. Provider hataları ERROR. Tam input/prompt/output DB izinde; credential/header/base64 loglanmaz. PostgreSQL motor, Nginx startup/native error/access ve Caddy native logları da JSON sözleşmesine uyar. `cmd/logwrap` resmî DB/Nginx entrypoint'lerini çalıştırır, stdout/stderr satırlarını normalize eder, credential değerlerini maskeler ve PostgreSQL SIGINT/Nginx SIGQUIT dahil stop sinyalini child'a iletir. DB/dashboard Docker target'ları ana Dockerfile'dadır. Docker log rotation 10 MB × 3.

## Model seçimi, bütçe ve AI kullanımı

Luna kısa metin için düşük maliyet; Schnell hızlı/ucuz; Dev aynı parametre sözleşmesiyle runtime model eklemeyi göstermek için seçildi. Dört gerçek zincir tamamlandı: direct Schnell, FlowKit Schnell, API ile Dev'e geçiş, ikinci workflow Dev. Her birinde tek LLM completion ve tek image submit var; FlowKit'te 15, direct dahil toplam 20 onaylı anahtarlı HTTP çağrısı (sorgular dahil). Katalog fiyatıyla dört görsel planı fal yaklaşık **$0.056**; canlı historical sorgular sonrasında fal tahmin toplamı **$0.106**. Doğrulanmış OpenRouter toplamı **$0.0001962**; fal gerçek toplamı ve birleşik gerçek toplam bilinmiyor. Sağlayıcı bakiye/sınır tüketimi billing yetkisi olmadan doğrulanmış sayılmaz. Fal $5 / OpenRouter $2 üst sınırları ve demo inceleme eşikleri $0.50 / $0.05 ayrı tutuldu; planlanan kontrol tutarları eşiklerin altında. Testler ve startup provider çağırmaz; test harcaması $0. Ek üretim/retry/sorgu yalnızca açık kullanıcı onayıyla yapılır. [Son bütçe ve çağrı özeti](docs/evidence/flowkit-approved.json) `final_budget_summary` alanında.

Codex planlama, uygulama, test ve dokümantasyonda kullanıldı. [AGENTS.md](AGENTS.md) değişmez kuralları, [AI_LOG.md](docs/AI_LOG.md) gerçek karar/hata/düzeltmeleri içerir. Örneğin ilk max_tokens örneği API dokümanında deprecated olduğu görülerek düzeltildi; River default checkpoint'i tek başına yeterli sanılmayıp DB commit sınırı incelendi. Kodun kritik transaction/retry/recovery davranışları mock + PostgreSQL ve gerçek worker process testleriyle doğrulandı.

Bir hafta daha olsaydı çoklu host için object storage, provider submission belirsizliğini azaltacak sağlayıcı destekli idempotency, otomatik fiyat güncelleme ve hesap ölçeğinde maliyet raporları, operasyonel metrics/traces ve genişletilebilir profile şemalarına öncelik verirdim. Yerel volume, tek host ve sağlayıcı ücret kaydının erişimine/zamanına bağımlılık bu teslimin bilinçli sınırlarıdır.
