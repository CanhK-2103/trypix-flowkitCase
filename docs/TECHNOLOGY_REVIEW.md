# FlowKit teknoloji değerlendirmesi

Araştırma tarihi: 10 Ekim 2026. Durum: araştırma ve karar kaydı. Kullanıcı sqlc, TanStack Query ve CodeMirror 6 kullanımını onaylamıştır; bu üç seçim PLAN.md'ye işlenmiştir. Diğer araştırma seçenekleri benimsenmemiştir. Birinci ve ikinci maddede onaylanan kurulum/log kararları da PLAN.md'de yer alır. Uygulama kodlaması, bağımlılık kurulumu veya ücretli sağlayıcı çağrısı bu araştırmanın parçası değildir.

Resmi dokümantasyon, proje sahiplerinin repoları ve release sayfaları incelendi. Aşağıdaki tercih önerileri kaynaklarda anlatılan özelliklerle FlowKit gereksinimlerinin karşılaştırılmasından çıkan mühendislik değerlendirmeleridir. FlowKit üzerinde benchmark, uyumluluk denemesi veya çalışan prototip henüz yapılmadı. Değerlendirme ölçütleri: doğru transaction sınırları, crash/retry davranışını açıklayabilme, kod miktarı, ek servis ihtiyacı, dinamik katalog/graf desteği, test edilebilirlik ve tek komutla teslim.

## 1. Güncel sürüm görünümü

Bu tablo araştırma sırasında resmi sayfalarda görülen sürümleri kaydeder. Uygulama başlamadan önce patch sürümleri, container tag/digest'leri ve birlikte çalışabilirlik yeniden kontrol edilip sabitlenir; teslimde hareketli latest tag kullanılmaz.

| Teknoloji | 10 Ekim 2026'da görülen sürüm/durum | Projeye etkisi ve kaynak |
|---|---|---|
| Go | 1.27.2 stable | Go tercihi korunabilir. [Resmi indirmeler](https://go.dev/dl/) |
| PostgreSQL | 18.6 stable; 19 Beta 4 | Bu teslim için 18.x'in kararlı patch'i tercih edilir. [Resmi release listesi](https://www.postgresql.org/) |
| chi | 5.3.2 | Mevcut net/http router seçimi güncel. [Release](https://github.com/go-chi/chi/releases/tag/v5.3.2) |
| pgx | 5.11.0 | Yeni sürümün tarih/saat dönüşümü davranışları, UTC ve numeric round-trip kontrollerinde dikkate alınır. [Release](https://github.com/jackc/pgx/releases/tag/v5.11.0) |
| River Go | 0.49.0 | Resumable job ve transaction checkpoint özellikleri ayrıca incelenmiştir. Çok dilli repodaki latest etiketi Ruby release'ine işaret edebildiğinden Go tag'i ayrı kontrol edilir. [Go release](https://github.com/riverqueue/river/releases/tag/v0.49.0) |
| Goose | 3.28.0 | Mevcut SQL migration yaklaşımı sürdürülebilir. [Release](https://github.com/pressly/goose/releases/tag/v3.28.0) |
| React | 19.3; release 19.3.0 | Dashboard için kararlı React ailesi. [Resmi sürümler](https://react.dev/versions) |
| Vite | 8.1 duyurulmuş kararlı minor; 8 ailesi Rolldown kullanıyor | Yeni proje için güncel kararlı patch değerlendirilebilir. FlowKit build hızına ilişkin ölçüm yapılmadı. [Vite 8](https://vite.dev/blog/announcing-vite8), [Vite 8.1](https://vite.dev/blog/announcing-vite8-1) |
| TypeScript | Resmi indirme sayfası 7.0'ı latest olarak bildiriyor | React/Vite/editor bağımlılıklarıyla typecheck uyumu doğrulanarak patch sabitlenir. [Resmi indirme](https://www.typescriptlang.org/download/) |
| Node.js | 24.21.0 latest LTS; 26.11.1 Current | Dashboard build aşamasında 24 LTS önerilir. Statik dashboard sunumu Node süreci gerektirmez. [Resmi release durumu](https://nodejs.org/en/about/previous-releases), [Vite build yaklaşımı](https://vite.dev/guide/) |
| sqlc | 1.31.1 | Kullanımı onaylandı; SQL'den tipli pgx/v5 erişim kodu üretilecek. Kesin generator sürümü uygulama başlangıcında sabitlenecek. [Release](https://github.com/sqlc-dev/sqlc/releases/tag/v1.31.1) |
| Huma | 2.39.1 | chi üzerine OpenAPI/DTO katmanı için aday. [Release](https://github.com/danielgtaylor/huma/releases/tag/v2.39.1) |
| Caddy | 2.11.7 | Nginx yerine statik dashboard/reverse proxy/JSON log adayı. [Release](https://github.com/caddyserver/caddy/releases/tag/v2.11.7) |

## 2. Korunmasını önerdiğim temel seçimler

**Go + net/http + chi:** Go'nun ServeMux'u method/path routing destekleyen daha az bağımlılıklı bir seçenek. chi'nin route grupları ve mevcut middleware düzeni bu küçük API için yeterli; geçişten ölçülmüş bir kazanç yok. Fiber de değerlendirildi: fasthttp temeli sunuyor, ancak bu işte beklenen uzun işlem sağlayıcı beklemesi. Router değişiminin bütçe veya motor dayanıklılığına belirgin katkısı beklenmiyor. Bu bir iş yükü değerlendirmesidir. [ServeMux dokümantasyonu](https://pkg.go.dev/net/http#ServeMux), [Fiber v3](https://docs.gofiber.io/next/).

**PostgreSQL + River:** Run kaydı ile işin aynı transaction içinde commit edilmesi asıl avantaj. Asynq Redis temelli bir alternatif; PostgreSQL run kaydıyla Redis enqueue işlemi arasındaki atomiklik için ek düzen gerekir. Temporal ayrı bir orchestration servisi/SDK, Restate ise Go SDK ve kendi durable journal yaklaşımını sunuyor. Bu sistemlerle veri olarak verilen grafı yorumlamak mümkün; küçük doğrusal case'te kurulum ve açıklanacak yürütme modeli büyüyor. River'ın PostgreSQL transaction yaklaşımı daha uygun görünüyor. [River transactional enqueue](https://riverqueue.com/docs/transactional-enqueueing), [Asynq](https://github.com/hibiken/asynq), [Temporal deployment](https://docs.temporal.io/production-deployment), [Restate Go](https://docs.restate.dev/develop/go/services), [Restate runtime](https://restate.dev/).

**Goose + açık SQL:** Atlas migration üretimi ve schema diff için incelendi. Kısa, okunabilir SQL migration'ları bu kapsam için yeterli. Atlas'ın önceden onaylanan schema plan özelliği Pro olarak belgeleniyor; ilk teslim bu özelliğe ihtiyaç duymuyor. Goose/sqlc beraber kullanılabiliyor. [Goose](https://pressly.github.io/goose/), [Atlas versioned diff](https://atlasgo.io/versioned/diff), [Atlas schema plan](https://atlasgo.io/declarative/plan), [Goose/sqlc birlikte kullanım](https://pressly.github.io/goose/blog/2024/goose-sqlc/).

**JSON Schema + özel graf validator:** Mevcut santhosh-tekuri/jsonschema kütüphanesi v6 ailesinde 2020-12 desteğini belgeliyor. Huma veya editor lint katmanı döngü, önceki node referansı, katalog ve port ilişkilerini kendiliğinden doğrulamaz. Tek semantik doğrulayıcı korunmalı; API create/update/validate ve bootstrap aynı kuralları kullanmalı. [Kütüphane](https://github.com/santhosh-tekuri/jsonschema).

**React + TypeScript + Vite:** Yönetim panelinin SSR/SEO ihtiyacı tanımlı değil. Next.js statik export ile de çalışabiliyor; dolayısıyla zorunlu Node sunucusu gerektirdiği varsayılmıyor. Mevcut Vite SPA, runtime oluşan workflow/run kimliklerine ilişkin ekranları basit tutuyor. Vite 8'in yeni bundler'ı mevcut seçimin güncel kalabildiğini gösteriyor. [Next.js static export](https://nextjs.org/docs/app/guides/static-exports), [Vite 8](https://vite.dev/blog/announcing-vite8).

**Go slog + kalıcı run izi:** JSONHandler satır başına JSON ve özelleştirilebilir alanlar sunuyor; ayrıntılı log kararımızı karşılıyor. OpenTelemetry Go'da traces/metrics stable, logs ise resmi durum tablosunda release candidate. İlk teslimde Collector/Jaeger/Loki eklemek yerine ortak slog ve DB izi daha küçük bir operasyonel yapı sağlar. OpenTelemetry trace/metrics sonraki gelişim adayıdır. [slog](https://pkg.go.dev/log/slog), [OpenTelemetry Go durumu](https://opentelemetry.io/docs/languages/go/).

**Polling + yerel artifact volume:** Case sonucu iletme ve görsel saklama kararını bize bırakıyor. İki saniyelik polling ile tek host volume gereksinimleri karşılıyor. SSE/WebSocket ve object storage çoklu istemci/host veya canlı event ihtiyacı oluştuğunda yeniden değerlendirilebilir. İlk teslimde temel kabul testlerine katkı sağlayan değişikliklere öncelik verilir.

## 3. Değerlendirilen araçlar ve kararları

### A. pgx'in yanına sqlc — onaylandı

SQL dosyalarından tipli Go parametre/sonuç kodu üretir. pgx/v5 ve transaction'a bağlı Queries kullanımını destekler. Sorgu yazımı, unique constraint, expected-state UPDATE ve River InsertTx transaction'ı görünür kalır. Manuel Scan/alan sıralaması kodu azalabilir. [Üretim ve pgx desteği](https://docs.sqlc.dev/en/latest/howto/generate.html), [Transaction kullanımı](https://docs.sqlc.dev/en/latest/howto/transactions.html).

Yeni model veya workflow yalnız yeni veri satırlarıdır; sqlc generate gerektirmez. Kod üretimi SQL/schema değişikliğine bağlıdır. Generated kod repoya girer; geliştirme/CI için generator sürümü sabitlenir. JSONB, UUID, UTC timestamp ve NUMERIC(24,12) eşlemeleri ile run/job rollback davranışı kontrol edilecektir. Onaylanan kapsam PLAN.md Bölüm 7.1'de tanımlıdır: db/queries, internal/storage/dbgen, sqlc.yaml, ortak pgx transaction ve CI yeniden üretim/fark kontrolü. Çalışan entegrasyon henüz doğrulanmadı.

### B. Nginx yerine Caddy — benimsenmedi; Nginx korunuyor

Caddy statik dosya sunumu ve reverse proxy yanında JSON erişim/runtime log yapılandırmasını destekler. İkinci maddede kararlaştırılan log kapsamına uymak için az konfigürasyonla sonuç verebilir; ek kalıcı servis gerektirmez. [Statik dosya](https://caddyserver.com/docs/caddyfile/directives/file_server), [Reverse proxy](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy), [Access log](https://caddyserver.com/docs/caddyfile/directives/log), [Runtime log yapısı](https://caddyserver.com/docs/logging).

Karşılaştırmada localhost:8080 HTTP erişimi, API/artifact/health path'leri, SPA fallback, request_id aktarımı ve JSON loglar kontrol edilmelidir. API 404'ünün frontend HTML cevabına dönüşmemesi gerekir. Basic password protection eklenirse hem API hem dashboard kapsanır. Caddy'nin kendi logları ile Go run loglarının alan kapsamı ayrıca belgelenir; gateway her worker bağlamını kendiliğinden bilmez. Bu araştırmada Nginx kararı henüz değiştirilmedi.

### C. Dashboard veri yönetimi için TanStack Query — onaylandı

Aktif run polling'ini ve terminal durumda durmayı query durumuna göre yönetebilir; liste/detail cache ve kaydetme sonrasında invalidation kodunu toparlayabilir. [Resmi polling rehberi](https://tanstack.com/query/latest/docs/framework/react/guides/polling), [useQuery](https://tanstack.com/query/latest/docs/framework/react/reference/functions/useQuery).

Onaylanan FlowKit ayarları PLAN.md Bölüm 14.1'dedir: aktif run interval 2000 ms, terminal durumda false; GET ağ/geçici 5xx için en fazla bir ek deneme, 1 saniye bekleme ve kalıcı 4xx için retry yok; refetchOnWindowFocus false; bütün mutation'larda otomatik retry false. Kullanıcı/ağ kaynaklı aynı run isteğinde mevcut Idempotency-Key tekrar kullanılır. Frontend sorgu retry'si provider node attempt sayacından ayrıdır. Üretim başlatan POST useQuery ile yönetilmez. Query cache'i kaydedilmemiş editor metnini ezmez.

### D. JSON editörü için CodeMirror 6 — onaylandı

JSON language/parser ve lint diagnostic eklentileriyle sözdizimi hataları editörde gösterilecektir. Backend findings listesi ayrı doğrulama sonucudur; semantik kurallar tek API doğrulayıcısında kalır. Onaylanan kapsam PLAN.md Bölüm 14.2'de React editor yaşam döngüsü, bozuk JSON için gönderimi engelleme, API üzerinden validate/save ve JSON Pointer bulgularını gösterme olarak tanımlıdır. [JSON paketi README/API](https://raw.githubusercontent.com/codemirror/lang-json/main/README.md), [Lint rehberi](https://codemirror.net/examples/lint/).

Araştırmada GitHub lang-json reposunun arşivlenmiş olduğu görüldü; resmi README bunun yeni upstream'e taşındığını açıkça bildiriyor. Kurulumda güncel upstream/tag/npm paketi kontrol edilir. Eski GitHub arşivi tek başına bakımın bittiği kanıtı olarak alınmaz. Monaco da VS Code temelli güçlü editor seçeneğidir; schema IntelliSense kapsamı büyürse yeniden değerlendirilir. İki editor için FlowKit bundle boyutu ölçülmedi. [Monaco resmi repo](https://github.com/microsoft/monaco-editor).

### E. chi üzerine Huma — benimsenmedi; chi korunuyor

Huma router bağımsız OpenAPI 3.1/JSON Schema katmanı ve tipli handler'lar sunar; chi ile beraber kullanılabilir. Endpoint dokümanının handler sözleşmesine bağlı üretilmesi manuel OpenAPI bakımını azaltabilir. [Huma özellikleri ve router desteği](https://huma.rocks/why/).

Varsayılan hata gövdesi RFC 9457 Problem Details'tır; mevcut error/findings sözleşmemizle eşleştirme gerekir. Özelleştirilebilir hata modeli belgelenmiştir. Framework'ün HTTP body doğrulamasının semantik validator'dan önce durarak bağımsız workflow bulgularını gizlememesi gerekir. Create/PUT 422 ve standalone validate 200 davranışları, decimal string/null alanları, dynamic params/inputSchema ve artifact streaming açıklığı korunmalıdır. Bu uyum sağlanmadan benimsenmiş sayılmaz. [Hata özelleştirme](https://huma.rocks/features/response-errors/), [Request validation](https://huma.rocks/features/request-validation/).

### F. River resumable/checkpoint yardımcıları — benimsenmedi; mevcut DB fazları korunuyor

River tamamlanan named step'leri atlama ve transaction içinde checkpoint kalıcılaştırma API'leri sunuyor. Kritik ayrıntı: varsayılan ilerleme kaydı worker attempt'i bittikten sonra kalıcılaşır; süreç arada çökerse ilerleme kaybolabilir. Transaction checkpoint'i uygulama yazılarıyla commit edilebilir; dış yan etkiler yine idempotency/gönderim-belirsizliği yönetimi gerektirir. [Resumable jobs ve durable checkpoints](https://riverqueue.com/docs/resumable-jobs).

Fal provider_request_id aynı kısa DB transaction'ında hemen kaydedilmelidir; bunun polling veya job sonunda yazılmasına izin verilmez. DB'deki node/provider fazları, toplam üç domain attempt ve maliyet ledger'ı doğruluk kaynağı olarak kalır. Yardımcı API'ler gerçekten daha az kodla bu kuralları koruyorsa kullanılabilir; çift checkpoint sahibi oluşturarak karmaşıklığı artırıyorsa mevcut açık state machine tercih edilir. Ücretli River Pro Workflows bu değerlendirme için gerekli değildir.

## 4. Onaylanan kararlar ve uygulama sırası

1. Uygulama başlangıcında seçilen kararlı sürümleri Go/Node/container/lock dosyaları ve sqlc generator sürümüyle birlikte sabitlemek.
2. Go/DB aşamasında sqlc/pgx erişimini kurmak; generated kodu repoya almak ve JSONB/UUID/UTC/null/decimal dönüşümlerini doğrulamak.
3. Run acceptance aşamasında sqlc kayıtları ile River job'ın aynı transaction'da commit/rollback olduğunu kanıtlamak.
4. Dashboard aşamasında TanStack Query ve CodeMirror'ı uygulamak; terminal polling, mutation/idempotency ve syntax/backend findings davranışlarını kontrol etmek.
5. Nginx, chi ve uygulamanın DB faz yaklaşımını korumak; Caddy, Huma veya River checkpoint helper entegrasyonu eklememek.

Bu rapor araştırma gerekçeleri ile kullanıcı tarafından onaylanan seçimleri ayırır. PLAN.md'nin mimari tablosu, veri erişimi/dashboard sözleşmeleri, proje düzeni, uygulama sırası ve kabul kontrolleri sqlc/TanStack Query/CodeMirror ile güncellenmiştir.

10 Ekim 2026 uygulama notu: seçilen araçlar kuruldu ve sürümleri sabitlendi. sqlc transaction/type dönüşümleri ve yeniden üretimi, TanStack Query polling/request davranışları ve CodeMirror güncelleme davranışı test edildi. Çalıştırılmış kanıtlar ve bekleyen gerçek sağlayıcı kontrolleri [ACCEPTANCE.md](ACCEPTANCE.md) dosyasında tutulur.
