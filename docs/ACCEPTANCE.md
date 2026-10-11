# Kabul durumu

Bu dosya gerçek kanıtı planlanan kontrollerden ayırır. Testler ücretli sağlayıcı çağrısı yapmaz.

## 10 Ekim 2026 — çalıştırılmış kontroller

- Go paketleri derlendi; graf doğrulama ve motor testleri geçti.
- Dedicated PostgreSQL 18.6 üzerinde katalog/workflow CRUD, doğrulama findings listesi, geçersiz update sonrası revision korunması, model devre dışı bırakma ve soft-delete testleri geçti.
- Aynı idempotency key ile 10 eşzamanlı istek: yalnızca 1 run ve 1 River job; diğerleri aynı run'ı döndürdü. Farklı input conflict ve workflow silindikten sonra replay doğrulandı.
- Run ile River job aynı transaction'da rollback oldu; orphan job/run oluşmadı. NUMERIC(24,12), UUID, JSONB ve UTC timestamp dönüşümleri kontrol edildi.
- Gerçek River worker ve local HTTP mock'larıyla başarı, transient 503 → retry → başarı, kalıcı image 422 → doğru node/error code ile failure doğrulandı. Kalıcı görsel API'den okundu.
- Worker child process, fal request ID commit edildikten sonra fakat River Work dönmeden zorla sonlandırıldı. İkinci process aynı DB'den devam ederek tamamladı: 1 fal submit, 1 LLM call, 1 image domain attempt, 1 provider request ve 1 maliyet kaydı.
- Çalışan API/worker değiştirilmeden yeni compatible model alias'ı API'den kaydedildi, workflow'un modelRef'i API ile değiştirildi ve mock run yeni endpoint ile tamamlandı. İkinci workflow da HTTP ile eklendi.
- Frontend TypeScript/build, 4 request/polling testi ve CodeMirror external update/user edit regresyon testi geçti.
- `docker compose up --build -d` temiz volume'larla çalıştı; API ready, worker, dashboard, 3 model ve 2 workflow doğrulandı. Bootstrap tekrar çalıştırıldı; mevcut workflow revision 3 ve name korundu.
- Tarayıcıda JSON doğrulama/kaydetme, save sonrası dirty temizlenmesi ve bozuk JSON için butonların kapanması kontrol edildi. İlk kurulum ekranı dar/masaüstü görünümünde incelendi.
- API container'ı farklı Docker IP'si ile yeniden oluşturuldu. Gateway container'ı değiştirilmeden `/health/ready` yaklaşık 2,5 saniyede başarılı oldu; models/workflows API erişimi tekrar kontrol edildi.
- API/worker/init/bootstrap çıktılarındaki 1157 satır parse edildi; JSON olmayan veya run_id/node_id/attempt alanı eksik satır sayısı 0. Ayrıca logger context alanlarının duplicate/null ile ezilmediği test edildi.
- sqlc 1.31.1 tekrar üretiminde generated dosyaların hash'leri değişmedi; go vet geçti. Frontend production dependency audit 0 açık bildirdi.

## 11 Ekim 2026 — anahtarsız temiz paket kontrolü

- Ayrı `flowkit-package-check` Compose projesi yeni volume'larla `up --build --detach --wait` kullanılarak açıldı; normal çalışma ortamı korunarak kontrol edildi.
- Gateway üzerinden API ready, derlenmiş dashboard JavaScript dosyası, 3 enabled model ve 2 örnek workflow doğrulandı; iki örneğin standalone validate sonucu geçerliydi.
- Gerçek API ve River worker üzerinde sağlayıcısız `input → output` run'ı tamamlandı. Aynı idempotency key ile replay aynı run'ı verdi. Her iki node succeeded, provider request sayısı 0, maliyet `0.000000000000` ve cost_status complete.
- Bu kontrol `tools/check-compose.mjs` ile CI'ya eklendi. Bu run AI üretimi değildir; aşağıdaki canlı kanıtların yerine geçmez.

## 11 Ekim 2026 — başlangıç tahmini ve gerçek maliyet ek kapsamı

- `/workflows/{id}/estimate` üretimden önce node ve toplam tahmini döndürüyor; run/provider çağrısı oluşturmadığı PostgreSQL + local HTTP fixture testinde doğrulandı. Başlangıç tahmini run snapshot'ında korunuyor.
- Fal billed-event isteği request ID, endpoint ID ve source=billed filtrelerini gönderiyor; pagination, indirim sonrası tutar, nano USD, zero gerçek ücret, eksik/malformed kayıt, farklı request ID ve 403/503 yolları local mock ile testli. OpenRouter generation fallback exact decimal tutarı testli.
- Gerçek toplam ücret kaydı eksikken null/pending. Geciken fal kaydı ve sonraki API refresh/reconciliation sonrasında gerçek toplam `0.002720000000`, fal `0.002700000000`, OpenRouter `0.000020000000` olarak DB'de doğrulandı; bunlar **mock tutarlarıdır**. Submit/LLM sayıları birer kaldı; tekrar uzlaştırma double-charge oluşturmadı.
- Billing yetki hatası successful run'ı değiştirmedi; erişim düzeltildikten sonra refresh çalıştı. Geçici 5xx backoff, 24 saatlik bekleme sınırı, DB checkpoint'ten devam etme ve başlangıç tahmininin korunması engine testleriyle doğrulandı.
- Frontend sekiz test, typecheck/build; tam Go/PostgreSQL entegrasyon testleri, go vet ve sqlc tekrar üretim kontrolü geçti. Docker Compose yeniden build ile sağlıklı başladı; gateway/gerçek worker üzerinde ücretsiz input→output smoke kontrolü geçti.
- Gerçek fal billing key yetkileri ve canlı faturalama tutarları henüz doğrulanmadı; anahtar girişi kullanıcı isteğiyle en sona bırakıldı.

## 11 Ekim 2026 — ilk canlı OpenRouter adımı

- Kullanıcı onayıyla yalnızca bir `POST /api/v1/chat/completions` çağrısı yapıldı. Requested ve actual model `openai/gpt-6-luna`; reasoning none, max_completion_tokens 300.
- Girdi: “Ahşap saplı, mat mavi seramik kahve kupası.” Model İngilizce ürün görseli prompt'u üretti; yanıt 1671 ms'de geldi.
- Provider request ID: `gen-1791671071-VEEBbsEAFLKL1RRnHBHh`. Kullanım: 58 input, 71 output, toplam 129 token. Yanıttaki gerçek `usage.cost`: `0.0000413 USD`.
- Gönderim checkpoint'i ve ham sağlayıcı yanıtı ignored `evidence/local/direct.json` dosyasında. Bu OpenRouter adımında fal çağrısı yapılmadı; ayrı onayla sonraki gönderim aşağıda kaydedildi.

## 11 Ekim 2026 — ilk canlı fal kuyruk gönderimi

- Ayrı kullanıcı onayıyla yalnızca bir `POST https://queue.fal.run/fal-ai/flux/schnell` çağrısı yapıldı. Girdi önceki OpenRouter çıktısı; 512×512, 1 görsel, 4 inference step.
- Sağlayıcı kuyruğu isteği kabul etti: `provider_request_id=01a127ed-1583-7c50-8eb7-0688dddc4466`. Gönderim 593 ms. Request ID ve status/result URL'leri derhal ignored `evidence/local/direct.json` dosyasına yazıldı.
- Faz `fal_waiting`. Status/result sorgulanmadığından tamamlanmış görsel kanıtı henüz yok; 0.003 USD public fiyat tahmini gerçek billed ücret olarak sunulmuyor.
- Bu doğrudan deneme dosya checkpoint'i kullanır; FlowKit run'ı veya DB üretim kaydı henüz oluşturulmadı. DB üzerinden canlı kabul aşağıda bekliyor.

## 11 Ekim 2026 — ilk canlı fal durum sorgusu

- Ayrı kullanıcı onayıyla aynı fal request ID'nin kaydedilmiş status URL'si yalnızca bir kez sorgulandı. `COMPLETED` döndü; sağlayıcı hatası yok, sorgu süresi 522 ms.
- Status yanıtı ignored `evidence/local/direct.json` içinde. Güncel faz `fal_result_pending`; sonuç/görsel henüz alınmadı. Yeni üretim, retry, ek polling veya billing sorgusu yapılmadı.

## 11 Ekim 2026 — ilk doğrudan zincirin tamamlanması

- Ayrı kullanıcı onayıyla aynı request ID'nin result URL'si yalnızca bir kez sorgulandı; 533 ms'de 512×512 tek görsel ve billable units 1 döndü. Güncel faz `complete`.
- Görsel: https://v3b.fal.media/files/b/0aaddf91/LSonJO8IbWMVUppDPmUY7.jpg. Anahtarsız [doğrudan deney kanıtı](evidence/direct.json), model/prompt/parametre eşleştirmesi README'de. Ham yanıt yerel ignored checkpoint'te.
- Bu deney toplam dört anahtarlı HTTP çağrısıyla yapıldı: 1 OpenRouter completion, 1 fal submit, 1 fal status, 1 fal result. Billing ve ek üretim yapılmadı.
- OpenRouter gerçek yanıt tutarı 0.0000413 USD; fal actual cost ve gerçek toplam null/pending. Üretimin tamamlanması billing doğrulaması sayılmadı. FlowKit DB run'ı henüz yok.

## 11 Ekim 2026 — canlı fal faturalama yetki kontrolü

- Koşullu ayrı kullanıcı onayıyla mevcut request ID, `fal-ai/flux/schnell` ve `source=billed` filtreleriyle tek read-only billing-events GET isteği yapıldı. Transport ilave istekleri (pagination dahil) yerelde engelleyecek şekilde sınırlandı.
- HTTP 403: “This API key is not permitted to perform this action.”; süre 475 ms. Mevcut anahtarın faturalama okuma erişimi olmadığı canlı olarak görüldü. Resmi endpoint BILLING/FULL preset veya `billing:usage:read`, `auth:keys:read`, `account:users:read` yetkileri gerektiriyor.
- Güncel gerçek maliyet durumu unavailable; fal actual ve gerçek toplam null. Üretim complete olarak kaldı. Toplam beş anahtarlı HTTP çağrısı; ilave model üretimi veya retry yapılmadı. Anahtarsız kanıt erişim hatasını da içeriyor.

## FlowKit canlı kabul — tamamlanan ve bekleyen

Çağrı başına onaylı kabul hazırlığı tamam: ayrı boş `flowkit-acceptance` ortamında API ve kontrollü River worker çalışıyor (dashboard 8081, yerel kontrol 8082). Yalnızca ilk iki model ve ürün workflow'u API'den eklendi. Başlangıçta provider request/onaylı sağlayıcı çağrısı 0 olarak doğrulandı. `d597ac0` commit'inin backend/dashboard/compose CI kontrolleri başarılı.

Kullanıcının ayrı onayıyla run `a733fb1e-686e-4d78-8386-09ce5953a36a` içindeki LLM adımı canlı tamamlandı: actual model `openai/gpt-6-luna`, request `gen-1791724330-VzRw2gWX6NozFJbhqaTE`, 1 attempt, 2142 ms, 58 input + 68 output token; gerçek `usage.cost` ve DB kaydı `0.000039800000 USD`. Başlangıç tahmini `0.003157700000 USD` korundu. API/worker container kimlikleri ve başlangıç zamanları aynı.

Sonraki ayrı onayla fal Schnell submit (512×512, 1 görsel, 4 step) yapıldı: request `01a12b1f-c88d-73a2-85f1-cce572dc075b`, 1 attempt, HTTP 200; request ID ve queue URL'leri 1421 ms içinde `waiting_provider` fazıyla DB'ye commit edildi, version 7 doğrudan DB sorgusunda doğrulandı.

Üç okuma ayrıca onaylandı: pricing POST başarılı/0.003 USD tahmini; tek status GET COMPLETED, tek result GET geçerli görsel. Onay beklenirken 600 saniyelik duvar saati deadline'ı aşıldığı için demo worker'ın insan bekleme saati düzeltildi; API/worker bu düzeltme sırasında yeniden oluşturuldu. Özgün deadline `2026-10-11T13:31:04.606542458Z`, kaydedilen insan bekleme süresi 1618044 ms; aynı request ID, tek image attempt ve tek submit korundu. Bu düzeltme runtime model kabulünün restart olmadan gerçekleştiği kanıtı olarak sunulmaz.

Run succeeded; input/LLM/image/output succeeded. Kalıcı artifact `70266b82-d304-4ef6-80c1-9696e044f5d1`, 512×512 JPEG/133147 byte, model `fal-ai/flux/schnell`; gateway GET HTTP 200. DB'de birer LLM/image attempt, bir artifact ve değişmemiş başlangıç tahmini `0.003157700000` doğrulandı. Çalıştırma tahmini `0.003039800000`, OpenRouter actual `0.000039800000`; fal actual/actual total null/pending. Toplam 5 credentialed çağrı (completion, submit, pricing, status, result); ek polling/retry/billing yapılmadı. [Tamamlanan ilk run kanıtı](evidence/flowkit-approved.json).

Kalan canlı kabul:

- Schnell → API ile Dev kaydı → modelRef güncelleme → gerçek Dev run'ı; servis restart olmadan.
- İkinci workflow'un gerçek Dev run'ı.
- Fal gerçek harcamaları ve kalan canlı run'ların fiyat/maliyet kayıtları; ilk OpenRouter yanıt tutarı yukarıda doğrulandı.

Başlangıçta yerel API anahtarları yoktu. Anahtarlar girildikten sonra kullanıcı her gerçek anahtar kullanımı için ayrı onay şartı koydu. Canlı adımlar yalnızca açıkça onaylanan çağrılarla yürütülür; otomatik ek çağrı yapan `cmd/direct` veya `cmd/acceptance` bu çağrıların kapsamı ayrıca onaylanmadan çalıştırılmaz. Çıktılar run/request kimlikleriyle bu dosyaya eklenir. Mock kanıtı gerçek üretim kanıtı olarak sunulmaz.
