# Çağrı başına onaylı yerel kabul demosu

Bu araç kullanıcının her gerçek anahtar kullanımını ayrıca onaylama şartı için hazırlanmıştır. Normal kurulumun bir zorunluluğu veya dashboard özelliği değildir. Standart worker otomatik çalışır; bu yerel araç aynı PostgreSQL, River ve Engine üzerinde sağlayıcı çağrılarını tek tek bekletir.

## Başlangıç

PowerShell; mevcut `.env` anahtarları sadece backend container environment'ına yüklenir:

```powershell
$env:DASHBOARD_PORT='8081'
$env:LOAD_EXAMPLES='false'
docker compose -p flowkit-acceptance -f compose.yaml -f compose.approval.yaml up -d --build --wait db init api bootstrap dashboard approved-worker
```

Standart `worker` bu projede başlatılmaz. Dashboard/API `http://localhost:8081`; yerel operatör kontrolü `http://localhost:8082`. Her iki port yalnızca 127.0.0.1'e bağlıdır. Başlatma gerçek üretim, pricing, polling veya billing çağrısı yapmaz. Kontrol endpoint'leri genel kullanıma açılmamalıdır.

Başlangıçta sadece `prompt-writer`, `flux-schnell` ve ürün workflow'u örnek JSON dosyalarından HTTP API ile eklenir. İlk gerçek run tamamlandıktan sonra aynı çalışan servislerde `flux-dev` kaydedilir ve ürün workflow'unun modelRef'i API'den değiştirilir. Son olarak etkinlik workflow'u API'den eklenir. API/approved-worker container ID, StartedAt ve image kimlikleri önce/sonra kanıt olarak karşılaştırılır.

## Bir run'ı kontrollü yürütme

1. `/workflows/{id}/estimate` yerel fiyat snapshot'ıyla üretim öncesi tahmini verir; sağlayıcıya çağrı yapmaz.
2. `POST /workflows/{id}/runs` ve `Idempotency-Key` run/River job'ını normal API transaction'ında oluşturur.
3. `POST http://localhost:8082/track` gövdesi `{"run_id":"..."}`. Worker bu run'ın yerel fazlarını işler; sağlayıcıya gitmeden bekler.
4. `GET http://localhost:8082/next?run_id=...` bir sonraki çağrının run/version/node/attempt/provider/model/action/method/url alanlarını döndürür. Bu GET sağlayıcı anahtarı kullanmaz.
5. Operatör sağlayıcı/model, işlem ve tahmini bedeli kullanıcıya bildirir; açık onay gelmeden sonraki adım yapılmaz.
6. `POST http://localhost:8082/approve` gövdesi `{"operation":NEXT_RESPONSE,"approval_id":"unique-user-approval-reference"}`. `NEXT_RESPONSE` önceki GET'ten alınan nesnenin tamamıdır.
7. Onay yalnızca o run sürümü ve tek sağlayıcı HTTP çağrısı içindir. Worker onayı dispatch'ten önce tüketir, sonucu normal Engine/DB checkpoint'ine kaydeder. Sonraki işlem, ilave polling ve retry yeni onay bekler.

Fal pricing ve status ayrı kalıcı worker adımlarına ayrılmıştır. Request ID submit sonrasında yine hemen commit edilir. Onay beklemek domain attempt veya sağlayıcı isteği oluşturmaz. Eski sürüme/başka node'a ait onay ve aynı approval ID'nin tekrarı reddedilir. Bir transport çağrısı dışında hiçbir credentialed HTTP isteğine izin verilmez; redirect ve bağlantı yeniden kullanımı kapalıdır. Onaylar bellektedir; worker yeniden açılırsa yeniden onay gerekir, otomatik devam etmez.

Standart node deadline'ları korunur (LLM 120 saniye, görsel 600 saniye). Görsel submit sonrası onay bekleyişi de bu süreye dahildir. Süre aşılırsa run standart hata politikasıyla başarısız olur; geçmiş deadline değiştirilmez veya bilinen request ID silinmez.

Görsel sonucu kaydedildikten sonra anahtarsız medya indirme ve yerel output fazları otomatik tamamlanır. Bu araç billing reconciler başlatmaz. Gerçek fal ücretini öğrenmek için ayrıca onaylı, exact request ID'ye ait billing isteği ve uygun yetkili anahtar gerekir. Ücret bilinmiyorsa actual total null kalır.

## Kanıt ve test

`cmd/approved-worker/main_test.go` sadece yerel HTTP fixture'ları kullanır: onaysız hiçbir çağrı olmaması, yanlış/eski/tekrarlanan onayın reddi, tek onayla tek çağrı, 503 retry için yeni onay ve başarılı tam Engine akışı. Mock sonuçları canlı kanıt olarak sunulmaz. Canlı kabul checkpoint'leri `evidence/local/` altında, paylaşılacak anahtarsız kanıtlar `docs/evidence/` altında saklanır.
