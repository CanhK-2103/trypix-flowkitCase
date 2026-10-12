# FlowKit — tamamlanan case kontrolleri, 12 Ekim 2026

Referans: kullanıcının `C:\Users\PC\Downloads\case.txt` dosyası. İlk denetimdeki beş uygulama/deployment/dokümantasyon bulgusu tamamlandı. Kullanıcının kapsam dışında bıraktığı **fal billing yetkisi ve gerçek toplamı canlı doğrulama** için ek sorgu yapılmadı.

Uygulama kodu `14cf20a6bd53eb6904736938a6114002c5de5830` sürümünde GitHub'a gönderildi ve VPS'e build/recreate edildi. [Bu kod sürümünün CI'sı](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38241210130) backend/dashboard/compose job'larında başarılıdır. Sonraki dokümantasyon commit'leri uygulama image'larını değiştirmez.

## Güncel ve doğrulanmış durum

- Public repo: https://github.com/CanhK-2103/trypix-flowkitCase.
- Public uygulama: https://flowkit.canhurkurtoglu.com; tek şifreli giriş, HTTPS ve oturumsuz API/asset koruması çalışır.
- Server proje dizini `/opt/flowkit`. API/DB/gate healthy; dashboard/proxy/**worker running**; init/bootstrap exited(0).
- Provider anahtarları yalnız ignored, mode 600 server `.env` içindedir. **BILLING_DISABLED=true**: otomatik accounting ve refresh planlama kapalı. Key değerleri frontend/Git/loglara verilmedi.
- Cloud'da **3 aktif model, 2 aktif workflow, 3 gerçek başarılı geçmiş run ve 3 görsel** bulunur. Bunlar önceki onaylı yerel kabul DB'sinden snapshot/tarih/ücret bilgileri korunarak taşındı; yeni ücretli üretim yapılmadı. River işleri taşınmadı.
- Ek olarak ücretsiz, sağlayıcısız bir cloud worker kontrol run'ı var: `edbca753-ebe1-4e96-aef3-90412a1e5f79`. Kontrol workflow'u soft-delete edildi. Global `/runs` toplamı 4; provider request toplamı hâlâ eski 6, tamamlanmış River job sayısı 1.
- Üç görsel authenticated public API'den HTTP 200 döndü; SHA-256 değerleri orijinallerle aynı. [Tamamlama kanıtı](evidence/deployment-fixes.json).
- Metadata/UUID ve graf bulguları public validate/create/update üzerinde birlikte doğrulandı. Farklı `message` alanı ve hiç dokunulmamış `approved:false` değeri gerçek browser/form ve cloud worker üzerinden başarılı.
- Bütün güncel container logları JSON ve run_id/node_id/attempt alanlarını içeriyor. PostgreSQL/Nginx native logları da normalize edildi.
- Son tam Go/PostgreSQL suite'i 39.866 saniyede başarılı; gofmt, vet, sqlc ve OpenAPI regeneration kontrolleri geçti. Frontend **13 test**, typecheck/build başarılı. Standart ve production Compose smoke başarılı. Production smoke DB/Nginx/gate/proxy restart, veri/oturum kalıcılığı ve native PG ERROR logunu kapsıyor.
- Tamamlama turunda yeni OpenRouter/fal üretim, pricing, polling veya billing çağrısı: **0**. Cloud'un gerçek sağlayıcılarla yeni üretimi bu turda tekrar denenmedi; mevcut adapter'ların önceki gerçek kabul kanıtı korunur.

## İlk denetimdeki bulguların sonucu

| Bulgu | Yapılan düzeltme | Kanıt |
|---|---|---|
| Dashboard gizli description gönderiyordu | Form/payload workflow şemasından oluşturulur; workflow/revision/schema değişiminde reset; boolean false, numeric ve enum türleri korunur | WorkflowInput.test.tsx, browser [başarı ekranı](input-regression-fixed.jpg), cloud providerless run |
| UUID validate/create/update bulguları tutarsızdı | Ortak metadata UUID finding; body/path mismatch diğer graf bulgularıyla toplanır; geçersiz write kalıcı kaydı değiştirmez | API unit + PostgreSQL validation_test.go; public üç endpoint kanıtı |
| Cloud worker/keys/geçmiş yoktu | Worker etkin; keys private env'de; üç başarılı tarihsel run ve mevcut media taşındı; accounting kapalı; worker gerçek sağlayıcı çağrısı olmadan kontrol edildi | deployment-fixes.json; SHA-256/artifact ve providerless idempotency |
| DB/Nginx native logları JSON değildi | cmd/logwrap resmî entrypoint'leri ve stop sinyallerini korur; native/access/startup/error satırları JSON/context; secret redaction | Unit tests; production smoke ve cloud tüm-service log audit'i |
| Acceptance/deploy metinleri eskimişti | README, ACCEPTANCE ve deploy README güncel durumu/CI/kanıtları gösterir; eski kurulum kayıtları tarihsel diye işaretlendi | İlgili dokümanlar |
| Fal billing yetkisi yok | **Kullanıcı talebiyle kapsam dışı.** Yeni accounting sorgusu yapılmadı; bilinmeyen actual/total null olarak korunur | BILLING_DISABLED, mevcut 403 kaydı, mock accounting testleri |

Testlerin yakaladığı ek iki nokta da düzeltildi: yeni `/capabilities` endpoint'i Nginx API route listesine eklendi; kontrollü worker testinin Windows'ta tekrar edebilen saat damgası kimlikleri benzersiz fixture kimliklerine çevrildi (30 tekrar başarılı). Production onay mekanizması gevşetilmedi. Input node raw girdi izi de kalıcılaştırıldı; eski immutable run'ların dashboard izi kendi run.input kaydını gösterir.

## Case gereksinimleriyle karşılaştırma

“Testli” gerçek provider çağrısı anlamına gelmez. “Gerçek kanıt” önceki açık onaylı provider sonuçlarını; “cloud” mevcut hosting kontrolünü ifade eder.

| Gereksinim | Sonuç / kontrol |
|---|---|
| Go backend, ayrı worker, PostgreSQL | Var; üçü de cloud'da aktif. cmd/flowkit, internal/jobs, compose.yaml |
| Workflow/model tanımları veri olarak DB'de | Var; JSONB/catalog, model kimlikleri engine'de hardcode edilmez |
| Katalog API kayıt/liste/disable | Var/testli: POST/GET /models, PATCH /models/{alias} |
| Workflow CRUD ve bulgu listesi | Var/testli/public doğrulanmış: create/list/get/update/delete/validate; UUID dahil ortak bulgular |
| Dashboard yalnız HTTP API kullanır | Var: api.ts, TanStack Query, OpenAPI; kapsanmayan gizli dashboard operasyonu yok |
| Node/edge/kind/port JSON sözleşmesi | Var/testli: input/llm/image/output; text/prompt/image; schemaVersion:1 |
| Tek input/output, doğrusal ve bağlı graf | Var/testli; cycle/fan-out/fan-in/disconnected reddi |
| Altı zorunlu validation kuralı | Hepsi testli: cycle, fan-out, eksik port, tip, kayıtlı olmayan model, çözülemeyen template |
| Girdi ve önceki çıktı template'leri | Var/testli/gerçek kanıtlı; input.description ve nodes.art_director.prompt |
| Alias referansı; model tahmini/fallback yok | Var/testli; unknown/disabled alias save/run'da reddedilir |
| Uyumlu yeni model kod/restart/deploy olmadan | Gerçek kanıtlı + mock integration: runtime_before/runtime_after_all aynı Git/container kimlikleri |
| Asenkron run_id ve sıralı yürütme | Var/testli/gerçek kanıtlı/cloud providerless doğrulanmış; run+River enqueue tek transaction |
| Fal queue submit/status/result | Var/testli/önceki gerçek kanıtlı |
| Provider ID hemen kalıcı kayıt | Var; pricing/poll/media öncesi commit; gerçek kayıt ve process kill testi |
| Tipli hata ve max 3 geçici domain attempt | Var/testli; 2/4 saniye+jitter, Retry-After; queue retry ayrı |
| Failed node/kod/neden kalıcı | Var/testli; kalıcı provider hatası doğru node'u fail eder, sonrakiler skipped |
| Mock testler provider/pricing/media gerçek çağırmaz | Var/testli; httptest/local mocks ve isolated PostgreSQL |
| İlk doğrudan OpenRouter→fal denemesi + README eşleştirme | Gerçek kanıtlı: direct.json ve README tablo |
| İkinci farklı workflow sadece veriyle | Gerçek kanıtlı; ürün/Schnell ve etkinlik/Dev cloud'da mevcut |
| Her görsel yanında gerçek üretici model | Var; run image.actual_model_id/provenance; önceki gerçek dashboard ve current artifact kanıtı |
| Run input/output/model/attempt/süre/hata izi | Var/testli; immutable snapshot + normalized node/attempt/request tabloları; cloud geçmiş mevcut |
| Her container logu JSON/context; provider ERROR | Var/testli/cloud doğrulanmış; native DB/Nginx/Caddy dahil |
| Dashboard list/editor/validate/save/input/history/detail | Var; farklı scalar alanlar/boolean/enum regression testleri ve browser doğrulaması |
| docker compose up: tam sistem | Var; standart Compose smoke ve CI temiz kurulumu; production worker başlangıç profili ayrı |
| .env.example, README endpoint/cURL/OpenAPI | Var; gerçek env Git'te yok |
| GitHub public veya yetkili private | Public; ek erişim daveti gerekmiyor |
| Opsiyonel deployment parola korumalı | Yapıldı; HTTPS/password-only/all-route gate ve private backend portları |
| Bonus: restart aynı fal ID, tekrar submit yok | Var; gerçek process kill + mock/PG integration testli |
| Bonus: idempotency | Var; eşzamanlı 10 istek → tek run/job; conflict ve snapshot testleri; cloud replay |
| Bonus: node + total tahmini maliyet | Var/testli/gerçek kanıtlı; exact decimal, immutable before-run estimate |
| Ek istek: ayrı gerçek node/total | Kod/mock testleri var; OpenRouter usage actual geçmişte doğrulandı; fal live accounting kapsam dışı/null |
| Bütçe yönetimi/model gerekçesi | README ve evidence final_budget_summary; minimal onaylı çağrılar, test harcaması 0 |
| AI talimatı/kullanımı/hataları/kararları | AGENTS.md, AI_LOG.md, README; gerçek başarısız test ve düzeltmeler açık |
| Bir hafta daha olsaydı paragrafı | README son paragrafı |

Canvas, paralel node, subworkflow, ödeme/promotion ve uygulama içi hesap/yetki sistemi kapsam dışı kalır. Farklı run'lar eşzamanlı olabilir; deployment gate case'in opsiyonel hosting şifre korumasıdır.

## Kendin inceleme sırası — 22 kontrol

### 1. Repository ve CI

[Repository](https://github.com/CanhK-2103/trypix-flowkitCase) ve [uygulama kodu CI](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38241210130) aç. Backend/dashboard/compose yeşil. Backend'de PostgreSQL integration testleri skip olmamalı; CI ayrı flowkit_test DB kullanır. Sonraki README/kanıt commit'lerini aynı repo history'den görebilirsin.

### 2. HTTPS ve site şifresi

Gizli pencerede https://flowkit.canhurkurtoglu.com aç: tek şifre alanı; şifresiz `/models`, `/workflows`, `/runs`, `/artifacts/...` 401. Giriş sonrası dashboard/JSON; logout sonrası yeniden login. Şifre yerel ignored `.env` içindeki VPS_DASHBOARD_PASSWORD; server'da bcrypt hash var. Değerlendiriciye yalnız site şifresi verilir.

### 3. Server checkout ve bütün servisler

```sh
cd /opt/flowkit
git rev-parse HEAD
git status --short
docker compose -p flowkit -f compose.yaml -f compose.production.yaml ps -a
```

API/DB/gate healthy, dashboard/proxy/**worker running**, init/bootstrap exited(0). Backend host portları private; dışarıda yalnız proxy HTTP/HTTPS. Sonraki docs-only Git SHA farklı olsa da deployed application code revision 14cf20a'dır.

### 4. Model kataloğu

Girişten sonra https://flowkit.canhurkurtoglu.com/models: `prompt-writer → openai/gpt-6-luna`, `flux-schnell → fal-ai/flux/schnell`, `flux-dev → fal-ai/flux/dev`; enabled:true. Katalog ekranı zorunlu değildir. POST/PATCH örneklerini OpenAPI'de, kayıt/liste/disable testini integration_test.go'da incele.

### 5. İki workflow

Sol listede ürün ve etkinlik; “JSON tanımı”nı ayrı aç. Ürün revision 3/image modelRef flux-schnell, etkinlik revision 1/flux-dev. input/llm/image/output ve üç edge. Aynı veri `/workflows` JSON endpoint'inde vardır.

### 6. Graf, port ve template

[Schema](../schemas/workflow.schema.json), [product.json](../examples/workflows/product.json), [event.json](../examples/workflows/event.json): id/kind/modelRef/params/promptTemplate; from/fromPort/to/toPort; input.description ve önceki node prompt referansı. Input schema düz string/number/integer/boolean ve enum alanlarını destekler.

### 7. Bulgular ve hatalı kaydın reddi

Editor'da sadece taslakta bilinmeyen modelRef ve eksik edge oluşturup “Doğrula” kullan: birden çok code/path/node_id bulgusu. Taslağı kaydetmeden sayfayı yenileyebilirsin. UUID hatası da valid:false/findings döndürür; update body/path uyuşmazlığı graf bulgularını gizlemez. Public test sonucu deployment-fixes.json metadata_validation alanında.

### 8. HTTP API kapsamı

[README API tablosu](../README.md#api), [OpenAPI](openapi.json), web/src/api.ts. Katalog, workflow CRUD/validate, runs/status/detail/history, estimate, artifacts, health, capabilities. Dashboard browser Network panelinde bunları kullanır. Geçersiz CRUD testleri kalıcı kaydı değiştirmediğini de doğrular.

### 9. Asenkron çalışma

server.go startRun: run/snapshot/River InsertTx tek transaction; HTTP 202 hemen. jobs.go ayrı worker. Cloud ücretsiz kontrol edbca753 succeeded; `POST` replay 200 aynı ID. Gerçek AI üretimini yeniden kontrol etmek ayrı sağlayıcı kullanım onayı ister.

### 10. Fal queue ve hemen ID commit

engine.go Submit sonrası ProviderRequestID/URL ataması + save; sonra pricing/status/result/media fazları. Önceki canlı event run'ında ID 764 ms içinde commit edilmiş. Killed-worker testi bu commit ile River Work dönüşü arasındaki aralığı test eder.

### 11. Retry ve failure yolları

engine_test.go TestTransientRetryThenSuccess/TestRetryCap/TestPermanentFailureCorrectNode; integration success/retry/permanent subtest'leri. Geçici hata retry→başarı; max 3 domain attempt; permanent hata render/INVALID_INPUT/provider bad_request; sonrakiler skipped. Poll hatası aynı ticket'ı kullanır.

### 12. Cloud run geçmişi ve adım izi

Ürün/etkinlik seç → “Son çalıştırmalar” → başarılı run → node satırlarını aç. Girdi/çıktı, rendered prompt, producer model, request ID, attempt, süre ve maliyet ayrı. API'den doğrudan JSON:

| Run | Model | Authenticated API |
|---|---|---|
| a733fb1e-686e-4d78-8386-09ce5953a36a | fal-ai/flux/schnell | https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a |
| 217fb931-3af8-4313-89b2-436985798449 | fal-ai/flux/dev | https://flowkit.canhurkurtoglu.com/runs/217fb931-3af8-4313-89b2-436985798449 |
| 614b472e-e6c8-4e17-b07d-995f4307b321 | fal-ai/flux/dev | https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321 |

### 13. Yeni model/workflow için restart gerekmemesi

[flowkit-approved.json](evidence/flowkit-approved.json) runtime_before/runtime_after_all ve mutation request/response gövdeleri: aynı Git SHA ve dört container ID/StartedAt/image. Ürün/Schnell → API Dev kaydı + modelRef değişimi → başarılı Dev run → ürün Schnell'e dönüş → ikinci etkinlik/Dev workflow ve run. Geçmiş model/revision snapshot'ları korunur. Cloud'da Dev zaten kayıtlı; aynı seed POST'u yeni-model demosu sayılmaz.

### 14. İlk doğrudan deneme ve eşleştirme

[direct.json](evidence/direct.json) ve README “İlk doğrudan akış → FlowKit” tablosu: gerçek model/prompt/parametre/usage, fal ticket/status/result. Komutları çalıştırmadan inceleyebilirsin; `cmd/direct`/live-demo çalıştırmak bu turda yapılmamış yeni gerçek çağrılar başlatabilir.

### 15. Restart bonusu

integration_test.go TestKilledWorkerResumesDurableFalTicket: gerçek process kill, ikinci worker, succeeded, tek submit/LLM completion/image attempt/request/cost. Bu mock+gerçek PostgreSQL testidir; yeni ücretli fal isteği sırasında VPS öldürülmedi. ID'siz uncertain sending, kör resubmit yerine PROVIDER_SUBMISSION_UNKNOWN verir.

### 16. Idempotency bonusu

TestConcurrentIdempotencyAndSnapshot: 10 eşzamanlı aynı key/body tek run/job; farklı body 409. Replay aynı run_id, workflow sonradan değişse/silinse eski snapshot korunur. Cloud providerless replay de 200/same ID olarak doğrulandı.

### 17. Başlangıç/node/toplam maliyet

Formdaki başlangıç tahmini provider çağırmaz. Run initial_estimate değişmez; çalıştırma tahmini ve OpenRouter usage gerçek ücreti ayrı. Gerçek fal/total null kalır; dashboard billing'in kapalı olduğunu açıklar ve refresh butonu yoktur. Mock TestQuoteAndDelayedBillingPersistedWithAPIScheduledRefresh ücret geç gelince doğru total/no-double-charge davranışını gösterir. Canlı billing yetkisi kullanıcı talebiyle kapsam dışı.

### 18. Her servis logu

```sh
cd /opt/flowkit
docker compose -p flowkit -f compose.yaml -f compose.production.yaml logs --no-color --no-log-prefix --tail=20 api worker db dashboard gate proxy init bootstrap
```

Her satır JSON ve run_id/node_id/attempt; bağlam yoksa null. Native PostgreSQL/Nginx satırları da JSON. Provider ERROR kod/mesajları engine.event/telemetry testinde; credential değerleri loglanmaz. Input/prompt/output ayrıntıları DB izindedir.

### 19. Mock testleri kendin çalıştırma

```powershell
go test ./internal/workflow ./internal/engine ./internal/providers ./internal/telemetry -count=1 -v
cd web
npm test
npm run build
```

Tam suite yalnız ayrılmış flowkit_test DB ister; bu DB testlerde temizlenir, cloud/normal DB URL'sini kullanma:

```powershell
docker compose -p flowkit-tests -f compose.test.yaml up -d --wait
$env:TEST_DATABASE_URL='postgres://flowkit:flowkit@localhost:55432/flowkit_test?sslmode=disable'
go test ./... -count=1 -timeout=6m
go vet ./...
sqlc generate
git diff --exit-code -- internal/storage/dbgen
```

Testler local mock provider/pricing/media kullanır. CI bütün suite için hazır maliyetsiz inceleme yoludur.

### 20. Tek komut kurulum ve saklama

compose.yaml: DB→init migrations→API→API üzerinden bootstrap→dashboard, ayrı worker. DB/artifact volume'ları kalıcı. .env.example provider/deploy ayarlarını içerir; normal `docker compose up` tam stack'i açar. Provider anahtarları olmadan AI run kabul edilmez; standart smoke providerless input→output ile gerçek worker'ı doğrular. Mevcut anahtarlarla normal worker açmak kuyruktaki ücretli işleri çalıştırabilir; agent ayrıca onay almadan böyle iş oluşturmaz.

### 21. Bütçe

README model seçimi/bütçe ve evidence final_budget_summary: dört önceki gerçek zincir; OpenRouter bilinen actual toplam 0.0001962 USD; fal tahmini 0.106 USD; fal actual/combined actual bilinmiyor. Case limitleri 5/2 USD; review eşikleri 0.50/0.05. Bu tamamlamanın yeni provider harcaması 0; bakiye yetkisizken doğrulanmış sayılmaz.

### 22. Görseller, AI şeffaflığı ve teslim dosyaları

Cloud sonuç görselinin altında gerçek model görünür, “Görseli aç” kalıcı `/artifacts/{id}` üzerinden açar. [Deployment kanıtı](evidence/deployment-fixes.json) checksum eşitliği; [önceki ürün](dashboard-schnell-live.jpg)/[etkinlik](dashboard-event-live.jpg) ve [form düzeltmesi](input-regression-fixed.jpg) ekranları. [AGENTS.md](../AGENTS.md), [AI_LOG.md](AI_LOG.md), README mimari/graf/endpoint/retry/bütçe/AI/bir hafta bölümleri kararları ve gerçek hataları açıklar.

Fal accounting yetkisi dışındaki ilk beş bulgu için gerekli kod, test, dokümantasyon ve server tamamlama işlemleri yapılmış ve yukarıdaki kanıtlarla doğrulanmıştır. Yeni ücretli cloud üretimiyle tekrar doğrulama yapılmış gibi sunulmaz.
