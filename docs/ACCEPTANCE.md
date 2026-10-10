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

## Henüz doğrulanmamış canlı kabul

- Gerçek doğrudan OpenRouter → fal denemesi.
- Aynı modeller/prompt/parametrelerle FlowKit run'ı.
- Schnell → API ile Dev kaydı → modelRef güncelleme → gerçek Dev run'ı; servis restart olmadan.
- İkinci workflow'un gerçek Dev run'ı.
- Gerçek sağlayıcı harcamaları ve fiyat snapshot'ları.

Başlangıçta yerel API anahtarları yoktu. Kullanıcının 11 Ekim 2026 isteğiyle anahtarların girilmesi ve canlı kabul en sona bırakıldı. Anahtarlar hazır olunca `cmd/direct` ve `cmd/acceptance` çalıştırılır; çıktılar run/request kimlikleriyle bu dosyaya eklenir. Mock kanıtı gerçek üretim kanıtı olarak sunulmaz.
