# VPS deployment

Mevcut Go API, River worker, PostgreSQL ve React dashboard aynı Compose tanımıyla tek Ubuntu 24.04 VPS üzerinde çalışır. Production override yalnızca giriş noktasını, DB parolasını ve worker'ın varsayılan başlangıcını değiştirir. Caddy 2.11.7 HTTPS sertifikasını otomatik alıp yeniler; dashboard, API, health ve görseller aynı Basic Auth korumasındadır. PostgreSQL, API ve dashboard için host portu yayınlanmaz. Caddy admin API kapalıdır.

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

`.env` içine `FLOWKIT_DOMAIN`, `DASHBOARD_USERNAME`, güçlü/random **hex** `POSTGRES_PASSWORD` ve `DASHBOARD_PASSWORD_HASH` yazın. PostgreSQL URL'sine gömüldüğü için DB parolasında yalnız hex karakter kullanın. Örnek üretim: `openssl rand -hex 32`. Gateway parolasını etkileşimli hashleyin:

```sh
docker run --rm -it caddy:2.11.7-alpine caddy hash-password
```

Hash'i `.env` içinde tek tırnakla saklayın: `DASHBOARD_PASSWORD_HASH='$2a$...'`. Plaintext dashboard parolasını repoya veya Compose tanımına yazmayın. Provider anahtarları ilk kurulumda **boş** tutulur. Local `.env` dosyasını bütünüyle sunucuya kopyalamayın; SSH parolası/yerel kontrol bilgileri sunucunun uygulama ortamına taşınmaz.

## Güvenli ilk başlangıç

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
# Üretim başlatmaz. İlk istek 401, ikinci istek kullanıcı parolasını sorarak 200 dönmeli.
curl -s -o /dev/null -w '%{http_code}\n' https://YOUR_SUBDOMAIN/models
curl -u reviewer https://YOUR_SUBDOMAIN/health/ready
docker compose -p flowkit -f compose.yaml -f compose.production.yaml logs --tail=50 api proxy
```

Dashboard ve tüm API yolları parolasız 401 dönmelidir; HTTP HTTPS'e yönlenmelidir. Açık DB/API/dashboard host portu olmamalıdır. Sertifika tarayıcı tarafından güvenilir olmalıdır. Uygulama/provider anahtarları frontend veya loglarda görünmemelidir. Yerel production smoke kontrolü gerçek sağlayıcı çağırmaz:

```sh
node tools/check-production.mjs
```

PostgreSQL ve görseller `database`/`artifacts` volume'larında, TLS verisi `caddy_data`/`caddy_config` volume'larında kalır. `docker compose down --volumes` production verisini siler; güncelleme için kullanılmaz. Kod güncellemesi: `git pull --ff-only`, ardından yukarıdaki ilk başlangıç komutu. Worker daha önce açıkça başlatılmışsa profil onu otomatik durdurmaz; onay gerektiren operasyondan önce `docker compose ... stop worker` kullanın.

DB yedeğini `umask 077` ile `docker compose ... exec -T db pg_dump -U flowkit -Fc flowkit > backup.dump` olarak alın; tamamlanmış görsellerin artifacts volume'unu da yedekleyin ve sunucu dışında saklayın. Çalışan PostgreSQL data dosyalarını sıradan dosya kopyasıyla yedeklemek yerine pg_dump kullanın. Tam bir restore provası yapılmadan yedekleme doğrulanmış sayılmaz. Kamatera trial bitişi ve ücret koşulları kullanıcı hesabından takip edilir; hosting kapanmadan yedekler alınır.

## Durum

Production override yerelde ayrı, anahtarsız Compose ortamında test edildi: tüm yolların parola koruması, normal sertifika doğrulamasıyla yerel HTTPS, yönlendirme, kapalı backend portları, varsayılan kapalı worker, açıkça başlatılan sağlayıcısız worker run'ı ve idempotency, proxy restart ve 26 JSON access log satırında null bağlam. Test kendi geçici container/volume'larını kaldırdı. Go/PostgreSQL/vet/sqlc/frontend kontrolleri de başarılı.

Sunucu SSH ön kontrolü Ubuntu 24.04/x86_64, 1 CPU, yaklaşık 4 GB RAM ve 50 GB disk gösterdi; Docker Engine 29.9.0 ve Compose 5.6.0 resmî apt deposundan kuruldu. Cloud kontrolü sağlayıcısız yapıldı; yeni fal/OpenRouter run'ları için ayrı kullanıcı onayı gerekir.

8852cd3 sürümü VPS üzerinde build edilip production override ile başlatıldı; komut exit 0. API `ready`, iki örnek workflow mevcut, worker kapalı, provider anahtarları boş ve backend host portları kapalı olarak kontrol edildi. Aynı sürümün [CI'sı](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38165184856) production testi dahil başarılı.

**Public yönetim paneli:** https://flowkit.canhurkurtoglu.com. Cloudflare journey/wesley nameserver'ları aktif; A kaydı verilen VPS adresine yönlenir, DNS only. DNS hazır olmadan başlayan ACME denemeleri NXDOMAIN/backoff aldı; DNS'in iki public resolver ve VPS üzerinde doğru olduğu doğrulanınca yalnız proxy bir kez yeniden başlatıldı. Ardından Let's Encrypt YE2 issuer'lı public sertifika sistem trust store + hostname/SAN ile doğrulandı; sertifika validasyonunu kapatma veya uyarı bypass yapılmadı. Normal DNS üzerinden HTTPS sorgusu da 401 döndü.

Public kontrol: HTTP 308; dashboard/models/workflows/runs/health/artifacts ve static asset parolasız 401; yanlış parola ve yetkisiz workflow POST 401; doğru parola ile readiness/dashboard/API/JS asset 200. Üç model ve iki workflow hazır. [Anahtarsız deployment kanıtı](../docs/evidence/deployment.json) sertifika ve HTTP sonuçlarını içerir; kullanıcı parolası veya auth header içermez. Gerçek sağlayıcı çağrısı 0. Worker kapalı/provider anahtarları boş olduğu için cloud üretim aktif değildir; yerel gerçek run/medya cloud'a taşınmamıştır. Değerlendiriciye URL ile gateway erişim bilgisi ayrıca verilir; giriş bilgileri README/Git'e konmaz.
