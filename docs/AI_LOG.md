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
