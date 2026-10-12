# VPS deployment

Mevcut Go API, River worker, PostgreSQL ve React dashboard aynı Compose tanımıyla tek Ubuntu 24.04 VPS üzerinde çalışır. Caddy 2.11.7 HTTPS sertifikasını otomatik alıp yeniler. Production'a özel `flowkit gate` süreci kullanıcı adı istemeyen şifre formunu sunar; dashboard, API, health ve görseller aynı oturum korumasındadır. PostgreSQL, API, dashboard ve gate için host portu yayınlanmaz. Caddy admin API kapalıdır. Gate DB veya provider kullanmaz; yerel Compose davranışı değişmez.

Proxy yalnız TCP portlarını yayınlar; [Caddy protocols](https://caddyserver.com/docs/caddyfile/options#protocols) ayarı `h1 h2` ile sınırlıdır. Kullanılmayan HTTP/3/QUIC başlatılmaz. Caddy'nin bazı native startup/shutdown logları config encoder'ından geçmez. Küçük `cmd/proxy` süreci Caddy stdout/stderr JSON'unu aynı event/alanları koruyarak service=edge ve null run_id/node_id/attempt ile tamamlar; JSON dışı bir process mesajını JSON'a çevirir. SIGTERM child Caddy'ye iletilir. Base image hâlâ pinli Caddy digest'idir; yeni auth/provider bağımlılığı eklenmez. Startup/shutdown ve access dahil bütün proxy JSON satırları production testinde kontrol edilir.

Şifre sunucuda bcrypt hash olarak tutulur; SSH parolasıyla aynı olmamalıdır. Oturum 12 saatlik HMAC imzalı `__Host-flowkit_session` cookie'sidir: Secure, HttpOnly, SameSite=Strict, Path=/, Domain yok. Kalıcı session key sayesinde gate restart oturumu bozmaz; şifre hash'ini veya session key'i değiştirmek bütün eski oturumları geçersiz kılar. Çıkış tarayıcı cookie'sini siler; kopyalanmış bir token TTL bitene/secret dönene kadar geçerlidir. Login/logout aynı HTTPS origin'den gelir; tarayıcı mutation'larında Origin/Fetch Metadata kontrol edilir. Login IP başına dakikada en fazla beş denemedir; limiter bounded ve gate restart'ta sıfırlanır. Caddy, client IP header'ını kendisi yazar; gate yalnız private Docker ağındadır. Proxy Nginx/API'ye Authorization veya Cookie aktarmaz; yanıtlar no-store'dur. Hesap/yetki sistemi eklenmez.

## DNS ve erişim

Cloudflare DNS ekranında yeni bir **A** kaydı ekleyin: seçtiğiniz alt alan (örneğin `flowkit`), VPS public IPv4 adresi, TTL Auto, başlangıçta **DNS only**. Domainin nameserver'larının Cloudflare'a yönlenmiş olması gerekir. Mevcut apex/mail/diğer kayıtları koruyun; otomatik DNS taramasını mevcut sağlayıcının kayıtlarıyla karşılaştırın. HTTP 80 sertifika doğrulaması/HTTPS yönlendirmesi, HTTPS 443 uygulama için açık olmalıdır. SSH portunu erişim için koruyun. Docker portları host firewall kurallarından bağımsız etkilenebilir; yalnızca proxy 80/443 yayınlar, DB 5432 ve kontrol araçları yayınlanmaz.

Cloudflare proxy daha sonra istenirse açılabilir; origin sertifikası doğrulandıktan sonra SSL/TLS **Full (strict)** seçin. İlk kurulum DNS only ile origin HTTPS'ini doğrudan doğrular. Proxy kullanılsa da origin'deki parola koruması devam eder.

## Sunucu hazırlığı

Docker Engine ve Compose plugin'i [resmî Ubuntu kurulumuyla](https://docs.docker.com/engine/install/ubuntu/) kurun. `git`, `curl`, Docker ve `docker compose` çalışmalıdır. Repo public olduğu için GitHub token gerekmez:

```sh
git clone https://github.com/CanhK-2103/trypix-flowkitCase.git /opt/flowkit
cd /opt/flowkit
cp .env.example .env
chmod 600 .env
```

`.env` içine `FLOWKIT_DOMAIN`, güçlü/random **hex** `POSTGRES_PASSWORD`, `DASHBOARD_PASSWORD_HASH` ve bağımsız **32-byte hex** `DASHBOARD_SESSION_KEY` yazın. DB parolası URL'ye gömüldüğü için hex olmalıdır; DB parolası ve session key'i ayrı ayrı `openssl rand -hex 32` ile üretin. Site şifresini (en fazla 72 UTF-8 byte) etkileşimli hashleyin:

```sh
docker run --rm -it caddy:2.11.7-alpine caddy hash-password
```

Hash'i `.env` içinde tek tırnakla saklayın: `DASHBOARD_PASSWORD_HASH='$2a$...'`. Plaintext dashboard parolasını repoya veya Compose tanımına yazmayın. Provider anahtarları ilk kurulumda **boş** tutulur. Local `.env` dosyasını bütünüyle sunucuya kopyalamayın; SSH parolası/yerel kontrol bilgileri sunucunun uygulama ortamına taşınmaz.

## Güvenli ilk başlangıç

DB ve dashboard image'ları ana Dockerfile'ın `database`/`dashboard` target'larından build edilir. Resmî entrypoint'leri `cmd/logwrap` çalıştırır; PostgreSQL motor ve Nginx startup/native error/access çıktıları dahil her satır JSON ve ortak null run/node/attempt bağlamı taşır. Stop sinyali child'a iletilir, mevcut PostgreSQL/artifact volume'ları korunur.

Faturalama okuma yetkisi olmayan kurulumda `.env` içine `BILLING_DISABLED=true` yazılır. API `/capabilities` ile bu durumu bildirir; otomatik billing ve refresh planlama kapalıdır. Üretim/quote/OpenRouter usage maliyetleri korunur; bilinmeyen fal gerçek tutarı uydurulmaz.

```sh
docker compose -p flowkit -f compose.yaml -f compose.production.yaml up -d --build --wait
docker compose -p flowkit -f compose.yaml -f compose.production.yaml ps
```

Production override worker'a `generation` profili verir; bu komut worker başlatmaz. Model kataloğu, örnek iki workflow, JSON editörü, doğrulama, geçmiş ve katalog tabanlı maliyet tahmini açılır. Provider anahtarları boşken üretim isteği anlamlı `PROVIDER_NOT_CONFIGURED` hatası verir. API ve proxy startup üretim/fiyat/billing çağrısı yapmaz.

**Anahtarın varlığı çağrı izni değildir.** Bu oturumda her live credential use ayrı onay gerektirir. Normal worker çalışırsa kuyruk, retry, polling ve bekleyen billing sorgularını otomatik yürütür. Onay kapsamı alınmadan production worker'ı başlatmayın. Çağrı başına kontrollü demo gerekiyorsa [onaylı worker aracını](../docs/APPROVED_DEMO.md) kullanın; kontrol portu yalnız SSH tüneli/loopback üzerinden erişilebilir olmalı, internete yayınlanmamalıdır.

Kullanıcı normal worker'ın yapacağı çağrıların kapsamını ayrıca onayladığında, provider anahtarlarını sunucunun ignored `.env` dosyasına ekleyip API konfigürasyonunu yenileyin ve worker'ı **açıkça** başlatın:

```sh
docker compose -p flowkit -f compose.yaml -f compose.production.yaml up -d api
docker compose -p flowkit -f compose.yaml -f compose.production.yaml up -d worker
```

Bu komutlar .env içine anahtar koymayı tek başına izin saymaz. Uygulamanın normal worker davranışı değiştirilmez; production profilinin amacı hazırlık sırasında fark edilmeden çalıştırmayı önlemektir. Worker kapalıyken mevcut anahtarlarla run kabul edilmişse işlem queued kalır.

## Kabul ve bakım

```sh
# Üretim başlatmaz. API parolasız 401 döner; dashboard kökü login formuna 303 döner.
curl -s -o /dev/null -w '%{http_code}\n' https://YOUR_SUBDOMAIN/models
curl -s -D - -o /dev/null https://YOUR_SUBDOMAIN/
docker compose -p flowkit -f compose.yaml -f compose.production.yaml logs --tail=50 api gate proxy
```

Tarayıcı `/auth/login` formunda yalnız şifre ister; başarılı giriş `/` adresine döner. `/auth/session` yalnız korumanın aktif/oturumun geçerli olup olmadığını söyler. `POST /auth/logout` cookie'yi siler; dashboard'da Çıkış butonu vardır. API curl erişimi de aynı oturum cookie'sini kullanır:

```sh
umask 077
read -r -s -p 'FlowKit password: ' FLOWKIT_PASSWORD; echo
printf '%s' "$FLOWKIT_PASSWORD" | curl -s -c .env.cookies \
  -H 'Origin: https://YOUR_SUBDOMAIN' \
  --data-urlencode 'password@-' https://YOUR_SUBDOMAIN/auth/login -o /dev/null
unset FLOWKIT_PASSWORD
curl -b .env.cookies https://YOUR_SUBDOMAIN/health/ready
```

Şifre stdin üzerinden gider, argv'de bulunmaz; `.env.cookies` ignored bir dosyadır. API/asset yolları oturumsuz 401, dashboard kökü GET ile login formuna 303 döner; HTTP HTTPS'e yönlenir. DB/API/dashboard/gate host portu olmamalıdır. Sertifika tarayıcı tarafından güvenilir olmalıdır. Uygulama/provider anahtarları frontend veya loglarda görünmemelidir. Yerel production smoke kontrolü gerçek sağlayıcı çağırmaz:

```sh
node tools/check-production.mjs
```

PostgreSQL ve görseller `database`/`artifacts` volume'larında, TLS verisi `caddy_data`/`caddy_config` volume'larında kalır. `docker compose down --volumes` production verisini siler; güncelleme için kullanılmaz. Kod güncellemesi: `git pull --ff-only`, ardından yukarıdaki ilk başlangıç komutu. Worker daha önce açıkça başlatılmışsa profil onu otomatik durdurmaz; onay gerektiren operasyondan önce `docker compose ... stop worker` kullanın.

DB yedeğini `umask 077` ile `docker compose ... exec -T db pg_dump -U flowkit -Fc flowkit > backup.dump` olarak alın; tamamlanmış görsellerin artifacts volume'unu da yedekleyin ve sunucu dışında saklayın. Çalışan PostgreSQL data dosyalarını sıradan dosya kopyasıyla yedeklemek yerine pg_dump kullanın. Tam bir restore provası yapılmadan yedekleme doğrulanmış sayılmaz. Kamatera trial bitişi ve ücret koşulları kullanıcı hesabından takip edilir; hosting kapanmadan yedekler alınır.

## Durum

Production override yerelde ayrı, anahtarsız Compose ortamında test edildi: tek şifreli login, API/asset koruması, normal CA doğrulamasıyla HTTPS, Origin/CSRF reddi, logout, kapalı backend portları, varsayılan kapalı worker, açıkça başlatılan sağlayıcısız worker run'ı/idempotency, gate/proxy restart sonrası aynı oturum ve JSON logların null bağlamı. Test kendi geçici container/volume'larını kaldırdı. Go/PostgreSQL/vet/sqlc/frontend kontrolleri de başarılı.

Sunucu SSH ön kontrolü Ubuntu 24.04/x86_64, 1 CPU, yaklaşık 4 GB RAM ve 50 GB disk gösterdi; Docker Engine 29.9.0 ve Compose 5.6.0 resmî apt deposundan kuruldu. Cloud kontrolü sağlayıcısız yapıldı; yeni fal/OpenRouter run'ları için ayrı kullanıcı onayı gerekir.

8852cd3 sürümü VPS üzerinde build edilip production override ile başlatıldı; komut exit 0. API `ready`, iki örnek workflow mevcut, worker kapalı, provider anahtarları boş ve backend host portları kapalı olarak kontrol edildi. Aynı sürümün [CI'sı](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38165184856) production testi dahil başarılı.

**Public yönetim paneli:** https://flowkit.canhurkurtoglu.com. Cloudflare journey/wesley nameserver'ları aktif; A kaydı verilen VPS adresine yönlenir, DNS only. DNS hazır olmadan başlayan ACME denemeleri NXDOMAIN/backoff aldı; DNS'in iki public resolver ve VPS üzerinde doğru olduğu doğrulanınca yalnız proxy bir kez yeniden başlatıldı. Ardından Let's Encrypt YE2 issuer'lı public sertifika sistem trust store + hostname/SAN ile doğrulandı; sertifika validasyonunu kapatma veya uyarı bypass yapılmadı. Normal DNS üzerinden HTTPS sorgusu da 401 döndü.

İlk public kontrol Basic Auth ile HTTP 308, yetkisiz 401/yetkili 200 ve güvenilir TLS'i doğruladı. Kullanıcının sonraki isteğiyle tek şifreli giriş ve ayrı site şifresi eklendi; 268d72d build/startup exit 0, gate/API healthy. [Bu sürümün CI'sı](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38167811202) bütün job'ları geçti. Güncel [anahtarsız deployment kanıtı](../docs/evidence/deployment.json) password-only form 200, root login 303, yetkisiz API/asset ve yanlış şifre 401, doğru şifreyle 200, CSRF 403 ve logout/sonrası 401 sonuçlarını gösterir. Üç model ve iki workflow hazır. Kanıt parola/hash/session cookie içermez; [boş login ekranı](../docs/flowkit-login.jpg) yayından alındı. Worker kapalı/provider anahtarları boş olduğu için cloud üretim aktif değildir; yerel gerçek run/medya cloud'a taşınmamıştır. Değerlendiriciye URL ile yalnız site şifresi ayrıca verilir; SSH erişim bilgisi verilmez.

11 Ekim'deki proxy log düzeltmesi 3bf8667 olarak VPS'e build/recreate edildi; [o sürümün CI'sı](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38168698336) başarılıdır. Sonraki ef9bbdb düzeltmesi gerçek tarayıcı login formundaki Origin=null davranışını same-origin Referrer-Policy ile giderdi; strict origin koruması korundu. Bu düzeltmenin [CI'sı](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38170131348) ve 71208e6 takip kaydının [CI'sı](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38170369875) başarılıdır. Bunlar tarihsel kurulum kayıtlarıdır; sonraki tamamlamalar aşağıda kaydedilir.
