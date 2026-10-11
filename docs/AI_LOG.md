# AI çalışma kaydı

## 2026-10-10 — uygulama başlangıcı

- Codex ile case maddeleri, açık tasarım kararları, hata/retry sınırları ve üç bonus planlandı. Kullanıcı sqlc, TanStack Query ve CodeMirror kullanımını onayladı.
- Kullanıcının sağladığı OpenRouter/fal belgeleri ve resmi API sözleşmeleri incelendi. Önceki planın max_tokens örneğinin deprecated olduğu ve 403'ün her zaman auth hatası olmadığı fark edildi; max_completion_tokens ve ayrı içerik reddi sınıflandırmasıyla düzeltildi.
- River'ın varsayılan resumable checkpoint'lerinin job sonunda yazılması, fal request ID'nin hemen kaydedilmesine yetmiyor. Uygulamanın kendi DB fazları korundu; provider kabulü ile DB commit arasındaki belirsizlik açıkça sınır olarak kaydedildi.
- Çalışma klasöründeki eski tracked dosyalar başlangıçta silinmiş haldeydi. Yeni uygulama bu durum üzerinden kuruldu. GitHub hedefi kullanıcı tarafından https://github.com/CanhK-2103/trypix-flowkitCase olarak verildi.
- Başlangıçta ortamda provider anahtarları ve Go binary'si yok; Docker, Node ve GitHub erişimi mevcut. Gerçek sağlayıcı denemesi henüz yapılmadı.

## 2026-10-10 — uygulama ve doğrulama

- Go API/validator, pgx/sqlc storage, River faz motoru, OpenRouter/fal adaptörleri, artifact saklama, idempotency ve maliyet ledger'ı uygulandı. Generated sqlc kodu gerçek generator ile üretildi.
- Local HTTP mock'ları ve dedicated PostgreSQL üzerinde success/retry/permanent failure/CRUD/transaction rollback/idempotency testleri çalıştırıldı. Worker gerçek child process olarak request-ID commit sonrası, River Work bitmeden öldürüldü; restart testi submit/attempt/cost çoğalmasını reddetti ve geçti.
- İlk frontend API testleri ve TypeScript/build geçti. Docker Compose ilk temiz kurulumda 3 model/2 workflow yükledi; provider üretim çağrısı yapılmadı.
- Tarayıcıda save sonrası CodeMirror'ın server yanıtını onChange ile kullanıcı düzenlemesi sanıp dirty işaretini yeniden açtığı gözlendi. External update sırasında callback bastırıldı; jsdom regresyon testi eklendi. Ayrıca editor lazy-load ile başlangıç bundle'ı yaklaşık 714 kB'den 273 kB'ye ayrıldı (görsel kaliteye yönelik ek üretim yok).
- Regresyon testinin Element/HTMLElement typecheck hatası build sırasında yakalandı ve DOM selector tipi düzeltilerek giderildi. Yalnızca test geçmesine güvenilmedi; typecheck ve Docker build ayrıca kontrol edildi.
- Sağlayıcı HTTP fixture'ları gerçek response model ID, usage decimal precision, refusal/empty/truncated completion, fal error_type ve queue URL'lerini doğrular. Pricing ve media testleri de local mock kullanır.
- Son Compose rebuild sonrasında API sağlıklı kalırken gateway 502 döndü. Nginx'in startup'ta çözdüğü eski container IP'sini tuttuğu loglardan görüldü. Docker DNS resolver ve değişken proxy hedefi eklendi. Eski IP geçici bir test container'ıyla ayrıldı, API farklı IP ile yeniden oluşturuldu; gateway container kimliği değişmeden health bağlantısı yaklaşık 2,5 saniyede toparlandı. İlk upstream resolve denemesi 15 saniyelik toparlanma kontrolünü geçmediği için kullanılmadı.
- İlk GitHub CI checkout'ında `internal/artifacts` paketi eksikti: `.gitignore` içindeki `artifacts/` her seviyedeki aynı isimli klasörü dışlıyordu. Yerel testte dosya bulunduğundan görülmemişti. Kural `/artifacts/` olarak yalnızca çıktı klasörüne daraltıldı ve kaynak dosyası repoya eklendi; bu hata temiz checkout doğrulamasının önemini gösterdi.

## 2026-10-11 — anahtarlardan bağımsız teslim kontrolü

- Kullanıcı API anahtarlarını bütün bağımsız işler bittikten sonra gireceğini belirtti; canlı deneme istenmedi ve yapılmadı.
- Temiz checkout paketlemesini sürekli doğrulamak için CI'ya tam Compose kurulum kontrolü eklendi. Ücretli sağlayıcıya giden node içermeyen `input → output` workflow ile gerçek API/River worker ve idempotency kontrol edilir; bu run canlı AI kanıtı olarak sunulmaz.
- OpenAPI üretim farkı CI'ya eklendi. Önceki CI'daki Node 20 action uyarısı için resmi [checkout v7.0.1](https://github.com/actions/checkout/releases/tag/v7.0.1), [setup-go v7.0.0](https://github.com/actions/setup-go/releases/tag/v7.0.0), [setup-node v7.1.0](https://github.com/actions/setup-node/releases/tag/v7.1.0) etiketleri doğrulandı ve commit SHA'ları sabitlendi. Runner Ubuntu 24.04 olarak sabitlendi.
- Plan kontrol listesi uygulanmış/test edilmiş maddeler ile canlı sağlayıcı bekleyen maddeleri ayıracak şekilde güncellendi.

## 2026-10-11 — başlangıç tahmini ve gerçek toplam maliyet

- Kullanıcı tahmini/gerçek adım ücretlerini, üretim öncesi tahmini toplamı ve üretim sonrası gerçek toplamı ayrı istedi. Önceki uygulamanın fal ücretini yalnızca tahmin ettiğini açıkça belirttikten sonra kapsam genişletildi.
- Resmi fal Billing Events dokümanında source=billed ve request_id filtresi doğrulandı. Unit price × billable units gerçek tutar diye sunulmadı; indirim sonrası per-request cost_total kullanıldı. Billing erişimi için ek yetkiler gerektiği ve kayıt gecikmesinin olabileceği belgelenip unavailable/pending durumları uygulandı.
- OR usage.cost yoksa generation metadata fallback eklendi. Başlangıç tahmini paid çağrı olmadan katalog/parametreler ve açık token varsayımı ile hesaplanır; run kabulündeki snapshot değişmez.
- Gerçek toplamın önceki reported ara toplamından farklı olduğu kod/UI sözleşmesiyle ayrıldı: herhangi bir ücret eksikse actual total null olur. Mock testlerde gecikme, backoff, 403, pagination, kesin decimal, restart/refresh ve duplicate üretim olmaması doğrulandı; canlı ücret kanıtı üretildiği iddia edilmedi.
- OpenAPI düzenlemesinde initial_estimate alanı ilk olarak NodeRun bölümüne eklendi; üretilen şema kontrolünde fark edilip Run bölümüne taşındı. Frontend typecheck/build, engine ve PostgreSQL testleri ayrı kontrol edildi.

## 2026-10-11 — onaylı ilk canlı OpenRouter çağrısı

- Kullanıcı anahtarları yerel `.env` dosyasına ekledi ve her gerçek anahtar kullanımından önce ayrı onay istedi. Bu sınır AGENTS.md'ye kaydedildi; dosyada anahtar bulunması kullanım izni sayılmadı.
- Kullanıcı yalnızca bir OpenRouter completion isteğini onayladı. Otomatik fal/pricing/polling çağrıları da yapan mevcut direct komutu çalıştırılmadı; mevcut Go sağlayıcı adaptörü tek istekle kullanıldı. İstek öncesi ve yanıt sonrası ignored `evidence/local/direct.json` dosyasına checkpoint yazıldı.
- `openai/gpt-6-luna`, reasoning none ve max_completion_tokens 300 ile başarılı yanıt döndü. Gerçek response ID: `gen-1791671071-VEEBbsEAFLKL1RRnHBHh`; 58 input + 71 output token; sağlayıcının `usage.cost` tutarı `0.0000413 USD`; istek süresi 1671 ms. Bu tutar doğrudan sağlayıcı yanıtıdır, tahmin değildir.
- Fal üretimi ve tam FlowKit canlı kabulü henüz yapılmadı. Bu aşamada retry, billing veya başka anahtarlı sağlayıcı isteği gönderilmedi; sonraki kullanım ayrı kullanıcı onayı bekliyor.

## 2026-10-11 — onaylı ilk canlı fal kuyruk gönderimi

- Kullanıcı tek fal üretim isteğini ayrıca onayladı. Kaydedilmiş OpenRouter prompt'u ile `fal-ai/flux/schnell`, 512×512, 1 görsel, 4 inference step kullanılarak yalnızca bir queue submit yapıldı.
- Fal istek kimliği `01a127ed-1583-7c50-8eb7-0688dddc4466`; gönderim süresi 593 ms. Kimlik ve sağlayıcının status/result URL'leri herhangi bir sorgu yapılmadan `evidence/local/direct.json` dosyasında kalıcılaştırıldı. Faz `fal_waiting`; tamamlanma ve gerçek fal maliyeti henüz sorgulanmadı.
- Kullanıcıya bu doğrudan API deneyinin dosya checkpoint'i kullandığı ve henüz bir FlowKit DB run'ı oluşturmadığı açıklandı. FlowKit motorunun DB kayıtlarıyla doğrudan deneme kanıtı karıştırılmadı.
- Ek status/result/pricing/billing isteği ve retry gönderilmedi. Public model sayfasından kontrol edilen 0.003 USD tahmini gerçek fal ücreti olarak sunulmadı.

## 2026-10-11 — onaylı fal durum sorgusu

- Kullanıcının ayrı onayıyla kaydedilmiş fal status URL'si bir kez sorgulandı. Aynı `01a127ed-1583-7c50-8eb7-0688dddc4466` isteği `COMPLETED` ve hatasız döndü; sorgu 522 ms sürdü.
- Ham status yanıtı ve onay kapsamı `evidence/local/direct.json` içinde kalıcılaştırıldı; faz `fal_result_pending`. Toplam üç anahtarlı çağrı yapıldı: bir OpenRouter completion, bir fal submit ve bir fal status. Result, billing, ek polling veya yeni üretim çağrısı yapılmadı.

## 2026-10-11 — onaylı fal sonucu ve doğrudan denemenin tamamlanması

- Ayrı kullanıcı onayıyla aynı fal isteğinin result URL'si bir kez sorgulandı. 512×512 tek görsel, `X-Fal-Billable-Units=1` ve geçerli görsel URL'si geldi; sorgu 533 ms sürdü. Doğrudan deneme fazı `complete`; toplam dört anahtarlı HTTP çağrısı, birer LLM ve görsel üretimi var.
- Görsel URL'si ve ham yanıt ignored checkpoint'e yazıldı. Anahtarsız, gerekli alanlarla sınırlı kanıt `docs/evidence/direct.json` olarak hazırlandı; README eşleştirme tablosuyla ilişkilendirildi. Doğrudan deneme DB run'ı olarak sunulmadı.
- Billable unit değeri gerçek ücret sayılmadı; fal actual ve actual total null/pending olarak korundu. Billing veya ek üretim çağrısı yapılmadı.

## 2026-10-11 — koşullu onaylı fal faturalama sorgusu

- Kullanıcı yalnızca ek ücret oluşturmayacaksa bir faturalama sorgusuna izin verdi. Resmi Platform API/billing-events ve fiyat belgeleri incelendi: kayıt okuyan yönetim endpoint'i üretim çalıştırmıyor; bu okuma için ayrı bir bedel listelenmiyor. Ücretsiz okuma değerlendirmesi ile tek GET yapıldı; belgelerin açıkça sorgu fiyat garantisi verdiği iddia edilmedi.
- Mevcut sağlayıcı billing adaptörü kullanıldı, ancak HTTP transport en fazla bir GET'e ve yalnızca onaylanan request ID/model/source=billed filtresine izin verecek şekilde sınırlandı. Olası ek pagination isteği dahi yerelde engellenir; otomatik retry yok.
- Tek sorgu 475 ms'de HTTP 403 ve “This API key is not permitted to perform this action.” döndürdü. Kullanılan anahtar bu faturalama okumasına yetkili değil; üretim başarı durumu korunarak gerçek fal ücret ve gerçek toplam null/unavailable bırakıldı.
- Sonuç ignored checkpoint, anahtarsız kanıt ve kabul kayıtlarına yazıldı. Toplam beş anahtarlı HTTP çağrısı; üretim sayısı bir LLM ve bir görsel olarak kaldı. BILLING veya FULL yetkili ayrı FAL_BILLING_KEY olmadan gerçek fal ücreti doğrulanmış gibi gösterilmedi.

## 2026-10-11 — Docker yeniden açılışında yönetim servislerinin toparlanması

- Kullanıcı bilgisayarı kapatıp Docker Desktop'ı yeniden açtıktan sonra sistemin aktif olmasını istedi. PostgreSQL container'ı exited, API/worker restart döngüsünde ve gateway 502 durumundaydı; veritabanında restart politikası olmadığı görüldü.
- Gerçek çağrılar için verilen onay sınırı korunarak önce worker açıkça durduruldu. Mevcut volume'lar korunarak yalnızca db/init/api/bootstrap/dashboard servisleri başlatıldı; API ready ve dashboard HTTP 200 doğrulandı. Sağlayıcı isteği veya run başlatılmadı.
- PostgreSQL'e `restart: unless-stopped` eklendi ve çalışan container'ın restart politikası veriyi/container'ı yeniden oluşturmadan güncellendi. Compose konfigürasyonu doğrulandı; README'de worker kapalı yönetim başlangıcı ve kuyrukta bekleme davranışı belgelendi.

## 2026-10-11 — çağrı başına onaylı FlowKit kabul hazırlığı

- Kullanıcı kalan canlı kabul senaryolarına geçilmesini istedi; önceki her anahtar kullanımından önce ayrı onay şartı korundu. Standart worker pricing, polling, retry ve billing çağrılarını otomatik yaptığından tek completion onayıyla başlatılmadı.
- Yerel `cmd/approved-worker` kabul aracı aynı River job/Engine/PostgreSQL akışını kullanacak şekilde hazırlandı. Run'lar açıkça track edilir, credentialed çağrı öncesinde run sürümü/node/model/operation eşleşen tek kullanımlık onay beklenir. Onaysız beklemek domain attempt oluşturmaz; yeniden gönderim/polling de ayrı onay ister. Billing reconciler bu araçta çalışmaz.
- Fal historical pricing ve status tek Engine.Step içinde arka arkaya çağrılıyordu. Pricing checkpoint'i sonrasında River'a dönecek şekilde iki ayrı kalıcı adıma ayrıldı; request ID'nin polling/pricing öncesi commit edilme garantisi korundu.
- Yerel HTTP fixture'larıyla onaysız çağrı, yanlış/eski/tekrarlı onay, tek onayda tek HTTP çağrısı, 503 sonrası ayrı onayla retry ve tam başarılı Engine run'ı test edildi. HTTP/1 fresh connections, redirect engeli ve tek çağrı transport guard'ı ek isteği engelliyor.
- `gofmt`, tüm Go testleri ve dedicated PostgreSQL entegrasyon testleri, `go vet`, sekiz frontend testi, typecheck/build ve sqlc tekrar üretim kontrolü başarılı. Son transport ayarı sonrasında ilgili komut test/vet kontrolü tekrar geçti. Mock kanıtları canlı kabul sayılmadı; bu hazırlıkta gerçek sağlayıcı çağrısı yapılmadı.
- Commit `d597ac0` GitHub'a push edildi; üç CI job'ı başarılı. Ayrı boş `flowkit-acceptance` Compose ortamı 8081'de açıldı; standart worker başlatılmadı. API ile yalnızca prompt-writer/Schnell ve ürün workflow'u kaydedildi, yerel quote alındı ve idempotent run `a733fb1e-686e-4d78-8386-09ce5953a36a` oluşturuldu. Input succeeded; LLM pending/attempt 0; sonraki işlem tek OpenRouter completion ve onay bekliyor. Başlangıç tahmini 0.003157700000 USD, bunun LLM kısmı 0.000157700000 USD.
- Hazırlık scriptindeki PowerShell nested boş array `.Count` kontrolü yanlış pozitif verdi. Node bazında request sayıları toplanarak düzeltildi; DB/API kayıtlarında provider request 0 ve demo.approved_request log sayısı 0 doğrulandı. Run yeniden oluşturulmadı; checkpoint korundu. Canlı FlowKit üretimi hâlâ yapılmış sayılmıyor.

## 2026-10-11 — ilk FlowKit run'ında onaylı canlı LLM adımı

- Kullanıcı önce bildirilen tek `openai/gpt-6-luna` completion işlemini “evet” diyerek onayladı. Onay, DB run `a733fb1e-686e-4d78-8386-09ce5953a36a`, node `prompt_writer`, attempt 1 ve version 2'ye bağlandı; dispatch öncesinde ignored checkpoint'te tüketilmiş olarak kaydedildi.
- Kontrollü River worker aynı Engine üzerinden yalnızca bir OpenRouter POST gönderdi. Request ID `gen-1791724330-VzRw2gWX6NozFJbhqaTE`, actual model `openai/gpt-6-luna`, 58 input + 68 output = 126 token, süre 2142 ms. Sağlayıcının `usage.cost` gerçek tutarı `0.0000398 USD`; node/istek maliyeti DB'de exact decimal `0.000039800000` olarak doğrulandı.
- LLM prompt'u, model, ham yanıt, deneme ve maliyet normal run/node/provider request checkpoint'lerine kaydedildi. Başlangıç tahmini `0.003157700000 USD` değişmedi. Run hâlâ running, görsel node pending ve provider request sayısı 0; gerçek toplam null.
- API/approved-worker container ID, StartedAt ve image kimlikleri başlangıçla aynı. Tek approved request log'u var; fal submit/pricing/status/result, retry ve billing çağrıları yapılmadı. Sonraki operasyon fal Schnell submit ve ayrı kullanıcı onayı bekliyor. Anahtarsız kısmi kanıt `docs/evidence/flowkit-approved.json`; henüz tamamlanmış run kanıtı değildir.

## 2026-10-11 — ilk FlowKit run'ında onaylı fal gönderimi

- Kullanıcı tek `fal-ai/flux/schnell` queue submit işlemini ayrıca onayladı. Normal katalog/run snapshot'ı ve önceki LLM çıktısı kullanıldı: 512×512, 1 görsel, 4 inference step. Run version 5/node render/attempt 1'e bağlanan onay dispatch öncesi tüketildi.
- Kontrollü worker tek fal POST gönderdi; HTTP 200 döndü. `provider_request_id=01a12b1f-c88d-73a2-85f1-cce572dc075b`, status/result URL'leri ve `waiting_provider` fazı 2026-10-11T13:21:06Z'de PostgreSQL'e commit edildi. Submit başlangıcından commit'e 1421 ms. Request ID DB'den ayrıca okundu; checkpoint version 7.
- FlowKit oturumunda toplam iki anahtarlı HTTP çağrısı var (LLM ve fal submit). Pricing/status/result/billing veya retry yapılmadı. Başlangıç tahmini 0.003157700000 USD korundu; fal actual ve gerçek toplam null/pending. Sonraki operasyon provider pricing estimate; diğer çağrılar yeni açık onay bekliyor. Yerel ve anahtarsız kısmi kanıt güncellendi; üretim tamamlanmış olarak gösterilmedi.

## 2026-10-11 — insan onayı bekleyişinde demo deadline düzeltmesi

- Kullanıcı bir pricing, bir status ve COMPLETED ise bir result çağrısını açıkça onayladı. Çağrı öncesi DB kontrolünde görsel deadline'ın (13:31:04Z) insan onayı beklenirken geçtiği görüldü. Eski demo aracı onay bekleyişini domain/provider süresine dahil ediyordu; bu tasarım eksikliği Codex tarafından önceden öngörülebilecekken uygulanmamıştı. Hiçbir yeni sağlayıcı isteği gönderilmeden kontrollü worker durduruldu; run hâlâ waiting_provider ve bilinen fal ID korunuyordu.
- Kontrollü demo moduna kalıcı `approval_pause` eklendi: bekleme başlangıcı, özgün deadline, toplam insan bekleme süresi ve kaynak. Onayla devamda yalnızca kaydedilmiş bekleme süresi deadline'a eklenir; özgün değer korunur. Production worker'ın duvar saati timeout politikası değişmedi. Eski araçtaki mevcut bilinen request ID, kaydedilmiş 13:21:06.028360866Z commit log zamanı üzerinden açık/audit edilen geçişle benimsenir; sonuç, maliyet, model veya istek kimliği yeniden yazılmaz.
- Mock regresyon testinde 10 dakikadan uzun insan bekleyişleri sonrası aynı request ID, tek domain attempt ve tek provider request ile status/result tamamlandı. Aynı test standart Engine'de expired deadline'ın PROVIDER_TIMEOUT ile fail olduğunu, hiçbir HTTP çağrısı yapılmadığını doğruladı. Başlangıç tahmini ve özgün deadline korundu.
- Tam Go/PostgreSQL testleri, go vet, sqlc kontrolü, sekiz frontend testi ve typecheck/build geçti. Demo API/approved-worker bu düzeltme için yeniden oluşturuldu; hiçbir ek üretim veya sorgu tetiklenmedi. Bu restart model ekleme kabul demosu olarak sunulmaz; yeni model/workflow kabulü için baseline ilk run tamamlandıktan sonra alınacak.
