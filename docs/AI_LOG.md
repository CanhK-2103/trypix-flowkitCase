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
