# FlowKit case denetimi — 12 Ekim 2026

Referans: kullanıcının `C:\Users\PC\Downloads\case.txt` dosyası. İncelenen GitHub `main` ve VPS checkout sürümü: `71208e6e5b2adfcb055263f0d99948b86359f9ae`. Bu rapor uygulama kodunu veya çalışan sunucu konfigürasyonunu değiştirmez. Denetimde OpenRouter/fal üretim, fiyat, polling veya billing çağrısı yapılmadı.

**İlk denetimin sonucu (13:07):** çekirdek backend, zorunlu mock testler, gerçek sağlayıcı kabul kanıtları ve üç bonus mevcut; iki uygulama hatası, bir log kapsamı farkı, eski dokümantasyon ifadeleri ve tamamlanmamış canlı deployment davranışı bulundu. Aşağıdaki bulgular o denetimin tarihsel kaydıdır. Kullanıcı aynı gün F/billing yetkisi dışındaki maddelerin tamamlanmasını istedi.

**Düzeltme aşaması:** A/B için form ve PostgreSQL/API regression testleri eklendi; D için resmî DB/Nginx entrypoint'leri/sinyalleri korunarak bütün servis logları normalize edildi. Standart Compose ve production smoke kontrolleri başarılı; frontend 13 test/typecheck/build geçti. E'deki eski doküman ifadeleri düzeltildi. C için üç başarılı gerçek run/medya yerel ortamdan dışa aktarıldı; deployment tamamlandığında sonuç aşağıya kaydedilir. F kullanıcı talebiyle dış doğrulama kapsamı dışındadır; yeni billing sorgusu yapılmaz.

## 1. Doğrulanan güncel durum

- Repository public: https://github.com/CanhK-2103/trypix-flowkitCase. `main` ile `/opt/flowkit` checkout HEAD aynı; server çalışma ağacında değişiklik yok.
- Güncel sürümün [CI çalışması](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38170369875): backend, dashboard ve compose job'larının tamamı başarılı. Backend job'ında gerçek PostgreSQL integration suite, gofmt/vet/sqlc; dashboard job'ında frontend test/typecheck/build ve OpenAPI regeneration; compose job'ında temiz kurulum ve production smoke bulunuyor.
- Denetimde workflow/engine/providers/telemetry unit testleri yeniden çalıştırıldı ve geçti. Frontend: 3 dosya / 9 test geçti. Bu testler yerel mock yanıtlar kullanır.
- Public HTTPS/login/logout/CSRF/oturumsuz API ve asset koruması 12 Ekim 2026, yaklaşık 13:07 Türkiye saatinde yeniden doğrulandı. HTTP 308 → HTTPS; dashboard oturumsuz 303 → login; API/asset oturumsuz 401; doğru şifre sonrası 200; farklı origin mutation 403; logout sonrası 401. TLS sistem trust store ve hostname ile doğrulandı.
- Cloudflare A kaydı `flowkit.canhurkurtoglu.com → 185.237.14.198` public resolver üzerinden doğrulandı.
- VPS: db/API/gate healthy; dashboard/proxy running; init/bootstrap başarılı tamamlanmış. Host üzerinde yalnız proxy HTTP/HTTPS portları yayınlanıyor. **Worker kapalı, provider anahtarları boş.**
- VPS PostgreSQL: **3 aktif model, 2 workflow, 0 run, 0 artifact, 0 provider request, 0 River job**. İki workflow: ürün/Schnell ve etkinlik/Dev.
- Denetimin ilk bölümünde yerel kabul DB'sindeki üç gerçek başarılı run ve model/artifact kimlikleri doğrudan SQL ile doğrulandı. 12 Ekim'de devam edilirken yerel Docker Desktop kapalıydı; integration/Compose suite'i bu devam bölümünde tekrar çalıştırılmadı. Güncel CI bunları çalıştırmış ve geçmiş durumda. Docker yeniden başlatılmadı; onaysız arka plan provider çağrısı başlatılmadı.

## 2. Açık bulgular

### A. Dashboard, `description` dışındaki girdi alanlarında run başlatamıyor

Kaynak: [web/src/main.tsx](../web/src/main.tsx), `values` başlangıcı ve workflow değişiminde `setValues({description: ""})`.

Sadece `message` kabul eden, `additionalProperties:false` içeren geçerli bir `input → output` workflow'u yerel API'den kaydedildi. Dashboard'da `message` doldurulup gönderildiğinde görünmeyen `description:""` alanı da request'e girdi. API **422 / INVALID_INPUT / additional properties 'description' not allowed** döndürdü. Bu senaryo tarayıcıda da tekrarlandı; AI node'u yoktu, provider çağrısı yapılmadı.

İki hazır örnek `description` kullandığı için çalışıyor. Hata yeni workflow'ların dashboard girdi formundaki esnekliğini etkiliyor; API'nin farklı şemaları desteklemesini engellemiyor. Aynı form state yaklaşımı, hiç dokunulmamış zorunlu boolean alanın `false` değerinin gönderilmesini de güvenceye almıyor; bu ikinci durum kaynak incelemesidir, ayrı tarayıcı denemesi yapılmadı.

Gerekli düzeltme: form state'ini seçilen workflow'un input schema'sından kurmak; sadece tanımlı alanları göndermek; workflow/revision değişiminde uyumlu biçimde sıfırlamak. Farklı alan adı ve boolean için form regression testi eklemek.

Yerel ekran kanıtı: `artifacts/audit-dashboard-input.jpg` (ignored). Geçici workflow ID: `2533d8e0-1fa6-43d7-b098-471d0a28a7e2`. Denetim devamında Docker kapalı olduğu için bu test kaydının soft-delete temizliği tamamlanamadı; sunucuda veya GitHub'da bu kayıt yok. Hiçbir başarılı run oluşturulmadı.

### B. Workflow ID hataları bulgu listesi sözleşmesini bozuyor

Kaynak: [internal/api/server.go](../internal/api/server.go), `createWorkflow`, `updateWorkflow`, `validateMetadata`.

Geçerli graf + `id:"invalid-uuid"`:

```json
{"validate_http":200,"validate_body":{"valid":true,"findings":[]},"create_http":422,"create_error":{"code":"VALIDATION_FAILED","message":"ID must be UUID"}}
```

Create yanıtında `findings` yok. Aynı ID yanında bilinmeyen alias, eksik bağlantılar ve çözülemeyen template varsa standalone validate 9 graf bulgusu döndürüyor; create sadece ID hatasıyla erken dönüp diğer bulguları göstermiyor. Update'te body ID/path ID uyuşmazlığı da aynı erken dönüş yaklaşımını kullanıyor (kaynak incelemesi).

Altı zorunlu graf kuralı uygulanmış ve test edilmiş durumda; bulgu ID/meta doğrulamasının endpoint'ler arasında tutarlı olmamasıdır. Gerekli düzeltme: metadata/ID bulgularını ortak doğrulamaya dahil etmek, bağımsız graf bulgularıyla birleştirmek; kaydetmeden doğrula ve create/update sonuçlarını regression testinde karşılaştırmak.

Yerel anahtarsız HTTP kanıtı: `artifacts/audit-api-contract.json` (ignored).

### C. Canlı site yönetim için hazır; üretim demosu aktif değil

https://flowkit.canhurkurtoglu.com çalışıyor ve parola korumalı. Fakat API/worker provider anahtarları boş ve worker kapalı. Bu yüzden gerçek AI run'ı başlayamaz; model node'u içeren run isteği `PROVIDER_NOT_CONFIGURED` ile reddedilir. Bu sonuç yapılandırma ve API guard'ı incelenerek doğrulandı; denetimde gerçek run başlatılmadı.

Önceki üç gerçek run ve görseller cloud DB/volume'larına aktarılmadı. Değerlendirici cloud'da run geçmişi/görsel sonuçları göremez. Case'te deploy opsiyoneldir; repository'deki gerçek kabul kanıtları ve çalışan standart Compose uygulaması ayrı olarak mevcut. Ancak bu URL'yi tam üretim demosu olarak teslim etmek için bu durumun açıkça çözülmesi gerekir. Worker/anahtarları etkinleştirmek, yeni üretim veya billing yapmak bu denetimin parçası olarak onaylanmadı.

### D. “Her log satırı JSON” şartının altyapı kapsamı eksik

12 Ekim VPS log denetimi:

| Servis | İncelenen satır | JSON + run_id/node_id/attempt |
|---|---:|---:|
| API | 25.727 | 25.727 |
| gate | 24.903 | 24.903 |
| proxy | 257 | 257 |
| dashboard/Nginx | 68 | 59 |
| PostgreSQL | 68 | 0 |

API/worker uygulama log sözleşmesi, provider ERROR kod/mesajları ve ortak null bağlam uygulanmış/test edilmiş. Nginx access logları da JSON. Nginx entrypoint/native error ve PostgreSQL motor logları aynı JSON sözleşmesinde değil; README bunu altyapı istisnası olarak açıklıyor. Case'in “her log satırı” ifadesi bütün container çıktısına uygulanırsa şart tamamen karşılanmıyor. Katı uyum isteniyorsa altyapı loglarının da normalize edilmesi gerekir; sadece Docker `json-file` log driver'ı kullanılması mesaj içeriğini JSON yapmaz.

### E. Teslim dokümanlarında eski ifadeler var

- [docs/ACCEPTANCE.md](ACCEPTANCE.md) son paragraf: “Bulut deploy isteğe bağlıdır ve yapılmadı.” Güncel cloud durumu ile çelişiyor.
- [deploy/README.md](../deploy/README.md) son paragraftaki “Son kod CI'sı” linki `3bf8667` sürümüne gidiyor. Güncel kod düzeltmesi `ef9bbdb`, güncel HEAD/CI `71208e6`.
- PLAN.md uygulama başlangıcının tarihsel planıdır; durum cümlesi güncel kabul/deploy özeti olarak okunmamalı.

README ana dosyası cloud worker/anahtar/history sınırını doğru açıklıyor. Eski kabul/deploy durum cümleleri teslimden önce güncellenmeli. Bu denetimde mevcut dokümanların ifadeleri düzeltilmedi; bulgular ayrı raporlandı.

### F. Gerçek fal ücreti ve birleşik gerçek toplam canlı doğrulanmış değil

Tahmini maliyet bonusu tam uygulanmış, exact decimal ile kaydediliyor. Kullanıcının ek istediği başlangıç tahmini snapshot'ı, adım başına gerçek maliyet ve gerçek toplam da kodda/mock testlerde mevcut.

OpenRouter gerçek tutarları canlı `usage.cost` üzerinden doğrulandı. fal exact-request billing denemesi mevcut anahtarın yetkisi nedeniyle 403 dönmüş; fal gerçek ücret ve birleşik actual total bilinmiyor. `null/pending/unavailable` kalmaları doğrudur. Yeni yetkili billing anahtarı ve ayrı çağrı onayı olmadan “gerçek toplam da canlı doğrulandı” denemez. Bu, case'teki **tahmini** maliyet bonusunun eksik olması anlamına gelmez.

## 3. Case maddeleriyle karşılaştırma

Durumlar: **Var/testli** = kod ve otomatik test; **Canlı kanıtlı** = önceki açık onaylı gerçek çağrı sonuçları; **Açık** = yukarıdaki bulgu veya deployment sınırı. Cloud'da gerçek run olmaması, geçmişteki yerel canlı kanıtı geçersiz kılmaz.

| Case gereksinimi | Sonuç | Kontrol yeri |
|---|---|---|
| Go backend ve ayrı asenkron worker | Var/testli; cloud worker kapalı | cmd/flowkit, internal/jobs, compose.yaml; bulgu C |
| PostgreSQL'de workflow/model verisi | Var/testli; cloud'da 3 model/2 workflow | db/migrations/00001_init.sql, internal/storage |
| Katalog kayıt/liste/devre dışı bırakma | Var/testli | POST/GET /models, PATCH /models/{alias}; CRUD integration testi |
| Workflow CRUD/validate | Var/testli; ID/findings istisnası açık | /workflows; bulgu B |
| Dashboard tüm işlemlerini HTTP API ile yapar | Var/testli | web/src/api.ts, main.tsx, docs/openapi.json |
| Yeni uyumlu model, kod/restart/deploy olmadan | Canlı kanıtlı ve mock integration testli | flowkit-approved.json runtime_before/runtime_after_all, TestHotAddModelAndWorkflowWithoutRestart |
| Katalog alias'ı; bilinmeyen modelin reddi | Var/testli | example modelRef, validate.go; unknown_model testi |
| JSON nodes/edges/kind/port sözleşmesi | Var/testli | schemas/workflow.schema.json; product/event JSON |
| Döngü/fan-out/eksik port/tip/model/template bulguları | Altı kural da var/testli | TestRejectGraphRules, TestIndependentFindings; bulgu B ayrı |
| Tek bağlı zincir/input/output; fan-in reddi | Var/testli | internal/workflow/validate.go |
| Girdi ve önceki node template referansları | Var/testli/canlı kanıtlı | product/event promptTemplate; Render testleri |
| Run start hemen run_id; arka plan sıralı yürütme | Var/testli/canlı kanıtlı | startRun + River InsertTx; async integration testi |
| fal queue submit/status/result | Var/testli/canlı kanıtlı | providers/client.go, direct.json/flowkit-approved.json |
| provider_request_id hemen commit | Var/testli/canlı kanıtlı | engine.go submit sonrası save; killed-worker integration testi |
| Tipli hatalar, geçici hatada en fazla 3 deneme | Var/testli | engine/client testleri; README retry politikası |
| Failed node/hata kalıcı; atlanan node izi | Var/testli | runs/node_runs/node_attempts/provider_requests; permanent failure integration testi |
| Testler gerçek sağlayıcı çağırmaz | Var/testli | httptest/mock implementasyonları; CI'da provider anahtarları yok |
| İlk doğrudan OpenRouter → fal denemesi | Canlı kanıtlı | docs/evidence/direct.json; README eşleştirme tablosu |
| İkinci farklı workflow, sadece API/veri | Canlı kanıtlı; temiz kurulumda iki seed | API mutation ve runtime kanıtı; examples/workflows |
| İki workflow farklı image modelleri kullanır | Var/testli/canlı kanıtlı | ürün Schnell, etkinlik Dev; cloud JSON ve geçmiş snapshot'lar |
| Her çıktı yanında gerçek üretici model | Var/canlı dashboard kanıtı | RunDetail image.actual_model_id; docs/dashboard-*-live.jpg |
| Node model/attempt/süre/input/output/error izi | Var/testli/canlı kanıtlı; cloud geçmiş boş | GET /runs/{id}, web RunDetail; bulgu C |
| JSON log ve provider ERROR log | Uygulama kapsamı testli; tüm altyapı satırlarında açık | internal/telemetry, engine.event; bulgu D |
| En az üç invalid kural ve temel mock başarı/retry/fail/CRUD | Var/testli; asgari sınırın üstünde | workflow/engine/providers/integration testleri ve CI |
| Workflow listesi/JSON doğrula-kaydet/run formu/geçmiş/detay | Var; farklı input field form hatası açık | web/src/main.tsx; bulgu A |
| docker compose up: bütün temel servisler | Standart Compose CI'da testli | compose.yaml, tools/check-compose.mjs |
| .env.example, README endpoint/cURL/OpenAPI | Var; eski teslim durum ifadeleri açık | .env.example, README, docs/openapi.json; bulgu E |
| GitHub public veya yetkili private | Public, sağlanmış | GitHub repository |
| Opsiyonel deployment parola korumalı | HTTPS/auth sağlanmış; gerçek üretim aktif değil | deploy/README, deployment.json; bulgu C |
| Bonus: worker restart aynı fal ID | Var, gerçek process kill + mock/PG testli | TestKilledWorkerResumesDurableFalTicket |
| Bonus: idempotency | Var, eşzamanlı 10 istek/tek run-job testli | TestConcurrentIdempotencyAndSnapshot |
| Bonus: node ve total tahmini maliyet | Var/testli/canlı kanıtlı | Quote/RunCosts, request ledger, decimal; cost tests |
| Ek istek: ayrı başlangıç/adım/gerçek/gerçek toplam | Var/mock testli; fal live actual yetki nedeniyle açık | billing tests, RunDetail; bulgu F |
| AI talimat dosyası ve hataların şeffaf anlatımı | Var | AGENTS.md, docs/AI_LOG.md, README AI bölümü |
| Model seçimi/bütçe gerekçesi | Var; fal toplamı tahmin olarak ayrılmış | README model/bütçe, evidence final_budget_summary |
| Bir hafta daha olsaydı paragrafı | Var | README son paragraf |
| Kapsam dışı canvas/paralel node/subworkflow/auth/payment/promotion | Eklenmemiş; deployment gate dışında auth yok; farklı run'lar eşzamanlı olabilir | PLAN.md kapsam/profil sınırları |

## 4. Kendin kontrol etmek için sıra

### 1 — GitHub ve test sonuçları

[Repository](https://github.com/CanhK-2103/trypix-flowkitCase) ve [CI](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38170369875) aç. Backend/dashboard/compose üçü de yeşil olmalı. Backend logunda integration testler skip olmamalı: CI ayrı `flowkit_test` PostgreSQL kullanır. Yeşil CI, bulgu A/B'nin yok olduğu anlamına gelmez; bu iki regression senaryosu mevcut testlerde yok.

### 2 — Public site / koruma

Gizli pencerede https://flowkit.canhurkurtoglu.com aç: tek şifre alanı gelmeli; şifresiz `/models`, `/workflows`, `/runs` JSON endpoint'leri 401 dönmeli. Giriş sonrası dashboard/JSON açılmalı, çıkış sonrası tekrar şifre istenmeli. Site şifresi yerel ignored `.env` içindeki `VPS_DASHBOARD_PASSWORD`; server'da yalnız bcrypt hash var. SSH şifresini değerlendiriciyle paylaşma.

### 3 — Sunucu checkout ve servisler

SSH ile aşağıdaki **salt okunur** komutları kullan:

```sh
cd /opt/flowkit
git rev-parse HEAD
git status --short
docker compose -p flowkit -f compose.yaml -f compose.production.yaml ps -a
```

Denetlenen sürüm `71208e6`. Init/bootstrap exited(0), API/db/gate healthy, dashboard/proxy running beklenir. Worker şu anda listede running değildir. Bu bilinçli deployment sınırıdır; `up --profile generation` veya anahtarlı normal worker başlatmak onaysız background çağrılara yol açabilir.

### 4 — Katalog

Giriş yaptıktan sonra https://flowkit.canhurkurtoglu.com/models aç. `prompt-writer → openai/gpt-6-luna`, `flux-schnell → fal-ai/flux/schnell`, `flux-dev → fal-ai/flux/dev` ve enabled:true kontrol et. Katalog ekranı case'te zorunlu değil. POST/PATCH örnekleri README/OpenAPI'de; CRUD mock testi models kaydet/liste/patch-disable işlemlerini doğrular.

### 5 — İki hazır workflow

Dashboard sol listede ürün ve etkinlik workflow'unu ayrı ayrı aç; “JSON tanımı”na geç. İkisinde input/llm/image/output ve 3 edge bulunmalı. Ürün image modelRef `flux-schnell`, etkinlik `flux-dev` olmalı. https://flowkit.canhurkurtoglu.com/workflows endpoint'inde de aynı veri görünür. Startup bu kayıtları API ile yükler; yeni model/ikinci workflow'un restart olmadan eklenmesinin ayrı gerçek demosu madde 13'tedir.

### 6 — Şema / port / template

[schemas/workflow.schema.json](../schemas/workflow.schema.json), [product.json](../examples/workflows/product.json), [event.json](../examples/workflows/event.json) karşılaştır. input.description, llm promptTemplate, image prompt bağlantısı ve event'teki `{{nodes.art_director.prompt}}` önceki çıktı referansını incele. Yeni düz scalar input alanları backend'de desteklenir; dashboard için bulgu A henüz düzeltilmedi.

### 7 — Bulgular listesi / hatalı grafın kaydedilmemesi

Editor'da sadece taslak kopya üzerinde modelRef'i katalogda olmayan alias yapıp bir zorunlu edge'i kaldır; “Doğrula” birden fazla code/path/node_id bulgusu göstermeli. Kopyayı kaydetme; sayfayı yenileyerek taslağı geri al. Altı zorunlu kuralın tek tek kanıtı [validate_test.go](../internal/workflow/validate_test.go) `TestRejectGraphRules` alt testleridir. ID/meta için bulgu B istisnasını ayrıca oku.

### 8 — API yüzeyi

[README API tablosu](../README.md#api) ile [docs/openapi.json](openapi.json) incele. Katalog, workflow CRUD/validate, run start/status/detail/history, estimate ve artifact endpoint'leri olmalı. Dashboard browser Network panelinde bu HTTP endpoint'lerini kullanır; ayrı gizli dashboard iş mantığı yok.

### 9 — Asenkron worker ve sıralı motor

[server.go](../internal/api/server.go) `startRun`, [jobs.go](../internal/jobs/jobs.go) ve [engine.go](../internal/engine/engine.go) oku. Run + snapshot + River insert aynı transaction'da, HTTP 202 hemen dönüyor. Node yürütme ayrı worker'da. Asenkron çalışmayı ücretsiz kanıtlayan `TestAsyncWorkerSuccessRetryAndPermanentFailure` ve CI Compose `input → output` run'ını incele; cloud'da şu anda gerçek üretim başlatılamaz.

### 10 — fal queue ve hemen kalıcılaşan ID

`engine.go` Submit sonrası `ProviderRequestID/StatusURL/ResponseURL` ataması ve `e.save` çağrısını kontrol et; pricing/status/result/media işlemleri sonraki fazlarda olmalı. `flowkit-approved.json` event run DB kanıtında submit'ten 764 ms sonra ID commit edilmiş. `TestKilledWorkerResumesDurableFalTicket` doğrudan bu commit sonrası process'i öldürüyor.

### 11 — Retry ve kalıcı hatalar

`TestTransientRetryThenSuccess`, `TestRetryCap`, `TestPermanentFailureCorrectNode`, integration success/retry/permanent subtest'lerini incele. Beklenen: transient sonra başarı; toplam en fazla 3 domain attempt; permanent hata render node'u ve INVALID_INPUT/provider bad_request ile kalıcı failure; sonraki node skipped. Request kimliği bilinen poll hatası yeni submit/ücret oluşturmaz.

### 12 — Run trace / producer / görseller

Önceden başarılı gerçek kayıtlar yerel kabul ortamındaki http://localhost:8081/runs endpoint'inde; Docker kapalıysa şu anda erişilmez. Repository'de [JSON run kanıtı](evidence/flowkit-approved.json), [ürün ekranı](dashboard-schnell-live.jpg), [etkinlik ekranı](dashboard-event-live.jpg) her zaman incelenebilir.

| Run | Model | Kalıcı artifact |
|---|---|---|
| a733fb1e-686e-4d78-8386-09ce5953a36a | fal-ai/flux/schnell | 70266b82-d304-4ef6-80c1-9696e044f5d1 |
| 217fb931-3af8-4313-89b2-436985798449 | fal-ai/flux/dev | f0f242b1-2b6d-4d1a-8925-6e136605d037 |
| 614b472e-e6c8-4e17-b07d-995f4307b321 | fal-ai/flux/dev | 53c51066-5d9f-4c39-b758-15a831a44ac7 |

Her kayıtta `workflow_snapshot`, `model_snapshot`, nodes, attempts, provider_requests, duration_ms, input/output/error ve image.actual_model_id kontrol et. Dashboard'da node satırını açınca aynı iz var. Görseller provider URL'sinden bağımsız volume'a indirilmiş ve `/artifacts/{id}` üzerinden HTTP 200 ile okunmuş; cloud'da bu artifact'ler yok.

### 13 — Kod/restart/deploy olmadan model/workflow ekleme

`flowkit-approved.json` içinde API mutation request/response gövdeleri, `runtime_before`, `runtime_after_all` ve `runtime_model_workflow_acceptance_complete:true` karşılaştır. Git SHA ve dört container ID/StartedAt/image kimlikleri aynı. Akış: ürün/Schnell → API'den Dev modeli → ürün modelRef Dev + başarılı run → ürünü Schnell'e geri alma → API'den etkinlik/Dev ekleme + başarılı run. Önceki run model/revision snapshot'ları değişmiyor. Hazır cloud seed'lerinde Dev zaten kayıtlı olduğundan aynı POST'u yeni model ekleme demosu olarak yeniden kullanma.

### 14 — İlk doğrudan deneme / eşleştirme

[direct.json](evidence/direct.json) ve [README eşleştirme tablosu](../README.md#ilk-doğrudan-akış--flowkit) aç. Gerçek OpenRouter modeli/prompt/parametre/usage; fal endpoint/input/request/status/result bulunmalı. Tablo aynı sözleşmenin input/llm/image/output node'larına nasıl taşındığını gösteriyor. `cmd/direct` veya live-demo komutlarını incelemek için çalıştırmak gerekmez; anahtarla çalıştırmak ayrı canlı kullanım onayı ister.

### 15 — Bonus restart

[integration_test.go](../internal/integration/integration_test.go) `TestKilledWorkerResumesDurableFalTicket`: first worker Process.Kill, ikinci worker recovery; final succeeded, submitCount==1, LLM completeCount==1, tek image attempt/request ve tek cost kaydı kontrol edilir. Bu mock+gerçek PostgreSQL+gerçek worker process testidir; ücretli fal isteği sırasında canlı sunucu öldürülmedi. Belirsiz sending fazında ID yoksa kör yeniden gönderim yerine PROVIDER_SUBMISSION_UNKNOWN oluşur.

### 16 — Bonus idempotency

Aynı dosyada `TestConcurrentIdempotencyAndSnapshot`: 10 eşzamanlı aynı key/body için tek run/tek River job; farklı body için 409. API 202 ilk istekte, replay 200 aynı run_id. Geçmiş workflow/model snapshot'ları katalog veya workflow değişiminden etkilenmez.

### 17 — Bonus tahmini ve ek gerçek maliyet

Dashboard üretim formunda **başlangıç tahmini toplam**, run detayında başlangıç/çalıştırma/gerçek node sütunları ve **gerçek toplam** kontrol edilir. Quote katalog verisidir; provider çağrısı yapmaz. Run initial_estimate sonradan değişmez; gerçek total tüm request tutarları bilinmeden dolmaz. `TestQuoteAndDelayedBillingPersistedWithAPIScheduledRefresh` mock accounting geç gelince total'i 0.002720000000 yapar ve restart/reconcile tekrarında ikinci ücret üretmez. Mevcut gerçek run'larda fal actual null olduğu için gerçek toplamın null olması beklenir. “Maliyet kaydını yenile” provider billing çağrısı planlar; izinsiz denemeyin.

### 18 — Loglar

SSH üzerinde salt okunur:

```sh
cd /opt/flowkit
docker compose -p flowkit -f compose.yaml -f compose.production.yaml logs --no-color --no-log-prefix --tail=20 api gate proxy
```

Satırlar JSON; run_id/node_id/attempt bulunur, run bağlamı yoksa null. Provider hata alanlarını engine.event ve telemetry testinde incele. Worker cloud'da kapalı olduğu için cloud'da gerçek provider error trace bekleme. Bütün altyapı satırları için bulgu D açık.

### 19 — Mock testleri kendin çalıştırma

Go/Node yerel araçlarıyla provider endpoint/key kullanmadan:

```powershell
go test ./internal/workflow ./internal/engine ./internal/providers ./internal/telemetry -count=1 -v
cd web
npm test
npm run build
```

Tam integration suite ayrıca yalnız `flowkit_test` DB ister (testler bu test DB'sini temizler; normal/local/cloud DB URL'sini kullanma). `compose.test.yaml` sadece izole test PostgreSQL içindir:

```powershell
docker compose -p flowkit-tests -f compose.test.yaml up -d --wait
$env:TEST_DATABASE_URL='postgres://flowkit:flowkit@localhost:55432/flowkit_test?sslmode=disable'
go test ./... -count=1 -timeout=6m
go vet ./...
sqlc generate
git diff --exit-code -- internal/storage/dbgen
```

Bu komutlar gerçek provider yerine httptest/mock kullanır. Docker Desktop'ın mevcut provider anahtarlı servisleri otomatik başlatabileceğini dikkate al; bu oturumda denetçi Docker'ı yeniden başlatmadı. CI bütün suite için hazır, maliyetsiz inceleme yoludur.

### 20 — Tek komut kurulum ve temiz seed davranışı

`compose.yaml`: DB → init migration → API readiness → API üzerinden bootstrap → dashboard; ayrı worker ve kalıcı DB/artifact volume'ları. CI Compose job'ı anahtarlar boşken tam `docker compose up` karşılığını ve sağlayıcısız async run'ı test eder. `.env.example` provider/DB/deploy ayarlarını içerir; gerçek `.env` Git'te yok. Kendi anahtarlarınla normal `docker compose up` worker'ı açabilir; mevcut oturumun tek-çağrı onay politikasını atlayarak deneme yapma.

### 21 — Bütçe

Evidence `final_budget_summary` ve README bütçe paragrafı: 4 gerçek zincir, OpenRouter bilinen actual toplam **0.0001962 USD**, fal tahmini toplam **0.106 USD**, fal actual/birleşik actual total bilinmiyor. Case limitleri fal 5 / OpenRouter 2 USD; ayrı review eşikleri 0.50 / 0.05. Test/startup ek üretim yapmaz. Bu tutarlar yalnız kaydedilmiş deneylerin toplamıdır; sağlayıcı hesabındaki kalan bakiye billing yetkisi olmadan doğrulanmış sayılmaz.

### 22 — AI şeffaflığı, kararlar ve teslim dosyaları

[AGENTS.md](../AGENTS.md), [AI_LOG.md](AI_LOG.md), README mimari/graf/endpoint/retry/model/AI bölümleri ve son “bir hafta” paragrafını incele. Checkpoint, exact decimal, uncertain submission, idempotency transaction ve profile sınırları açıklanmış. Bu rapordaki açık bulgular çözülmeden eski “tamamlandı” ifadelerini bütün case için hatasızlık garantisi olarak okuma.

## 5. Teslim öncesi önerilen sıra

1. Dashboard input state ve ID/findings davranışlarını düzelt, iki regression senaryosunu testlere ekle.
2. Log şartının bütün container'ları kapsayacak katı uygulamasını tamamla veya değerlendiren tarafla kapsamını açıkça netleştir.
3. Eski acceptance/deployment durum ifadelerini güncelle.
4. Canlı URL'nin üretim ve geçmiş/örnek sonuçlarını nasıl göstereceğini tamamla. Yeni provider kullanımını, polling/retry/pricing/billing dahil, ayrı açık onaylarla sınırla.
5. Yetkili fal accounting erişimi sağlanırsa kullanıcının ek istediği gerçek toplamı canlı doğrula; erişim yoksa null/unavailable sınırını teslimde koru ve açıkla.

Bu adımlar uygulanmadan “hiçbir eksik/hata yok, cloud'da her şey çalışıyor” sonucu verilmez.
