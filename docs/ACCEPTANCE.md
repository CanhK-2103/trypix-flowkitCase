# FlowKit kabul kontrol listesi

Teslim tarihi: 12 Ekim 2026. Bu dosya case gereksinimlerinin takip listesidir; işaretlenmemiş maddeler tamamlandı kabul edilmez.

## Gerçek sağlayıcı denemeleri

- [ ] Doğrudan OpenRouter -> fal.ai akışı başarıyla çalıştırıldı; model, prompt, parametre ve sonuç kaydı var.
- [ ] Aynı akış FlowKit'e taşındı; README eşleştirme tablosu gerçek denemeyle örtüşüyor.
- [ ] İkinci workflow yalnızca API/dashboard ile eklendi; farklı bir görsel modeli kullandı.
- [ ] Yeni uyumlu model API ile eklendi, workflow modelRef değiştirildi, yeni run tamamlandı; kod/deploy/restart yok.

## API ve veri

- [ ] Model kataloğu: alias ile kayıt, listeleme, devre dışı bırakma ve temel CRUD.
- [ ] Workflow: oluşturma, listeleme, detay, güncelleme, silme, ayrı doğrulama endpoint'i.
- [ ] JSON node alanları: id, kind, modelRef, params, prompt şablonu; edge alanları: from, fromPort, to, toPort.
- [ ] input, llm, image, output türleri; text, prompt, image portları.
- [ ] Döngü, dallanma/birleşme, bağlantısız node/zorunlu port, uyumsuz port, bilinmeyen model ve çözülemeyen şablon değişkeni reddediliyor.
- [ ] Oluşturma VE güncelleme bütün doğrulama bulgularını döndürüyor.
- [ ] POST /workflows/{id}/runs işi bitirmeden run_id dönüyor; durum ve detay endpoint'leri var.

## Motor ve görünürlük

- [ ] Ayrı worker adımları sırayla yürütüyor; önceki çıktı sonraki node tarafından kullanılabiliyor.
- [ ] fal queue submit -> status -> result; provider_request_id submit sonrası hemen kalıcı kaydediliyor.
- [ ] Geçici/kalıcı hata ayrımı; node başına toplam en fazla 3 deneme, artan bekleme.
- [ ] Başarısız node ve hata nedeni kalıcı; adım bazında girdi/çıktı, kullanılan model, süre, deneme izi var.
- [ ] JSON loglar: run_id, node_id, attempt; provider hata kodu ve mesajı error seviyesinde.
- [ ] Dashboard: workflow listesi, JSON editörü, doğrula/kaydet, girdi formu, run geçmişi/detayı.
- [ ] Her üretilen görselin yanında gerçek model gösteriliyor.

## Test ve teslim

- [ ] En az 3 doğrulama kuralını reddeden testler.
- [ ] Mock sağlayıcıyla başarılı run, geçici hata + retry + başarı, kalıcı hata + doğru node/kod testleri.
- [ ] Workflow ve katalog CRUD testleri; testlerde gerçek sağlayıcı çağrısı yok.
- [ ] docker compose up API, worker, PostgreSQL ve dashboard'u başlatıyor.
- [ ] .env.example, endpoint örnekleri, graf şeması, hata/retry politikası README'de.
- [ ] Gerçek iki run ve yeni model senaryosu README'de kanıtlanmış.
- [ ] AGENTS.md ve AI kullanım açıklaması; bir hafta daha olsaydı paragrafı.
- [ ] GitHub repository erişimi hazır.

## Bonus (zorunlu değil)

- [ ] Worker restart sonrası aynı fal isteğini tekrar göndermeden devam etme kanıtı.
- [ ] İstemci idempotency key.
- [ ] Tahmini maliyet izi.
