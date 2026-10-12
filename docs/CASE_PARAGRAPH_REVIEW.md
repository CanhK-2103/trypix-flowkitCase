# Case metniyle paragraf paragraf eşleştirme

Hazırlanma tarihi: **12 Ekim 2026**. Kaynak: kullanıcının `C:\Users\PC\Downloads\case.txt` dosyası. Bu rapor uygulama davranışını değiştirmez.

**Orijinal metin** alanları dosyadan doğrudan alınmıştır: kelimeler, noktalama, sıra, satır kırımları ve boş satırlar korunur. Ekrandaki kod blokları yalnız gösterim biçimidir. Raporun metin satır sonu LF; [orijinal dosyanın byte olarak aynı kopyası](CASE_SOURCE.txt) CRLF dahil korunmuştur. Kaynak SHA-256: `0135af2fa51eb2b83ddf96167442c58c42d0ee98b6d91247526a1bb24973e1e1`. [Bütünlük manifesti](evidence/case-paragraph-review.json) her bölümün kaynak satırlarını kaydeder.

Her alıntının altında **Biz ne yaptık**, **Yapıldı mı?**, **Sen nasıl inceleyebilirsin?** ve **Değerlendirici nasıl inceleyebilir?** bulunur. Dosyada ayrık paragraf boşlukları her yerde olmadığı için cümle/konu sınırları kullanıldı; liste maddeleri ayrı inceleme konusu olarak ele alındı. Kaynağın başlıkları, tekrarlanan şartları ve boş satırları atlanmadı.

**Kanıtların anlamı:** “mock/PostgreSQL testli” ücretli sağlayıcı çağrısı değildir. “Gerçek sağlayıcı” 11 Ekim'deki açık onaylı doğrudan deney ve üç FlowKit run'ıdır. Bu üç run ve görselleri cloud'a aynen taşındı; yeni ücretli cloud üretimi olarak sunulmaz. 12 Ekim cloud kontrolü sağlayıcısız input→output run'ı, login/TLS/API/media ve servis/log kontrolleridir. En son uygulama düzeltmesi `14cf20a`, rapor öncesi repository/server dokümantasyonu `0d9ae48`; kanıt snapshot'larının eski kabul sürümünü göstermesi normaldir.

**Açık sınırlar:** Case'in tahmini maliyet bonusu tamamlandı. Sonradan istediğin fal gerçek ücret/gerçek toplam canlı doğrulaması, anahtarın yetkisi olmadığı ve sen kapsam dışında bıraktığın için yapılmadı; `BILLING_DISABLED=true`, eksik gerçek toplam `null`. Kod hâkimiyetinin kişisel olarak gösterilmesi ve site erişim şifresinin değerlendiriciye özel kanalla iletilmesi insan tarafından tamamlanacak teslim adımlarıdır. Secret değerleri rapora alınmadı.

İnceleme sırasında üretim/billing başlatmadan mevcut run'ları, kanıtları ve mock testleri kullanabilirsin. E1–E5 ortak kontrol yolları raporun sonundadır.

## 01. Amaç: veriyle tanımlanan asenkron backend

**Orijinal metin — kaynak satır 1–4:**

<!-- case-source: start=1 end=4 -->
```text
Amaç
Bu çalışma, AI üretim akışlarını (workflow) kod yerine veri olarak tanımlayan ve asenkron çalıştıran küçük bir
backend platformunun (FlowKit) geliştirilmesini hedefler. Değerlendirmenin odağı backend'dir; üretilen
görsellerin kalitesi değerlendirilmez.
```

**Biz ne yaptık:** Go HTTP API, ayrı River worker ve PostgreSQL kuruldu. Workflow tanımları JSONB verisi; motor bu tanımları yorumluyor. Görsel kalitesine yönelik ek üretim yapılmadı.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [yayındaki dashboard](https://flowkit.canhurkurtoglu.com/) üzerinde bir workflow'un JSON tanımını ve geçmiş run'ını aç; adımlar ve sonuç birbirine karşılık gelmeli.

**Değerlendirici nasıl inceleyebilir?** [README](../README.md) mimari bölümünü, [çalıştırma motoru](../internal/engine/engine.go), [depolama kodunu](../internal/storage/store.go) ve [Compose tanımını](../compose.yaml) karşılaştır. Workflow'a özel yürütme kodu bulunmamalı.

## 02. Tanımsız alanlarda mühendislik kararları

**Orijinal metin — kaynak satır 5–5:**

<!-- case-source: start=5 end=5 -->
```text
Açıkça tarif edilmeyen alanlar adayın kendi mühendislik yaklaşımıyla tamamlanmalıdır.
```

**Biz ne yaptık:** Framework, veri tabanı, kuyruk, graf sözleşmesi, retry, polling ve görsel saklama tercihleri belirlendi; gerekçeleri ve sınırları README/PLAN'da yazıldı.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [README](../README.md) içinde mimari, hata/retry, maliyet ve son paragrafı oku.

**Değerlendirici nasıl inceleyebilir?** [PLAN](../PLAN.md) ile uygulamanın transaction/checkpoint/model profile kararlarını karşılaştır; adaydan bu tercihleri gerekçelendirmesini iste.

## 03. API ile tanımlanabilen workflow sistemi

**Orijinal metin — kaynak satır 6–8:**

<!-- case-source: start=6 end=8 -->
```text
Görev Tanımı
Sizden, AI üretim akışlarını kod yazmadan tanımlayıp çalıştırabildiğimiz küçük bir backend sistemi kurmanızı
istiyoruz.
```

**Biz ne yaptık:** Workflow oluşturma, JSON düzenleme, doğrulama ve run başlatma HTTP API üzerinden yapılabiliyor; dashboard aynı API'yi kullanıyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [yayındaki dashboard](https://flowkit.canhurkurtoglu.com/) içinde JSON tanımı/editörünü ve Doğrula/Kaydet kontrollerini incele; yalnız doğrulama kullanmak üretim başlatmaz.

**Değerlendirici nasıl inceleyebilir?** [HTTP API kodu](../internal/api/server.go) ve [OpenAPI](openapi.json) ile [dashboard API istemcisini](../web/src/api.ts) ve [ana ekranı](../web/src/main.tsx) karşılaştır.

## 04. Önce doğrudan OpenRouter → fal denemesi

**Orijinal metin — kaynak satır 9–12:**

<!-- case-source: start=9 end=12 -->
```text
İşe basit bir akışı doğrudan API çağrılarıyla deneyerek başlayın: önce bir LLM (OpenRouter), kullanıcının
girdiği kısa bir metinden (ör. bir ürün açıklaması) görsel üretim prompt'u yazsın; ardından bir görsel modeli
(fal.ai) bu prompt'la bir görsel üretsin. Bu iki adımı çalıştırıp prompt yapısını ve modellerin nasıl çağrıldığını
anladıktan sonra, aynı akışı kendi geliştireceğiniz sisteme taşıyacaksınız.
```

**Biz ne yaptık:** Gerçek OpenRouter prompt'u ve fal queue görseli ayrı, onaylı doğrudan deneyde üretildi; ardından FlowKit'in ilk gerçek run'ı aynı sözleşmeyle çalıştı. Anahtarlar daha sonra sağlandığı için bağımsız kod/mock geliştirmesi önce ilerledi; doğrudan gerçek deney ilk gerçek FlowKit kabul run'ından önce yapıldı.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [doğrudan deneme kanıtı](evidence/direct.json) içinde model, prompt, parametre, request ID, status ve görsel sonucunu oku; [Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a) ile karşılaştır.

**Değerlendirici nasıl inceleyebilir?** [Doğrudan deney kodunu](../cmd/direct/main.go), [doğrudan deneme kanıtı](evidence/direct.json) zamanlarını ve README eşleştirme tablosunu incele. Kod hazırlama ile canlı deney sırasını karıştırma; bu kaydı okumak yeni ücretli çağrı gerektirmez.

## 05. JSON tanımının DB'de saklanması ve arka planda yürütme

**Orijinal metin — kaynak satır 13–15:**

<!-- case-source: start=13 end=15 -->
```text
Geliştireceğiniz sistemde bu akış kodun içine yazılmayacak; adımları, kullanılan modelleri ve prompt'ları
anlatan bir JSON tanımı olarak veritabanında duracak. Sistem bu tanımı kaydederken doğrulayacak ve bir
kullanıcı akışı çalıştırdığında adımları arka planda sırayla fal.ai ve OpenRouter'a istek göndererek yürütecek.
```

**Biz ne yaptık:** Node/edge/modelRef/params/promptTemplate DB'de tutuluyor. Create/update doğrulamadan geçiyor; worker kaydedilmiş workflow/model snapshot'ından sıralı çalışıyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** Giriş sonrası /workflows endpoint'ini aç; definition.nodes/edges alanlarını JSON editörüyle karşılaştır. Run detayında workflow/model snapshot'larını incele.

**Değerlendirici nasıl inceleyebilir?** [HTTP API kodu](../internal/api/server.go) ve [OpenAPI](openapi.json), [SQL sorguları](../db/queries/flowkit.sql), [migration](../db/migrations/00001_init.sql) ve [çalıştırma motoru](../internal/engine/engine.go) üzerinde doğrulama → kayıt → job → yürütme zincirini izle.

## 06. Hata, retry ve açıklanabilir çalıştırma izi

**Orijinal metin — kaynak satır 16–18:**

<!-- case-source: start=16 end=18 -->
```text
Bir adım hata verirse sistem bunu anlamlı şekilde ele alacak (gerekiyorsa yeniden deneyecek) ve sonunda her
çalıştırmada hangi adımda ne olduğunu, hangi modelin kullanıldığını ve bir hata olduysa nedenini
gösterebilecek.
```

**Biz ne yaptık:** Hatalar sınıflandırılıyor; güvenli geçici hatalarda node retry uygulanıyor. Run/node/attempt/provider request kayıtları model, faz, süre ve hata nedenini kalıcı tutuyor.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Dashboard run detayında node'ları aç; model, deneme, süre ve girdi/çıktıyı incele. Hata yolları için ücretli üretim yerine test çıktısını kullan.

**Değerlendirici nasıl inceleyebilir?** [Motor testlerinde](../internal/engine/engine_test.go) TestTransientRetryThenSuccess, TestRetryCap ve TestPermanentFailureCorrectNode; integration testlerinde success/retry/permanent alt testlerini incele.

## 07. İki farklı workflow ve çıktı modeli

**Orijinal metin — kaynak satır 19–20:**

<!-- case-source: start=19 end=20 -->
```text
Sistemde en az iki farklı akış olmalı ve bu akışlar farklı görsel modelleri kullanmalıdır. Her üretimde, çıktıyı
hangi modelin ürettiği ekranda görünmelidir.
```

**Biz ne yaptık:** Ürün katalog görseli Schnell, etkinlik afişi Dev kullanıyor. Dashboard çıktının kaydedilmiş gerçek üretici modelini gösteriyor. İki workflow ve üç önceki gerçek run/görsel cloud'a taşındı.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [yayındaki dashboard](https://flowkit.canhurkurtoglu.com/) sol listesinde ürün ve etkinliği aç; görsel altındaki model etiketlerini karşılaştır. [Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a) ile [etkinlik Dev run](https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321) farklı fal modelleri göstermeli.

**Değerlendirici nasıl inceleyebilir?** [ürün JSON'u](../examples/workflows/product.json), [etkinlik JSON'u](../examples/workflows/event.json), [gerçek API/run kanıtı](evidence/flowkit-approved.json) ve [sunucu tamamlama kanıtı](evidence/deployment-fixes.json) içindeki provenance/artifact alanlarını karşılaştır. Güncel workflow modeliyle eski run modelinin ayrı tutulduğunu doğrula.

## 08. Kod/restart olmadan yeni model ve workflow

**Orijinal metin — kaynak satır 21–23:**

<!-- case-source: start=21 end=23 -->
```text
Sistemin asıl sınavı esnekliğidir: kurduğunuz yapıya ikinci, farklı bir akışı ve aynı sağlayıcıdan yeni bir
modeli kod değiştirmeden, servisi yeniden başlatmadan, yalnızca API veya dashboard üzerinden ekleyebiliyor
olmalısınız.
```

**Biz ne yaptık:** Dev API ile kataloğa eklendi; mevcut ürün workflow'u Dev'e geçirildi ve başarılı çalıştı. Ürün Schnell'e geri alındı; ikinci etkinlik workflow'u sadece API verisiyle eklendi ve Dev ile tamamlandı. Kabul aralığında Git/container kimlikleri aynı kaldı.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [gerçek API/run kanıtı](evidence/flowkit-approved.json) içinde runtime_before/runtime_after_all ve HTTP mutasyonlarını aç; [model değişimi Dev run](https://flowkit.canhurkurtoglu.com/runs/217fb931-3af8-4313-89b2-436985798449) ile [etkinlik Dev run](https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321) sonuçlarına bak.

**Değerlendirici nasıl inceleyebilir?** [PostgreSQL/worker testleri](../internal/integration/integration_test.go) içindeki TestHotAddModelAndWorkflowWithoutRestart ile gerçek kanıttaki Git SHA, container ID/StartedAt/image değerlerini karşılaştır. Zaten kayıtlı Dev'i tekrar POST etmek yeni-model kabulünün yerine geçmez.

## 09. Dashboard, hata testleri, Compose ve README

**Orijinal metin — kaynak satır 24–26:**

<!-- case-source: start=24 end=26 -->
```text
Bunlara ek olarak sistemi yönetmek için basit bir dashboard, hata senaryolarını da kapsayan testler, tek
komutla ( docker compose up ) ayağa kalkan bir kurulum ve kararlarınızı anlatan bir README teslim etmeniz
bekleniyor. Görsellerin kalitesi değil, sistemin doğru, dayanıklı ve genişletilebilir çalışması değerlendirilecektir.
```

**Biz ne yaptık:** React dashboard, otomatik mock/integration testleri, tek komutluk standart Compose kurulumu ve README teslim edildi. CI backend/dashboard/compose kontrollerini çalıştırıyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [yayındaki dashboard](https://flowkit.canhurkurtoglu.com/), [README](../README.md) ve [başarılı CI](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38243986198) bağlantılarını aç; CI'nin üç işi de başarılı olmalı.

**Değerlendirici nasıl inceleyebilir?** [CI dosyasını](../.github/workflows/ci.yml), [standart Compose kontrolünü](../tools/check-compose.mjs) ve test dosyalarını incele. Temiz kurulum kontrolü gerçek AI anahtarı gerektirmez.

## 10. Ayrıntı bölümlerine geçiş

**Orijinal metin — kaynak satır 27–29:**

<!-- case-source: start=27 end=29 -->
```text
Aşağıdaki bölümler bu isteklerin ayrıntılarını içerir.


```

**Biz ne yaptık:** Belgenin aşağıdaki gereksinimleri ayrı ayrı eşleştirildi; bu cümle yeni bir özellik istemiyor.

**Yapıldı mı?** Bilgilendirme / kapsam açıklaması; ayrı bir geliştirme işi değil.

**Sen nasıl inceleyebilirsin?** Aşağıdaki numaralı eşleştirmeleri sırayla incele.

**Değerlendirici nasıl inceleyebilir?** Bölüm başlıklarının ve tekrar eden gereksinimlerin de raporda kaynak sırasıyla korunduğunu kontrol et.

## 11. Kapsama giren alanlar

**Orijinal metin — kaynak satır 30–42:**

<!-- case-source: start=30 end=42 -->
```text
Kapsam
Bu çalışma şu alanlara odaklanır:
Backend servis mimarisi ve API tasarımı
Workflow tanımı ve doğrulama
Asenkron çalıştırma (job/worker), hata ve retry
yönetimi
AI sağlayıcı entegrasyonu (fal.ai ve OpenRouter)
Gözlemlenebilirlik (yapılandırılmış log ve
çalıştırma izi)
Basit bir yönetim paneli (dashboard)
Çalıştırma ortamı (Docker Compose)


```

**Biz ne yaptık:** API/worker mimarisi, JSON graf doğrulama, iki sağlayıcı adaptörü, retry, JSON log/kalıcı iz, React dashboard ve Compose kuruldu.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [README](../README.md) mimari şeması, [yayındaki dashboard](https://flowkit.canhurkurtoglu.com/) ve [başarılı CI](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38243986198) üzerinden listedeki alanları kontrol et.

**Değerlendirici nasıl inceleyebilir?** internal/api, internal/workflow, internal/engine, internal/providers, internal/jobs, internal/telemetry ve web/src dizinlerini Compose servisleriyle eşleştir.

## 12. Kapsam dışındaki alanlar

**Orijinal metin — kaynak satır 43–54:**

<!-- case-source: start=43 end=54 -->
```text
Kapsam dışı:
Sürükle-bırak görsel canvas (JSON editörü
yeterlidir)
Kullanıcı girişi ve yetkilendirme
Kredi/ödeme sistemi
Ortamlar arası (staging → production) promotion
Tek sıra halinde ilerlemeyen akışlar (bir adımın
çıktısının birden fazla adıma gitmesi, adımların
paralel çalışması) ve bir workflow'un başka bir
workflow'u çağırması
Çıktı kalitesinin iyileştirilmesi

```

**Biz ne yaptık:** Canvas, kullanıcı hesapları/rolleri, ödeme, promotion, paralel node/fan-out ve subworkflow eklenmedi. JSON editörü kullanılıyor. Yayındaki tek şifreli erişim kapısı, belgenin deploy korumasını karşılıyor; uygulama içi kullanıcı sistemi değil.

**Yapıldı mı?** Kapsama uygun; listedeki özellikler zorunlu değil.

**Sen nasıl inceleyebilirsin?** Editörde JSON bulunduğunu ve yayındaki girişin yalnız ortak site şifresi istediğini gör.

**Değerlendirici nasıl inceleyebilir?** [doğrulama kodu](../internal/workflow/validate.go) ve [doğrulama testleri](../internal/workflow/validate_test.go) üzerinde doğrusal graf dışının reddini, [gate kodunda](../internal/gate/gate.go) yalnız deployment oturumunu kontrol et. Bağımsız run'ların eşzamanlılığı workflow içi paralellik değildir.

## 13. HTTP API işlemleri

**Orijinal metin — kaynak satır 55–58:**

<!-- case-source: start=55 end=58 -->
```text
Gereksinimler
API
Tüm işlemler bir HTTP API üzerinden yapılabilmelidir: model kataloğu; workflow oluşturma, listeleme,
güncelleme, silme ve doğrulama; run başlatma, durum ve detay sorgulama.
```

**Biz ne yaptık:** Katalog kayıt/liste/detay/güncelleme-disable; workflow CRUD/validate; run başlatma/list/status/detail endpoint'leri var. Dashboard işlemleri için ayrı gizli backend yolu yok.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Giriş sonrası /models, /workflows, /runs ve bir /runs/{id}/status adresini aç; JSON yanıtları incele.

**Değerlendirici nasıl inceleyebilir?** [HTTP API kodu](../internal/api/server.go) ve [OpenAPI](openapi.json) route listesiyle [PostgreSQL/worker testleri](../internal/integration/integration_test.go) içindeki TestCatalogWorkflowCRUDAndValidation ve run testlerini karşılaştır.

## 14. Dashboard API'yi kullanır

**Orijinal metin — kaynak satır 59–59:**

<!-- case-source: start=59 end=59 -->
```text
Dashboard bu API'yi kullanmalıdır; dashboard'da yapılabilen her işlem API ile de yapılabilmelidir.
```

**Biz ne yaptık:** React fetch tabanlı API istemcisi ve TanStack Query kullanıyor. Dashboard'un DB veya sağlayıcıya doğrudan erişimi yok.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** Tarayıcı geliştirici araçları → Network bölümünü aç; workflow seç/Doğrula/run detayı aç. /workflows ve /runs çağrılarını gör.

**Değerlendirici nasıl inceleyebilir?** [api.ts](../web/src/api.ts), [main.tsx](../web/src/main.tsx), [API istemci testlerini](../web/src/api.test.ts) ve [Nginx yönlendirmesini](../web/nginx.conf) incele.

## 15. Endpoint listesi ve örnek istekler

**Orijinal metin — kaynak satır 60–60:**

<!-- case-source: start=60 end=60 -->
```text
README'de endpoint listesi ve örnek istekler ( curl veya bir Postman/OpenAPI dosyası) bulunmalıdır.
```

**Biz ne yaptık:** README'de yöntem/endpoint tablosu ve cURL örnekleri; repoda üretilen OpenAPI JSON'u var.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [README](../README.md) API bölümünü ve [OpenAPI](openapi.json) dosyasını aç.

**Değerlendirici nasıl inceleyebilir?** OpenAPI'yi istemciye import ederek endpoint/şema incelemesi yap; CI'nin OpenAPI regeneration fark kontrolünü gör. Örnekteki run POST'unu ücretli kullanım onayı olmadan çalıştırma.

## 16. Model kayıtlarının API verisi olması

**Orijinal metin — kaynak satır 61–63:**

<!-- case-source: start=61 end=63 -->
```text
Model kataloğu
Modeller API üzerinden kaydedilebilmelidir (dashboard'da katalog ekranı gerekmez). Aynı sağlayıcıda yeni
bir model eklemek kod değişikliği gerektirmemelidir.
```

**Biz ne yaptık:** POST /models alias/provider/modelId/inputProfile/defaults/pricing kaydediyor. Desteklenen aynı profil içinde yeni model kimliği veri olarak çalışıyor; katalog ekranı zorunlu olmadığı için ayrıca yapılmadı.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Giriş sonrası /models üzerinde prompt-writer, flux-schnell ve flux-dev kayıtlarını gör.

**Değerlendirici nasıl inceleyebilir?** [Model örneklerini](../examples/models/flux-dev.json), createModel kodunu, TestCompatibleCatalogModels ve [gerçek API/run kanıtı](evidence/flowkit-approved.json) içindeki Dev POST yanıtını incele.

## 17. Dört adımlı Schnell → Dev kabul senaryosu

**Orijinal metin — kaynak satır 64–70:**

<!-- case-source: start=64 end=70 -->
```text
Bu, çalışmanın en önemli kabul koşullarından biridir ve teslimden önce sizin tarafınızdan denenmiş
olmalıdır. Örnek senaryo:
1. Workflow'unuz fal-ai/flux/schnell modeliyle çalışıyor.
2. Yalnızca API isteğiyle katalogdaki fal-ai/flux/dev modelini kaydedersiniz.
3. Workflow'da ilgili node'un model referansını API ile yeni modele çevirir ve workflow'u çalıştırırsınız.
4. Run yeni modelle tamamlanır. Bu süreçte kod değişmez, servis yeniden başlatılmaz, deploy
yapılmaz.
```

**Biz ne yaptık:** Senaryo gerçek sağlayıcılarla tamamlandı: ilk Schnell run; POST ile Dev kaydı; PUT ile render.modelRef=flux-dev/revision 2; başarılı Dev run. Kod/container kimlikleri kabul boyunca değişmedi. Sonradan yapılan bug fix/deployment bu kabul aralığından ayrı kaydedildi.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a) ve [model değişimi Dev run](https://flowkit.canhurkurtoglu.com/runs/217fb931-3af8-4313-89b2-436985798449) sonuçlarını, [gerçek API/run kanıtı](evidence/flowkit-approved.json) içindeki kayıt/güncelleme isteklerini sırayla oku.

**Değerlendirici nasıl inceleyebilir?** runtime_before ile runtime_after_all Git/container kimliklerini karşılaştır; Dev run snapshot'ı revision 2 ve fal-ai/flux/dev olmalı. Başlangıçtaki demo bekleme düzeltmesi bu kesintisiz kabul kanıtı olarak gösterilmiyor.

## 18. Uyumlu model sözleşmesinin sınırı

**Orijinal metin — kaynak satır 71–74:**

<!-- case-source: start=71 end=74 -->
```text
Bu kural, aynı parametreleri alan modeller içindir (ör. flux/schnell ile flux/dev ; ikisi de prompt ve
image_size alır). Sisteminizin henüz göndermediği bir parametreyi bekleyen bir model için (ör. girdi
olarak görsel listesi isteyen bir model) küçük bir kod değişikliği gerekmesi kabul edilebilir; bu sınırı
README'de açıklayın.
```

**Biz ne yaptık:** openrouter-chat-text-v1 ve fal-text-to-image-v1 giriş profilleri destekleniyor. Schnell/Dev prompt ve image_size ile aynı adaptörü kullanıyor; step defaults katalog verisi. Yeni provider veya image-list gibi yeni sözleşme için adaptör/profile geliştirmesi gerekiyor; README'de açıklandı.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [README](../README.md) Yeni model / workflow ekleme bölümünün son paragrafını oku; iki fal katalog örneğinin inputProfile alanlarını karşılaştır.

**Değerlendirici nasıl inceleyebilir?** [Model/profile doğrulamasını](../internal/workflow/validate.go) ve [sağlayıcı istemcisini](../internal/providers/client.go) incele. Her model/parametre kombinasyonu için sınırsız destek iddiası yok.

## 19. Node modelRef alanında alias

**Orijinal metin — kaynak satır 75–75:**

<!-- case-source: start=75 end=75 -->
```text
Node'lar sağlayıcıyı doğrudan değil, katalogdaki alias 'ı referans almalıdır.
```

**Biz ne yaptık:** Workflow node'ları flux-schnell, flux-dev ve prompt-writer alias'larını referans alıyor. Gerçek modelId katalogdan snapshot'a çözülüyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [ürün JSON'u](../examples/workflows/product.json)/[etkinlik JSON'u](../examples/workflows/event.json) içindeki modelRef ile /models içindeki alias/modelId alanlarını karşılaştır.

**Değerlendirici nasıl inceleyebilir?** domain.Node/Model ve startRun snapshot çözümlemesini izle; workflow içinde provider endpoint ID'sinin modelRef yerine kullanılmadığını kontrol et.

## 20. Kayıtsız modele izin vermeme

**Orijinal metin — kaynak satır 76–77:**

<!-- case-source: start=76 end=77 -->
```text
Katalogda olmayan bir modele referans veren workflow kaydedilememelidir. Sistem hiçbir koşulda modeli
"tahmin etmemelidir".
```

**Biz ne yaptık:** Unknown/disabled alias doğrulama ve run hazırlığında reddediliyor. Fallback/model tahmini yapılmıyor; hatalı create/update DB'yi değiştirmiyor.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** JSON editöründe taslak modelRef'i not-registered yapıp Doğrula'ya bas; UNKNOWN_MODEL_ALIAS bulgusu bekle. Taslağı kaydetmeden yenile.

**Değerlendirici nasıl inceleyebilir?** [doğrulama kodu](../internal/workflow/validate.go) ve [doğrulama testleri](../internal/workflow/validate_test.go) ve [metadata/write testini](../internal/integration/validation_test.go) incele; public validate/create/update bulguları [sunucu tamamlama kanıtı](evidence/deployment-fixes.json) içinde.

## 21. Graf JSON sözleşmesi

**Orijinal metin — kaynak satır 78–80:**

<!-- case-source: start=78 end=80 -->
```text
Workflow tanımı ve doğrulama
Workflow bir JSON graf olarak tanımlanmalıdır: node'lar ( id , kind , modelRef , params , prompt
şablonu) ve edge'ler ( from , fromPort , to , toPort ).
```

**Biz ne yaptık:** schemaVersion:1, nodes id/kind/modelRef/params/promptTemplate, edges from/fromPort/to/toPort kullanılıyor. Input/output gibi model çağırmayan node'larda modelRef gerekmez.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [ürün JSON'u](../examples/workflows/product.json) ile [JSON şemasını](../schemas/workflow.schema.json) yan yana aç.

**Değerlendirici nasıl inceleyebilir?** [Domain türlerini](../internal/domain/types.go), schema.go ve örnekleri karşılaştır; şemaya uygun verinin backend semantik doğrulamasından da geçtiğini gör.

## 22. Node ve port türleri

**Orijinal metin — kaynak satır 81–81:**

<!-- case-source: start=81 end=81 -->
```text
Node tipleri ( kind ): input , llm , image , output . Port tipleri: text , prompt , image .
```

**Biz ne yaptık:** input/llm/image/output node'ları; text/prompt/image portları destekleniyor. LLM prompt üretir, image prompt alır ve image çıkarır; output üç tipi de kabul edebilir.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** İki örnek workflow'da kind ve edge portlarını incele.

**Değerlendirici nasıl inceleyebilir?** [doğrulama kodu](../internal/workflow/validate.go) ve [doğrulama testleri](../internal/workflow/validate_test.go) port denetimlerini ve TestRejectGraphRules içindeki port mismatch alt testini çalıştır/oku; tip uyumsuzluğu PORT_TYPE_MISMATCH olmalı.

## 23. Şablon değişkenleri

**Orijinal metin — kaynak satır 82–83:**

<!-- case-source: start=82 end=83 -->
```text
Prompt şablonları {{değişken}} biçiminde girdi değerlerine ve önceki node çıktılarına referans
verebilmelidir.
```

**Biz ne yaptık:** {{input.description}} ve {{nodes.art_director.prompt}} gibi önceki scalar çıktılara referanslar destekleniyor. Çözülemeyen/gelecek node değişkeni reddediliyor; kod çalıştıran template dili kullanılmıyor.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [etkinlik JSON'u](../examples/workflows/event.json) poster.promptTemplate alanını ve [etkinlik Dev run](https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321) içinde üretilen prompt'u incele.

**Değerlendirici nasıl inceleyebilir?** TestInputAndTemplate, template future/unknown alt testleri ve workflow.Render koduyla çözümleme sırasını doğrula.

## 24. Doğrusal ve bağlı akış

**Orijinal metin — kaynak satır 84–84:**

<!-- case-source: start=84 end=84 -->
```text
Akış doğrusal olmalıdır: tek input , tek output , aradaki her node'un en fazla bir girişi ve bir çıkışı vardır.
```

**Biz ne yaptık:** Tek input/output, bütün node'ların bağlı tek zinciri zorunlu. Cycle, fan-out, fan-in ve kopuk portlar reddediliyor.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Bir taslakta aynı input'tan ikinci edge ekleyip Doğrula kullan; çoklu outgoing bulgusu bekle.

**Değerlendirici nasıl inceleyebilir?** TestRejectGraphRules cycle/fanout/fanin/missing port alt testleri ve validator bağlantı kontrollerini incele; sonraki node'lara dallanarak yürütme yapılmamalı.

## 25. Create/update zorunlu ve toplu doğrulama

**Orijinal metin — kaynak satır 85–88:**

<!-- case-source: start=85 end=88 -->
```text
Workflow kaydedilirken (oluşturma ve güncellemede) doğrulama zorunludur ve tek bir hata mesajı değil,
bulguların listesini döndürmelidir. En az şunlar kontrol edilmelidir: döngü, bir adımın birden fazla adıma
bağlanması, bağlanmamış zorunlu port, uyumsuz port tipi, katalogda olmayan model, çözülemeyen şablon
değişkeni.
```

**Biz ne yaptık:** Standalone validate, create ve update bağımsız bulguların listesini döndürüyor. Zorunlu altı kuralın tamamı testli; ayrıca UUID, body/path mismatch, duplicate ID, fan-in gibi kontroller var. Geçersiz write eski kaydı değiştirmiyor.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Editörde aynı anda bilinmeyen alias, çözülemeyen template ve eksik edge oluştur; Doğrula tek hata yerine bir bulgu listesi göstermeli.

**Değerlendirici nasıl inceleyebilir?** TestIndependentFindings ve TestWorkflowMetadataFindingsAgreeAcrossValidationAndWrites testlerini incele. validate 200/valid:false; create/update 422/findings, DB unchanged beklenir.

## 26. Run'ın hemen kabul edilmesi

**Orijinal metin — kaynak satır 89–90:**

<!-- case-source: start=89 end=90 -->
```text
Çalıştırma
POST /workflows/{id}/runs hemen bir run_id dönmeli, çalıştırma arka planda (worker) yapılmalıdır.
```

**Biz ne yaptık:** POST run ilk istekte HTTP 202/run_id dönüyor. Run, immutable snapshot ve River enqueue aynı PostgreSQL transaction'ında; ayrı worker devam ediyor. Idempotent replay aynı run için 200 dönebilir.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [sunucu tamamlama kanıtı](evidence/deployment-fixes.json) providerless_worker_check alanını aç; ücretsiz cloud kontrolü succeeded. Dashboard durumları asenkron sorguluyor.

**Değerlendirici nasıl inceleyebilir?** startRun/InsertTx ve TestRunJobTransactionRollbackAndSQLTypes kodunu incele; transaction rollback yarım run veya sahipsiz job bırakmamalı. CI input→output kontrolü worker'ı sağlayıcı çağırmadan sınar.

## 27. Node'ların sıra ve veri aktarımı

**Orijinal metin — kaynak satır 91–91:**

<!-- case-source: start=91 end=91 -->
```text
Node'lar sırayla çalıştırılmalı, her node bir önceki node'un çıktısını kullanabilmelidir.
```

**Biz ne yaptık:** Motor edge sırasına göre input → llm → image → output yürütüyor. Bağlı port veya template ile önceki çıktıyı sonraki node'a aktarıyor.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a) detayında input.description, LLM prompt'u, render girdisi ve görsel sonucunu sırayla karşılaştır.

**Değerlendirici nasıl inceleyebilir?** [çalıştırma motoru](../internal/engine/engine.go) node sıralaması/bağlantı çözümlemesi ile TestSuccessfulRun ve async integration sonucunu incele; node girdi/çıktıları DB'de de saklanıyor.

## 28. Fal queue ve request ID'nin hemen saklanması

**Orijinal metin — kaynak satır 92–94:**

<!-- case-source: start=92 end=94 -->
```text
fal entegrasyonu fal'ın kuyruk (queue) modelini kullanmalıdır: istek gönderilir, durum sorgulanır, sonuç
alınır. Sağlayıcıdan dönen istek kimliği ( provider_request_id ) gönderimden hemen sonra
kaydedilmelidir.
```

**Biz ne yaptık:** Submit → status → result queue akışı var. Submit cevabındaki request ID/status/result URL'leri polling, pricing ve media indirmeden önce DB'ye commit ediliyor; her faz kalıcı. Etkinlik gerçek run'ında ID kalıcılaşması 764 ms olarak kaydedildi.

**Yapıldı mı?** Tamamlandı; adaptör/mock, PostgreSQL/process-kill ve önceki gerçek run kanıtları var.

**Sen nasıl inceleyebilirsin?** [etkinlik Dev run](https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321) image attempt/provider request kaydında ID ve fazları; [gerçek API/run kanıtı](evidence/flowkit-approved.json) içinde kalıcılaşma ölçümünü incele.

**Değerlendirici nasıl inceleyebilir?** [çalıştırma motoru](../internal/engine/engine.go) Submit sonrası save sınırını ve TestKilledWorkerResumesDurableFalTicket testini incele. River'ın attempt-sonu checkpoint'ine güvenilmediğini, doğrudan DB commit yapıldığını doğrula.

## 29. Tipli hata ve maksimum üç attempt

**Orijinal metin — kaynak satır 95–96:**

<!-- case-source: start=95 end=96 -->
```text
Hatalar tiplendirilmelidir (ör. sağlayıcı zaman aşımı, sağlayıcı 5xx, hatalı girdi, kayıtsız model). Yalnızca
geçici sağlayıcı hataları node bazında yeniden denenmelidir (en fazla 3 deneme, artan bekleme).
```

**Biz ne yaptık:** Timeout/network/rate limit/5xx ve kalıcı auth/input/content/response hataları ayrıldı. Güvenli geçici hatalarda maksimum üç toplam domain attempt, 2/4 saniye+jitter ve Retry-After uygulanıyor. Belirsiz submit kör tekrar edilmiyor; queue recovery ayrı.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Hata/retry README bölümünü ve mock retry test sonuçlarını oku; gerçek hata üretmek için ücretli sağlayıcı çağırma.

**Değerlendirici nasıl inceleyebilir?** TestTypedProviderErrors, TestTransientRetryThenSuccess, TestRetryCap, TestPollRetryKeepsRequestAndCost ve TestUncertainSubmissionIsNotRetried testlerini kontrol et; kalıcı hatalarda ek submit beklenmez.

## 30. Başarısız node ve nedenin kalıcı kaydı

**Orijinal metin — kaynak satır 97–97:**

<!-- case-source: start=97 end=97 -->
```text
Başarısız bir run'ın hangi node'da ve neden başarısız olduğu kalıcı olarak kaydedilmelidir.
```

**Biz ne yaptık:** Run.failed_node_id, tipli error ve node/attempt/provider hata detayları saklanıyor; sonraki node'lar skipped oluyor. Provider kod/mesaj ve HTTP durum korunuyor, sırlar maskeleniyor.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** TestPermanentFailureCorrectNode çıktısını ve run detay türlerinde failed_node_id/error alanlarını incele; mevcut gerçek run'lar başarılı olduğu için canlı failure kanıtı diye gösterilmez.

**Değerlendirici nasıl inceleyebilir?** [PostgreSQL/worker testleri](../internal/integration/integration_test.go) permanent alt testinde render/INVALID_INPUT/bad_request beklentisini ve DB'den okunmuş run'ı doğrula; [store.SaveRun](../internal/storage/store.go) kalıcı kayıt kodunu izle.

## 31. Mock testler ve gerçek demo ayrımı

**Orijinal metin — kaynak satır 98–101:**

<!-- case-source: start=98 end=101 -->
```text
Sağlayıcılar
Testler gerçek sağlayıcıları çağırmamalıdır. Testlerde sağlayıcı çağrılarının yerine sahte (mock) bir yanıt
kullanın: sabit bir çıktı dönsün ve test istediğinde hata üretebilsin (ör. zaman aşımı, 5xx). Demo ve
README'deki örnekler gerçek sağlayıcılarla çalışmalıdır.
```

**Biz ne yaptık:** Unit/integration testleri sahte completion/queue/pricing/media ve local httptest kullanıyor. Gerçek doğrudan deney ve üç FlowKit kabul run'ı ayrı onaylarla yapıldı; testler bunları çağırmıyor. Cloud'da geçmiş gerçek sonuçlar taşındı; bu son turda yeni ücretli cloud üretimi yapılmadı.

**Yapıldı mı?** Tamamlandı; test ve gerçek kanıt birbirinden ayrılmıştır.

**Sen nasıl inceleyebilirsin?** [başarılı CI](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38243986198) test sonuçlarıyla [doğrudan deneme kanıtı](evidence/direct.json)/[gerçek API/run kanıtı](evidence/flowkit-approved.json) canlı kanıtlarını ayrı incele.

**Değerlendirici nasıl inceleyebilir?** mockProvider/fake/fakeMedia/httptest.NewServer kullanımlarını ve CI'nin boş provider key'lerini incele. Başarılı mock testini yeni gerçek cloud üretimiyle karıştırma.

## 32. Merkez şart: doğrudan iki adımlı gerçek deney

**Orijinal metin — kaynak satır 102–107:**

<!-- case-source: start=102 end=107 -->
```text
İlk akışın taşınması (çalışmanın merkezi)
En az iki adımlı bir akışı önce doğrudan API çağrılarıyla deneyin: bir LLM adımı (OpenRouter) girdiden
prompt üretir, bir görsel modeli (fal.ai) bu prompt'la görsel üretir. Konu serbesttir (ör. ürün açıklamasından

katalog görseli). Modellerin API dokümantasyonu ve parametreleri fal.ai ve OpenRouter'ın herkese açık
model sayfalarında yer alır.
```

**Biz ne yaptık:** Ürün açıklaması OpenRouter openai/gpt-6-luna ile İngilizce prompt'a; prompt fal-ai/flux/schnell queue ile 512×512 görsele dönüştürüldü. Gerçek model/parametre/cevaplar kanıt dosyasında; API adaptörleri verilen doküman sözleşmeleriyle hazırlandı.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [doğrudan deneme kanıtı](evidence/direct.json) içinde completion usage/prompt ve fal request/status/result alanlarını aç. Bu deney standalone olduğu için /runs'ta dördüncü AI run olarak bulunmaz.

**Değerlendirici nasıl inceleyebilir?** README doğrudan deney bölümünü cmd/direct ve provider client ile karşılaştır; kaynak dosyadaki bu paragrafın ortasında bulunan boş satır aşağıdaki alıntıda aynen korunmuştur.

## 33. Doğrudan deney → FlowKit eşleştirme tablosu

**Orijinal metin — kaynak satır 108–109:**

<!-- case-source: start=108 end=109 -->
```text
Bu akışı FlowKit'e bir workflow olarak taşıyın. Denemede kullandığınız her adımın (model, prompt,
parametreler) sisteminizde neye karşılık geldiğini README'de bir eşleştirme tablosuyla gösterin.
```

**Biz ne yaptık:** README tablosu input.description, promptTemplate, prompt-writer/Schnell alias'ları, LLM parametreleri, image size/steps, queue fazları ve artifact/provenance karşılığını veriyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [README](../README.md) İlk doğrudan akış → FlowKit tablosunu [ürün JSON'u](../examples/workflows/product.json) ile karşılaştır.

**Değerlendirici nasıl inceleyebilir?** [doğrudan deneme kanıtı](evidence/direct.json) içindeki kullanılan model/prompt/parametre değerlerini [Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a) snapshot'ı ve README tablosuyla eşleştir; sadece kavramsal adlar değil gerçek değerler kontrol edilebilir.

## 34. İkinci workflow'un yalnız veri olarak eklenmesi

**Orijinal metin — kaynak satır 110–111:**

<!-- case-source: start=110 end=111 -->
```text
Ardından ikinci, farklı bir workflow'u yalnızca veri olarak (API veya dashboard üzerinden) ekleyin. Bu
işlem kod değişikliği ve deploy gerektirmemelidir.
```

**Biz ne yaptık:** Etkinlik afişi POST /workflows ile eklendi. Aynı API/worker, farklı promptTemplate/modelRef/params verisini yorumladı; kabul boyunca kod/restart/deploy değişmedi.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [etkinlik JSON'u](../examples/workflows/event.json), [etkinlik Dev run](https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321) ve [gerçek API/run kanıtı](evidence/flowkit-approved.json) içindeki ikinci workflow POST/response kaydını aç.

**Değerlendirici nasıl inceleyebilir?** Yeni workflow mutasyonu öncesi/sonrası runtime kimliklerini karşılaştır; TestHotAddModelAndWorkflowWithoutRestart testinde ek workflow'un aynı çalışan worker ile tamamlandığını kontrol et.

## 35. İki görsel modelinin katalogda olması

**Orijinal metin — kaynak satır 112–112:**

<!-- case-source: start=112 end=112 -->
```text
İki workflow'un görsel adımları farklı modeller kullanmalıdır; her iki model de katalogda kayıtlı olmalıdır.
```

**Biz ne yaptık:** Nihai ürün workflow'u flux-schnell, etkinlik workflow'u flux-dev kullanıyor; iki alias enabled katalog kayıtları. Önceki Dev'e geçiş run'ının model snapshot'ı korunuyor.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Giriş sonrası /models ve /workflows aç; ürün/etkinlik modelRef ile Schnell/Dev modelId değerlerini karşılaştır.

**Değerlendirici nasıl inceleyebilir?** [sunucu tamamlama kanıtı](evidence/deployment-fixes.json) ve [gerçek API/run kanıtı](evidence/flowkit-approved.json) üzerinden üç model/iki workflow ile iki farklı producer modelin kaydını doğrula.

## 36. JSON log, bağlam ve provider error seviyesi

**Orijinal metin — kaynak satır 113–115:**

<!-- case-source: start=113 end=115 -->
```text
Gözlemlenebilirlik
Her log satırı yapılandırılmış (JSON) olmalı ve en az run_id , node_id , attempt taşımalıdır. Hatalar da
(sağlayıcıdan dönen hata kodu ve mesajıyla) error seviyesinde loglanmalıdır.
```

**Biz ne yaptık:** API/worker/init/bootstrap/River logları JSON. DB/Nginx/Caddy/gate logları da normalize edildi; run_id/node_id/attempt her satırda, bağlam yoksa null. Provider hataları ERROR, provider kod/mesajıyla; credential redaction var.

**Yapıldı mı?** Tamamlandı; unit/Compose ve tüm sekiz cloud servisinin log kontrolü var.

**Sen nasıl inceleyebilirsin?** Rapordaki E4 komutuyla sunucu loglarını oku; JSON/context alanları olmalı. Tam input/prompt/output için run izini aç.

**Değerlendirici nasıl inceleyebilir?** [Telemetry testini](../internal/telemetry/log_test.go), [logwrap kod/testini](../cmd/logwrap/main_test.go), production smoke ve deployment-fixes server_audit.all_service_logs_json_context alanını incele.

## 37. Görsel yanında gerçek üretici model

**Orijinal metin — kaynak satır 116–116:**

<!-- case-source: start=116 end=116 -->
```text
Dashboard'da her üretimin yanında, çıktıyı gerçekte hangi modelin ürettiği gösterilmelidir.
```

**Biz ne yaptık:** Çıktı provenance/actual_model_id kaydı dashboard'da görselin yanında gösteriliyor. Tarihsel run, güncel workflow/catalog değişse bile kendi üretici modelini kullanıyor.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [yayındaki dashboard](https://flowkit.canhurkurtoglu.com/) ürün/etkinlik çıktılarındaki model etiketlerine bak; [model değişimi Dev run](https://flowkit.canhurkurtoglu.com/runs/217fb931-3af8-4313-89b2-436985798449) Dev kaldığı hâlde güncel ürün workflow'u Schnell olmalı.

**Değerlendirici nasıl inceleyebilir?** [main.tsx](../web/src/main.tsx) görüntü/model render'ını, [ürün ekranını](dashboard-schnell-live.jpg), [etkinlik ekranını](dashboard-event-live.jpg) ve run.output provenance verisini karşılaştır.

## 38. Run'da model/deneme/süre/hata sorularının cevabı

**Orijinal metin — kaynak satır 117–118:**

<!-- case-source: start=117 end=118 -->
```text
Bir run'ın detayından şu sorular cevaplanabilmelidir: Her node'da hangi model kullanıldı, kaç deneme
yapıldı, ne kadar sürdü? Başarısızsa hangi node'da ve neden?
```

**Biz ne yaptık:** Run detayı immutable workflow/model snapshot, node ve attempt zamanları/süreleri, request ID, girdi/çıktı, faz, maliyet ve failed node/error bilgilerini içeriyor.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a)/[etkinlik Dev run](https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321) aç veya dashboard'da node satırlarını genişlet; kullanılan model, attempt sayısı ve süreler görünmeli.

**Değerlendirici nasıl inceleyebilir?** Domain NodeRun/Attempt/Failure türleri, normalized node_runs/node_attempts/provider_requests tabloları ve failure integration assertion'larını incele. Tarihsel cloud objelerinin orijinallerle tam eşitliği kanıtta var.

## 39. Zorunlu test senaryoları

**Orijinal metin — kaynak satır 119–122:**

<!-- case-source: start=119 end=122 -->
```text
Testler
En az şunları kapsamalıdır: doğrulamanın en az üç farklı kuralı reddetmesi; mock sağlayıcıyla başarılı bir
run; geçici hatada retry ve ardından başarı; kalıcı hatada run'ın doğru node ve hata koduyla başarısız
olması; workflow ve katalog için temel CRUD.
```

**Biz ne yaptık:** Altı graf kuralı ve ek kurallar; mock başarılı run; geçici retry→başarı; üç attempt sınırı; kalıcı doğru-node hata; model/workflow CRUD testli. Integration gerçek PostgreSQL/River kullanır; sağlayıcılar local mock. Frontend 13 test de var.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [başarılı CI](https://github.com/CanhK-2103/trypix-flowkitCase/actions/runs/38243986198) backend/dashboard işlerini aç. E3 komutlarıyla ayrı test DB'sinde yeniden çalıştırabilirsin.

**Değerlendirici nasıl inceleyebilir?** TestRejectGraphRules, TestCatalogWorkflowCRUDAndValidation, TestAsyncWorkerSuccessRetryAndPermanentFailure ve engine testlerini incele. Sadece go test çıktısına bakma; integration skip olmaması için TEST_DATABASE_URL/CI postgres kontrolü yap.

## 40. GitHub'da paylaşım

**Orijinal metin — kaynak satır 123–124:**

<!-- case-source: start=123 end=124 -->
```text
Paylaşım ve çalıştırma
Kod GitHub'da olmalıdır; public ya da trypix.ai@gmail.com adresine erişim verilmiş private repo.
```

**Biz ne yaptık:** Repository public: CanhK-2103/trypix-flowkitCase. Kod, README, AGENTS, test, schema ve anahtarsız kanıt dosyaları repoda; private repo erişim daveti gerekmiyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [GitHub repository](https://github.com/CanhK-2103/trypix-flowkitCase) adresini oturumsuz/gizli pencerede aç.

**Değerlendirici nasıl inceleyebilir?** Repository ağacını, commit geçmişini ve Actions işlerini incele; .env/provider key/site parolası Git'te olmamalı. Public repo alternatif şartı karşılar.

## 41. Tek komut, .env.example ve çalıştırma adımları

**Orijinal metin — kaynak satır 125–127:**

<!-- case-source: start=125 end=127 -->
```text
Sistemin tamamı (API, worker, veritabanı, dashboard) docker compose up ile tek komutla ayağa
kalkmalıdır. Gerekli anahtarlar için bir .env.example dosyası bulunmalı, çalıştırma adımları README'de
anlatılmalıdır.
```

**Biz ne yaptık:** Standart compose.yaml API/worker/DB/dashboard ve migration/bootstrap'ı tek docker compose up ile açar. .env.example ve README kurulum adımları var. Üretim başlatmadan örnekler API ile yüklenir; kalıcı DB/media volume'ları var.

**Yapıldı mı?** Tamamlandı; temiz Compose kurulumu CI/smoke ile doğrulandı.

**Sen nasıl inceleyebilirsin?** Anahtarsız, ayrı temiz checkout'ta .env.example → .env kopyalayıp docker compose up kullan; http://localhost:8080, iki workflow ve modeller görünmeli.

**Değerlendirici nasıl inceleyebilir?** Compose bağımlılık/healthcheck/init/bootstrap/worker tanımlarını ve CI temiz-stack kontrolünü incele. Production override worker'ı bilinçli açar; standart yerel kurulum tek komut şartını karşılar.

## 42. Opsiyonel deployment ve şifre koruması

**Orijinal metin — kaynak satır 128–129:**

<!-- case-source: start=128 end=129 -->
```text
Bir platforma (ör. Vercel, Railway) deploy etmek isteğe bağlıdır ve artı olarak değerlendirilir. Deploy
ederseniz sistemi herkese açık bırakmayın (basit bir şifre koruması yeterlidir).
```

**Biz ne yaptık:** Kamatera VPS /opt/flowkit'e deploy edildi. Cloudflare DNS, HTTPS, Caddy ve tek şifreli gate var; dashboard/API/artifact girişsiz korumalı. Yalnız HTTP/HTTPS proxy portları dışarıya açık. Site şifresi Git'e yazılmadı.

**Yapıldı mı?** Tamamlandı; public TLS/auth ve cloud servis kontrolleri var.

**Sen nasıl inceleyebilirsin?** Gizli pencerede [yayındaki dashboard](https://flowkit.canhurkurtoglu.com/) aç; tek şifre alanı bekle. Şifresiz /models veya /runs erişimi 401; giriş sonrası 200, logout sonrası yeniden koruma.

**Değerlendirici nasıl inceleyebilir?** [Deployment yönergesini](../deploy/README.md), [TLS/auth kanıtını](evidence/deployment.json), gate testlerini ve server port kontrolünü incele. Erişim şifresi sahibinden özel kanalla alınmalı.

## 43. Temel katalog işlevleri

**Orijinal metin — kaynak satır 130–131:**

<!-- case-source: start=130 end=131 -->
```text
Temel İşlevler
Model kataloğu (API): kayıt, listeleme, devre dışı bırakma.
```

**Biz ne yaptık:** POST /models kayıt, GET /models liste/detay, PATCH /models/{alias} enabled:false ile disable sağlıyor. Geçmiş snapshot'lar model disable edilince değişmiyor.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Giriş sonrası /models kayıtlarını incele; paylaşılan aktif modelleri deneme için devre dışı bırakmak yerine CRUD testini kullan.

**Değerlendirici nasıl inceleyebilir?** [HTTP API kodu](../internal/api/server.go) ve [OpenAPI](openapi.json) create/list/get/patchModel ve TestCatalogWorkflowCRUDAndValidation kodunda kayıt/liste/disable beklentilerini kontrol et. Hard-delete şartı temel katalog listesinde yok.

## 44. Temel workflow işlevleri

**Orijinal metin — kaynak satır 132–132:**

<!-- case-source: start=132 end=132 -->
```text
Workflow: oluşturma, listeleme, güncelleme, silme, doğrulama (bulgu listesi).
```

**Biz ne yaptık:** POST/GET/PUT/DELETE ve standalone validate uygulanmış; delete soft-delete. Hatalı create/update bulgu listesi döndürüp DB'yi koruyor; run snapshot'ları silinmiyor.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Dashboard'da oluştur/düzenle/Doğrula/Kaydet kontrollerini incele; mevcut teslim workflow'larını silmeden test kanıtına bak.

**Değerlendirici nasıl inceleyebilir?** CRUD integration testi ve validation_test.go üzerinden geçerli revision artışı, silme ve geçersiz güncellemede unchanged DB kontrolünü incele.

## 45. Temel run işlevleri

**Orijinal metin — kaynak satır 133–133:**

<!-- case-source: start=133 end=133 -->
```text
Run: başlatma, durum sorgulama, adım adım iz (girdi, çıktı, model, deneme, süre, hata).
```

**Biz ne yaptık:** Run başlatma, filtreli geçmiş, hafif status ve ayrıntılı trace API var; raw input, output, model, attempt, duration, error saklanıyor. Input node raw kayıt düzeltmesi ve tarihsel dashboard izi tamamlandı.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a) JSON'u ile dashboard node detaylarını karşılaştır; girdi, prompt, model ve görsel bağlantısı görünmeli.

**Değerlendirici nasıl inceleyebilir?** GET /runs/{id}/status/detail route'ları, run/storage türleri ve TestSuccessfulRun raw-input assertion'ını incele; cloud providerless message/approved:false kaydı da kanıtta.

## 46. Temel dashboard işlevleri

**Orijinal metin — kaynak satır 134–135:**

<!-- case-source: start=134 end=135 -->
```text
Dashboard: workflow listesi, JSON editörü + doğrula/kaydet butonu, girdi formuyla çalıştırma, run geçmişi
ve run detayı.
```

**Biz ne yaptık:** Workflow listesi, CodeMirror JSON editörü, Doğrula/Kaydet, şemadan input formu, geçmiş ve run detayı var. Farklı alan isimleri/scalar türler/enum ve dokunulmamış false boolean için form düzeltildi/testlendi.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [yayındaki dashboard](https://flowkit.canhurkurtoglu.com/) üzerinde listeden workflow seç → JSON tanımı → Doğrula → geçmiş run → node detayları sırasını izle. Yeni ücretli run başlatmak inceleme için gerekli değil.

**Değerlendirici nasıl inceleyebilir?** WorkflowInput/JsonEditor/Costs/API frontend testleri ve [form regresyon ekranını](input-regression-fixed.jpg) incele; npm test toplam 13 test, npm run build typecheck+build yapar.

## 47. Backend dil tercihi

**Orijinal metin — kaynak satır 136–137:**

<!-- case-source: start=136 end=137 -->
```text
Dil Tercihi
Backend'in Go ile geliştirilmesini tercih ediyoruz. Bu bir zorunluluk değildir; farklı bir dille de geliştirebilirsiniz.
```

**Biz ne yaptık:** Backend Go ile geliştirildi; HTTP API, worker, bootstrap, doğrudan demo ve gate Go programları. SQL erişimi pgx/sqlc, migration Goose.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** go.mod ve cmd/internal dizinlerini aç.

**Değerlendirici nasıl inceleyebilir?** Go modülü ve Dockerfile builder'ını, cmd/flowkit api/worker mode'larını incele; Go test/vet/gofmt CI kontrolleri başarılı.

## 48. Dashboard teknoloji tercihi

**Orijinal metin — kaynak satır 138–138:**

<!-- case-source: start=138 end=138 -->
```text
Dashboard için teknoloji serbesttir.
```

**Biz ne yaptık:** React + TypeScript + Vite; TanStack Query veri/mutation/polling, CodeMirror JSON editörü. Derlenmiş statik dashboard Nginx üzerinden aynı origin API'ye bağlanıyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** web/package.json ile dashboard JSON editörünü aç.

**Değerlendirici nasıl inceleyebilir?** [package.json](../web/package.json), api.ts, JsonEditor.tsx ve Vite/Nginx/Docker target'larını incele; frontend gerçek anahtar taşımaz.

## 49. Açık tasarım kararlarına yaklaşım

**Orijinal metin — kaynak satır 139–140:**

<!-- case-source: start=139 end=140 -->
```text
Açık Tasarım Kararları
Aşağıdakiler bilinçli olarak tanımsız bırakılmıştır ve değerlendirmenin parçasıdır:
```

**Biz ne yaptık:** Belgenin bıraktığı altı tasarım kararı seçildi ve aşağıdaki altı kaynak maddesiyle tek tek eşleştirildi.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [README](../README.md) ile [PLAN](../PLAN.md) tasarım kararlarını oku.

**Değerlendirici nasıl inceleyebilir?** Sonraki altı eşleştirmeyi uygulama koduyla karşılaştır; planın yazılmış olmasıyla uygulanmış/testli davranışı ayrı değerlendir.

## 50. Framework ve veritabanı seçimi

**Orijinal metin — kaynak satır 141–141:**

<!-- case-source: start=141 end=141 -->
```text
Framework ve veritabanı
```

**Biz ne yaptık:** Go chi HTTP router, PostgreSQL, pgx + sqlc ve Goose seçildi. Tek DB, hem uygulama verisini hem River queue/checkpoint transaction'larını taşıyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** go.mod, sqlc.yaml ve db/migrations dizinlerini aç.

**Değerlendirici nasıl inceleyebilir?** API chi.NewRouter; storage.Pool/queries; migration init ve Compose DB bağımlılıklarını izle. Redis gibi ek altyapı kullanılmaması transaction bütünlüğünü sade tutuyor.

## 51. Kuyruk ve worker seçimi

**Orijinal metin — kaynak satır 142–142:**

<!-- case-source: start=142 end=142 -->
```text
Kuyruk/worker mekanizması (veritabanı tabanlı kuyruk, Redis, ayrı süreç vb.)
```

**Biz ne yaptık:** PostgreSQL tabanlı River ve API'den ayrı worker süreci seçildi. InsertTx run ile aynı commit; JobSnooze poll/retry beklemesinde slotu serbest bırakır. DB faz kayıtları durable recovery sağlar.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Sunucuda E4 servis komutuyla worker running, API ayrı servis gör; providerless cloud run başarılı.

**Değerlendirici nasıl inceleyebilir?** [jobs.go](../internal/jobs/jobs.go), startRun, transaction rollback ve process-kill testlerini incele. Domain attempt ile River altyapı retry'sini ayırt et.

## 52. Ek graf ayrıntıları

**Orijinal metin — kaynak satır 143–143:**

<!-- case-source: start=143 end=143 -->
```text
Graf JSON'unun zorunlu alanlar dışındaki ayrıntıları
```

**Biz ne yaptık:** schemaVersion:1, UUID workflow kimliği/revision, düz scalar inputSchema/textField, profil uyumlu params, promptTemplate ve model snapshot sözleşmesi var. Bulgular code/message/path/node_id taşıyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [JSON şemasını](../schemas/workflow.schema.json), [ürün JSON'u](../examples/workflows/product.json)/[etkinlik JSON'u](../examples/workflows/event.json) ve validate yanıtını incele.

**Değerlendirici nasıl inceleyebilir?** domain types, schema.go, metadata_test.go ve validation_test.go üzerinde yapısal/semantik/metadata doğrulama ayrımını kontrol et.

## 53. Retry süreleri ve geçici hata tanımı

**Orijinal metin — kaynak satır 144–144:**

<!-- case-source: start=144 end=144 -->
```text
Retry bekleme süreleri ve hangi hataların "geçici" sayıldığı
```

**Biz ne yaptık:** Maksimum üç domain attempt; 2/4 sn + 0–250 ms jitter; daha uzun Retry-After gözetilir. Timeout/network/429/5xx geçici; auth/payment/input/refusal/bozuk sonuç kalıcı. Belirsiz submission tekrar edilmez.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** README Hata, retry ve bonuslar bölümünü oku; retry testleri üzerinden sonuçları incele.

**Değerlendirici nasıl inceleyebilir?** providers.failure ve engine.providerFailure kodu; RetryCap/TypedProviderErrors/ResponseLossNotResubmitted testlerinde gerçek sınırı kontrol et. Kalıcı node deadline retry/restart ile sıfırlanmaz.

## 54. Sonucun istemciye iletilmesi

**Orijinal metin — kaynak satır 145–145:**

<!-- case-source: start=145 end=145 -->
```text
Sonucun istemciye iletilmesi (polling, SSE, WebSocket)
```

**Biz ne yaptık:** HTTP polling seçildi. Aktif run detayı TanStack Query ile 2 sn; bitmiş fakat ücret pending ise 15 sn; tamamlanmış/erişilemeyen ücret durumunda maliyet polling durur. Dashboard mutation retry kapalı.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Browser Network bölümünde geçmiş terminal run ile aktif durum sorgulamasını/API istemci testini incele; provider billing cloud'da kapalıdır.

**Değerlendirici nasıl inceleyebilir?** api.ts pollInterval/active ve api.test.ts; main.tsx refetchInterval/mutation ayarlarını karşılaştır. UI'nin DB okumaları sağlayıcı billing çağrısı değildir.

## 55. Görsel saklama seçimi

**Orijinal metin — kaynak satır 146–147:**

<!-- case-source: start=146 end=147 -->
```text
Görsel çıktıların saklanması

```

**Biz ne yaptık:** Görseller backend tarafından MIME/20 MiB sınırı ve atomik dosya taşıma ile kalıcı Docker volume'a indirilir; DB artifact kaydı/provenance saklanır. /artifacts/{id} sağlar; provider URL ömrüne bağımlı değildir.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Dashboard Görseli aç bağlantısını kullan; kalıcı /artifacts adresi olmalı. Üç mevcut görsel cloud'da açılmalı.

**Değerlendirici nasıl inceleyebilir?** [files.go](../internal/artifacts/files.go), Compose artifact volume ve deployment-fixes içindeki üç HTTP200/SHA256 kaydını incele; cloud dosyalarının orijinallerle aynı olduğu doğrulandı.

## 56. Bonusların önceliği

**Orijinal metin — kaynak satır 148–150:**

<!-- case-source: start=148 end=150 -->
```text
Opsiyonel Bonus
Zorunlu kısımları tamamladıktan sonra vaktiniz kalırsa, aşağıdaki bonuslardan bir veya birkaçını
ekleyebilirsiniz. Bonuslar zorunlu değildir; önceliğiniz zorunlu gereksinimlerin eksiksiz ve çalışır olmasıdır.
```

**Biz ne yaptık:** Üç bonus da eklendi: worker restart dayanıklılığı, idempotency ve tahmini maliyet izi. Zorunlu fonksiyonlar/testler tamamlandıktan sonra bonus kabul kontrolleri kaydedildi.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** Aşağıdaki üç eşleştirme ve README bonus bölümünü incele.

**Değerlendirici nasıl inceleyebilir?** Bonus kanıtlarını zorunlu API/validation/worker testlerinin yerine kullanma; hem temel fonksiyon hem bonus kontrolleri CI'da var.

## 57. Bonus: worker yeniden başlatma dayanıklılığı

**Orijinal metin — kaynak satır 151–152:**

<!-- case-source: start=151 end=152 -->
```text
Worker yeniden başlatma dayanıklılığı: Worker bir fal isteği sürerken çökerse, yeniden başladığında
aynı isteği tekrar göndermeden kaldığı yerden devam etsin. Bunu bir testle ya da kısa bir kayıtla gösterin.
```

**Biz ne yaptık:** DB'de bilinen fal request ID'si varsa yeni submit yapılmadan poll/result devam ediyor. Kayıtlı sending fakat ID yoksa PROVIDER_SUBMISSION_UNKNOWN ile güvenli fail oluyor. Gerçek worker process kill + local mock fal + gerçek PostgreSQL testi başarılı.

**Yapıldı mı?** Tamamlandı; bonusun izin verdiği test yoluyla gösterildi. Yeni ücretli fal isteği sırasında VPS kill demosu yapılmadı.

**Sen nasıl inceleyebilirsin?** E3 testi TestKilledWorkerResumesDurableFalTicket adıyla çalıştır veya CI çıktısını oku; ikinci worker succeeded/tek submit beklenir.

**Değerlendirici nasıl inceleyebilir?** TestResumeKnownFalRequest ve process-kill kodunda request-ID commit sonrası worker'ın öldürüldüğünü; ikinci süreçte aynı ticket/tek LLM completion/tek image attempt/tek charge kaldığını doğrula.

## 58. Bonus: idempotency key

**Orijinal metin — kaynak satır 153–154:**

<!-- case-source: start=153 end=154 -->
```text
Tekrar gönderime karşı koruma: Aynı istemci isteği iki kez gelirse ikinci bir run oluşmasın (idempotency
key).
```

**Biz ne yaptık:** Idempotency-Key + aynı workflow/input aynı run'a döner; farklı body 409. Global key run ömrünce kalıcı; workflow değişse/silinse tarihsel replay korunur. Run/job transaction ve on eşzamanlı istek tek-run testi var. Cloud ücretsiz replay 200/aynı ID doğrulandı.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [sunucu tamamlama kanıtı](evidence/deployment-fixes.json) providerless_worker_check.idempotency_replay ve run ID alanını incele; E3 concurrent testi kullan.

**Değerlendirici nasıl inceleyebilir?** TestConcurrentIdempotencyAndSnapshot ve startRun hash/unique-conflict/replay sırasını incele. Ücretli aynı isteği yeniden göndererek test etmen gerekmez.

## 59. Bonus: node tahminleri ve toplam maliyet izi

**Orijinal metin — kaynak satır 155–155:**

<!-- case-source: start=155 end=155 -->
```text
Maliyet izi: Her adımın tahmini sağlayıcı maliyeti kaydedilsin ve run detayında toplamı gösterilsin.
```

**Biz ne yaptık:** Başlamadan node ve toplam tahmin, immutable initial_estimate; run içinde güncel tahminler ayrı saklanıyor. Decimal string/NUMERIC ile exact aritmetik. Ek istek olan gerçek node/toplam alanları da kod/mock testli; OpenRouter actual canlı mevcut, fal actual/gerçek total yetki nedeniyle null. Billing cloud'da kapalı.

**Yapıldı mı?** Case'in tahmini maliyet bonusu tamamlandı. Ek fal gerçek ücret/gerçek toplam canlı doğrulaması kullanıcı kararıyla kapsam dışı.

**Sen nasıl inceleyebilirsin?** Formdaki başlangıç tahminine, geçmiş run maliyet sütunlarına bak. Bilinmeyen actual sıfır veya tahmini değer gibi gösterilmemeli; /capabilities billing_enabled:false olmalı.

**Değerlendirici nasıl inceleyebilir?** quote/cost/billing kodu, [billing testleri](../internal/engine/billing_test.go), TestQuoteAndDelayedBillingPersistedWithAPIScheduledRefresh ve initial_estimate snapshot testlerini incele. Ücretlerin tam bilindiği mock örnekte gerçek total, eksik örnekte null beklenir.

## 60. Değerlendirme: motor dayanıklılığı

**Orijinal metin — kaynak satır 156–157:**

<!-- case-source: start=156 end=157 -->
```text
Değerlendirme Kriterleri
Çalıştırma motorunun doğruluğu ve dayanıklılığı: asenkron akış, tipli hatalar, retry
```

**Biz ne yaptık:** Asenkron River, atomik enqueue, tipli provider hataları, transient-only retry ve durable phase recovery uygulanıp testlendi.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** CI backend işinde async/transaction/retry/restart testlerini incele.

**Değerlendirici nasıl inceleyebilir?** [çalıştırma motoru](../internal/engine/engine.go), jobs.go ve [PostgreSQL/worker testleri](../internal/integration/integration_test.go) davranışını oku; test sonuçları ölçülebilir kanıt, nihai kalite puanı değerlendiricinin kararıdır.

## 61. Değerlendirme: doğrulama

**Orijinal metin — kaynak satır 158–158:**

<!-- case-source: start=158 end=158 -->
```text
Doğrulama: hatalı bir workflow'un kaydedilmemesi, bulgu listesinin anlaşılırlığı
```

**Biz ne yaptık:** Toplu code/message/path/node_id bulguları, zorunlu create/update doğrulaması ve no-write-on-invalid davranışı var. Metadata UUID/mismatch tutarlılığı ayrıca düzeltildi.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** Birden çok bozuk alan içeren taslağı Doğrula; tüm bağımsız bulgular listelenmeli.

**Değerlendirici nasıl inceleyebilir?** Public metadata_validation kanıtı, TestIndependentFindings ve metadata write integration testini karşılaştır; geçersiz update sonrası revision/definition aynı kalmalı.

## 62. Değerlendirme: temiz soyutlamalar

**Orijinal metin — kaynak satır 159–159:**

<!-- case-source: start=159 end=159 -->
```text
Soyutlamaların temizliği: yeni model ve yeni workflow eklemek kod değişikliği gerektirmiyor
```

**Biz ne yaptık:** Workflow grafı ve uyumlu model kimlikleri data; provider/profile adaptörü sabit sözleşmeyi yürütür. Yeni Dev ve ikinci workflow API ile canlı, restart olmadan eklendi.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [gerçek API/run kanıtı](evidence/flowkit-approved.json) runtime_before/runtime_after_all ve mutasyon isteklerini incele.

**Değerlendirici nasıl inceleyebilir?** Provider/model profile sınırının açıklığını, katalog çözümlemesini ve TestHotAddModelAndWorkflowWithoutRestart testini incele. Yeni modeli kod içi switch-case'e ekleme bulunmamalı.

## 63. Değerlendirme: ilk akışın taşınması

**Orijinal metin — kaynak satır 160–160:**

<!-- case-source: start=160 end=160 -->
```text
İlk akışın sisteme taşınması ve eşleştirmenin netliği
```

**Biz ne yaptık:** Doğrudan gerçek deney, model/prompt/parametre eşleştirme tablosu ve aynı sözleşmeyle ilk başarılı FlowKit run mevcut.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** [doğrudan deneme kanıtı](evidence/direct.json), README tablosu ve [Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a) arasında her adıma karşılık bul.

**Değerlendirici nasıl inceleyebilir?** Standalone deneyin DB run'ı olmadığını, iki kaydın ayrı ama aynı tanımlı akışı gösterdiğini kontrol et. Canlı sıra ve bağımsız kod hazırlama açıklandı.

## 64. Değerlendirme: gözlemlenebilirlik

**Orijinal metin — kaynak satır 161–161:**

<!-- case-source: start=161 end=161 -->
```text
Gözlemlenebilirlik: bir run'da ne olduğunun logdan ve run detayından anlaşılabilmesi
```

**Biz ne yaptık:** JSON/context log, kalıcı phase/node/attempt/model/request izi ve dashboard detayları var; sekiz servis logları cloud'da kontrol edildi.

**Yapıldı mı?** Tamamlandı; testli ve cloud'da kontrol edildi.

**Sen nasıl inceleyebilirsin?** E4 loglarını bir run_id ile ara; dashboard node trace'iyle ilişkilendir.

**Değerlendirici nasıl inceleyebilir?** Telemetry/model provenance/Failure alanları ve deployment-fixes log sayımlarını incele. Log özetleri ile DB'nin ayrıntılı input/output izini ayrı değerlendir.

## 65. Değerlendirme: hata yolu testleri

**Orijinal metin — kaynak satır 162–162:**

<!-- case-source: start=162 end=162 -->
```text
Testlerin hata yollarını da kapsaması
```

**Biz ne yaptık:** Timeout/5xx/rate-limit/permanent input/refusal/response-loss, retry cap, billing permission, delayed accounting ve queue recovery yolları mock testlerle kapsandı.

**Yapıldı mı?** Tamamlandı; mock ve/veya PostgreSQL testleriyle doğrulandı.

**Sen nasıl inceleyebilirsin?** CI backend test isimlerini ve failure assertion'larını oku; bir başarılı run testiyle yetinme.

**Değerlendirici nasıl inceleyebilir?** providers/client_test.go, engine/engine_test.go, engine/billing_test.go ve async integration alt testlerini kontrol et. Gerçek sağlayıcıya hata ürettirmek test gereği değil.

## 66. Değerlendirme: kod düzeni ve kalite

**Orijinal metin — kaynak satır 163–163:**

<!-- case-source: start=163 end=163 -->
```text
Kod düzeni ve mühendislik kalitesi
```

**Biz ne yaptık:** cmd/internal/domain/storage/provider/engine/api ayrımı; generated sqlc, migration, API sözleşmesi, format/vet/test/build kontrolleri var. Kritik davranışlar uygulamaya benzemeyen gerçek DB/process/HTTP testleriyle sınandı.

**Yapıldı mı?** Teknik kontroller tamamlandı; mühendislik kalite değerlendirmesi değerlendiriciye ait.

**Sen nasıl inceleyebilirsin?** Repository ağacını ve CI format/vet/sqlc/typecheck kontrollerini incele.

**Değerlendirici nasıl inceleyebilir?** Bağımlılık sınırlarını, transaction ownership, test anlamlılığı ve hata handling'i code review ile değerlendir. Bu rapor kalite puanı veya kusursuzluk garantisi vermez.

## 67. Değerlendirme: AI kullanımının şeffaflığı

**Orijinal metin — kaynak satır 164–164:**

<!-- case-source: start=164 end=164 -->
```text
AI araçlarının kullanımının şeffaflığı
```

**Biz ne yaptık:** AGENTS.md proje kuralları, README kullanım özeti ve AI_LOG gerçek hata/düzeltme kaydı repoda. Max token/deprecated alan, River checkpoint, Origin, form ve test fixture hataları saklanmadı.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [AGENTS.md](../AGENTS.md) ve [AI_LOG.md](AI_LOG.md) dosyalarını oku.

**Değerlendirici nasıl inceleyebilir?** Kayıtlardaki yanlış yaklaşım → fark etme → düzeltme → test zincirini commit/test kanıtıyla karşılaştır; yalnız başarı hikâyesi yazılmamış olmalı.

## 68. Değerlendirme: gereksinim uygunluğu

**Orijinal metin — kaynak satır 165–165:**

<!-- case-source: start=165 end=165 -->
```text
Gereksinimlere uygunluk
```

**Biz ne yaptık:** Bu rapor belgenin tamamını kaynak sırasıyla eşleştiriyor; önceki CASE_AUDIT test/deployment sonuçlarını topluyor. Teknik şartlar için kanıt var; insanın kod hâkimiyeti görüşmede, fal exact accounting ek isteği ise kapsam dışında.

**Yapıldı mı?** Teknik gereksinimler için kanıt sunuldu; insan değerlendirmesi ve açık sınırlar ayrı belirtildi.

**Sen nasıl inceleyebilirsin?** Bu raporu paragraf paragraf işaretle; her kanıt bağlantısını açarak durum ifadesini doğrula.

**Değerlendirici nasıl inceleyebilir?** [Orijinal kaynak](CASE_SOURCE.txt), [önceki denetim](CASE_AUDIT.md) ve alıntı bütünlük manifestini karşılaştır; kanıt türlerinin birbirinin yerine kullanılmadığını kontrol et.

## 69. Değerlendirme: görsel kalitesi kapsam dışında

**Orijinal metin — kaynak satır 166–166:**

<!-- case-source: start=166 end=166 -->
```text
Çıktı (görsel) kalitesi değerlendirmeye dahil değildir.
```

**Biz ne yaptık:** Düşük çözünürlük ve az üretim kullanıldı. Başarı; model doğruluğu, queue sonucu, kalıcı artifact ve iz üzerinden değerlendirildi. Çıktı estetiği için ek ücretli run yapılmadı.

**Yapıldı mı?** Kapsama uygun.

**Sen nasıl inceleyebilirsin?** Mevcut görselin açıldığını ve doğru model etiketini kontrol et; estetik düzeltme için yeni run gerekmiyor.

**Değerlendirici nasıl inceleyebilir?** Artifact HTTP/MIME/boyut/provenance kontrollerini değerlendir; kalite iyileştirme şartı ekleme.

## 70. AI talimat dosyası ve hataların anlatılması

**Orijinal metin — kaynak satır 167–170:**

<!-- case-source: start=167 end=170 -->
```text
AI Araçları
Claude Code, Codex, Cursor gibi AI kodlama araçlarının kullanımı serbesttir ve teşvik edilir. Repoya bu
araçlar için kullandığınız proje talimat dosyasını (ör. CLAUDE.md , AGENTS.md ) ekleyin. README'de kısa bir
bölümde nasıl çalıştığınızı anlatın: aracı neyde kullandınız, nerede yanlış yönlendirdi ve bunu nasıl fark ettiniz.
```

**Biz ne yaptık:** Codex kullanıldı; AGENTS.md repoya kondu. README planlama/uygulama/test/dokümantasyon kullanımını ve yanlış max_tokens/checkpoint varsayımının fark edilmesini anlatıyor; AI_LOG ayrıntılı.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** README Model seçimi, bütçe ve AI kullanımı bölümünü, AGENTS.md ve AI_LOG'u sırayla oku.

**Değerlendirici nasıl inceleyebilir?** Belgedeki iki şartı ayrı kontrol et: talimat dosyası mevcut; kullanım/hata/fark etme özeti mevcut. Tam model tool transcript'i veya secret paylaşımı gerekmez.

## 71. Adayın kodu açıklayabilmesi

**Orijinal metin — kaynak satır 171–174:**

<!-- case-source: start=171 end=174 -->
```text
Önemli: Değerlendirdiğimiz şey kodu kimin yazdığı değil, AI'ın ürettiği koda sizin ne kadar hâkim
olduğunuzdur. Repodaki herhangi bir bölümün neden o şekilde yazıldığını açıklayabilmeniz ve
kararlarınızı savunabilmeniz beklenir. Açıklayamadığınız bir kod, yazılmamış kabul edilir.

```

**Biz ne yaptık:** Kararlar Türkçe README/PLAN/AI_LOG'da; kritik akışlar testlerle görünür. Ancak senin her bölümü açıklayabildiğin, otomatik test veya benim raporumla doğrulanamaz. E5 savunma soruları ve kod bağlantıları hazırlanmıştır.

**Yapıldı mı?** Teknik açıklama malzemesi hazır; kişisel kod hâkimiyeti senin incelemen ve değerlendiricinin görüşmesiyle doğrulanmalı.

**Sen nasıl inceleyebilirsin?** E5 sorularını dosyaları kapatıp kendi cümlelerinle yanıtla; sonra ilgili transaction/recovery/retry/maliyet kodunu açıp cevabını kontrol et.

**Değerlendirici nasıl inceleyebilir?** Adaydan belirli bir run'ı, InsertTx/checkpoint sınırını, uncertain submission davranışını ve null actual total nedenini kod üzerinde açıklamasını iste. Bu madde 'otomatik tamamlandı' diye işaretlenmedi.

## 72. Teslim: Compose servis ve dashboard

**Orijinal metin — kaynak satır 175–176:**

<!-- case-source: start=175 end=176 -->
```text
Teslimat
docker compose up ile çalışan servis ve dashboard
```

**Biz ne yaptık:** Tam stack Compose ile çalışıyor; temiz kurulum ve sağlayıcısız worker smoke CI'da. Yayında API/worker/DB/dashboard aktif.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** CI compose işi veya ayrı anahtarsız temiz kurulum; dashboard + örnek kayıtlar görünmeli.

**Değerlendirici nasıl inceleyebilir?** compose.yaml ve tools/check-compose.mjs ile DB→migration→API/bootstrap→dashboard/worker hazır olma sırasını doğrula.

## 73. Teslim: deployment URL ve erişim bilgisi

**Orijinal metin — kaynak satır 177–177:**

<!-- case-source: start=177 end=177 -->
```text
(İsteğe bağlı) deploy edilmiş ortamın URL'i ve erişim bilgisi
```

**Biz ne yaptık:** https://flowkit.canhurkurtoglu.com yayında. Ortak site şifresi ignored yerel .env'de VPS_DASHBOARD_PASSWORD, sunucuda yalnız hash; rapora/Git'e yazılmadı. Şifre değerlendiriciye ayrıca verilmeli.

**Yapıldı mı?** URL/koruma tamamlandı; erişim şifresinin değerlendiriciye özel kanalla iletilmesi teslim edenin işi.

**Sen nasıl inceleyebilirsin?** [yayındaki dashboard](https://flowkit.canhurkurtoglu.com/) girişini dene; yerel şifre bilgisini paylaşılacak sunucu/sağlayıcı anahtarlarıyla karıştırma.

**Değerlendirici nasıl inceleyebilir?** Sahibinden yalnız site erişim şifresini al; public domain TLS, login, korumalı API ve logout testlerini incele. SSH veya provider secret isteme.

## 74. Teslim: GitHub linki

**Orijinal metin — kaynak satır 178–178:**

<!-- case-source: start=178 end=178 -->
```text
GitHub repository linki
```

**Biz ne yaptık:** Public repository bağlantısı README ve teslim dokümanlarında mevcut.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** [Repository](https://github.com/CanhK-2103/trypix-flowkitCase) adresini aç.

**Değerlendirici nasıl inceleyebilir?** Kod ağacı, README, docs/kanıtlar ve Actions erişimini oturumsuz doğrula.

## 75. Teslim README: mimari ve çalışma akışı

**Orijinal metin — kaynak satır 179–180:**

<!-- case-source: start=179 end=180 -->
```text
Kısa bir README:
Mimari genel bakış ve çalıştırma akışı
```

**Biz ne yaptık:** README mimari diyagramı ve API→DB transaction→River worker→provider→artifact akışını anlatıyor; durable fazlar ve polling seçimi yazılı.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** README Mimari ve çalıştırma bölümünü oku; E5 akış sorusunu kendi cümlelerinle anlat.

**Değerlendirici nasıl inceleyebilir?** Diyagramdaki her bileşeni Compose/cmd/internal dosyalarına bağla; anlatımın deployed sistemle uyumlu olduğunu kontrol et.

## 76. Teslim README: endpoint ve örnekler

**Orijinal metin — kaynak satır 181–181:**

<!-- case-source: start=181 end=181 -->
```text
Endpoint listesi ve örnek istekler
```

**Biz ne yaptık:** Endpoint tablosu, cURL örnekleri, response/hata kodları ve OpenAPI bağlantısı var.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** README API bölümünde GET/CRUD/validate/run/status/detail/estimate/artifact/capabilities listelerini gör.

**Değerlendirici nasıl inceleyebilir?** OpenAPI route/schema ile server.go route'larını karşılaştır; yeni maliyet/capabilities endpoint'leri de belgelenmiş olmalı.

## 77. Teslim README: graf şeması ve kurallar

**Orijinal metin — kaynak satır 182–182:**

<!-- case-source: start=182 end=182 -->
```text
Graf JSON şeması ve doğrulama kuralları
```

**Biz ne yaptık:** README Graf ve doğrulama bölümünde schemaVersion, kind/port, inputSchema/template/alias ve bağlı doğrusal kurallar var; JSON schema ve iki örneğe link veriyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** README graf bölümünü ve examples/workflows JSON'larını aç.

**Değerlendirici nasıl inceleyebilir?** Doküman kurallarını schemas/workflow.schema.json ve workflow.Validate testleriyle karşılaştır; metadata UUID ve toplu findings davranışı dahil.

## 78. Teslim README: hata sınıfları ve retry

**Orijinal metin — kaynak satır 183–183:**

<!-- case-source: start=183 end=183 -->
```text
Hata sınıfları ve retry politikası
```

**Biz ne yaptık:** Geçici/kalıcı sınıflar, max 3 deneme, 2/4 sn+jitter/Retry-After, node deadline, recovery ayrımı ve uncertain submit sınırı anlatılmış.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** README Hata, retry ve bonuslar bölümünü oku.

**Değerlendirici nasıl inceleyebilir?** Belgedeki süre/sınıfları client/engine implementation ve RetryCap/typed-error testleriyle karşılaştır.

## 79. Teslim README: eşleştirme ve iki örnek run

**Orijinal metin — kaynak satır 184–184:**

<!-- case-source: start=184 end=184 -->
```text
İlk akış → FlowKit eşleştirme tablosu ve iki workflow'un örnek run'ları
```

**Biz ne yaptık:** İlk doğrudan akış eşleştirme tablosu; ilk Schnell, model değişimi Dev ve ikinci etkinlik Dev run kimlikleri, artifact ve maliyet bilgileri var. İki nihai workflow farklı model kullanıyor.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** README tabloyu oku; [Schnell run](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a) ve [etkinlik Dev run](https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321) bağlantılarını giriş yaptıktan sonra aç.

**Değerlendirici nasıl inceleyebilir?** README anlatımıyla direct.json/flowkit-approved.json run snapshot ve deployment-fixes taşıma kanıtını karşılaştır; cloud'da yeniden üretim yapıldığı iddiası yok.

## 80. Teslim README: yeni model/workflow gerçek istekleri

**Orijinal metin — kaynak satır 185–186:**

<!-- case-source: start=185 end=186 -->
```text
"Yeni bir model / workflow nasıl eklenir" bölümü; yukarıdaki yeni model senaryosunun gerçek API
istekleri ve sonuç run'ıyla birlikte
```

**Biz ne yaptık:** Yeni model / workflow ekleme bölümü POST Dev, PUT modelRef değişimi, POST ikinci workflow örnekleri ve profil sınırı içeriyor. Tam gerçek request/response gövdeleri, sonuç run ve restart olmadığının kanıtı JSON'da.

**Yapıldı mı?** Tamamlandı; önceki onaylı gerçek sağlayıcı denemesiyle doğrulandı.

**Sen nasıl inceleyebilirsin?** README bölümünü ve flowkit-approved.json mutasyon gövdelerini aç; Dev run succeeded gör.

**Değerlendirici nasıl inceleyebilir?** POST201/PUT200 revision2/POST202 ve terminal run kaydını, runtime kimlikleriyle birlikte doğrula. Şu an cloud'da seed Dev mevcut diye historical hot-add kanıtını geçersiz sayma; yeni-demo olarak tekrar seedleme de sayılmaz.

## 81. Teslim README: AI ile çalışma

**Orijinal metin — kaynak satır 187–187:**

<!-- case-source: start=187 end=187 -->
```text
AI araçlarıyla nasıl çalıştığınız
```

**Biz ne yaptık:** README Codex kullanım alanları, yanlış yönlendirme ve düzeltmeyi özetliyor; AGENTS/AI_LOG bağlantıları var.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** Model seçimi, bütçe ve AI kullanımı bölümündeki Codex paragrafını oku.

**Değerlendirici nasıl inceleyebilir?** Özetin gerçek AI_LOG kayıtlarıyla tutarlılığını ve kritik kararların testli olduğunu kontrol et.

## 82. Teslim README: bir hafta daha olsaydı

**Orijinal metin — kaynak satır 188–188:**

<!-- case-source: start=188 end=188 -->
```text
Bir hafta daha olsaydı neyi farklı yapacağınız (tek paragraf)
```

**Biz ne yaptık:** Son tek paragrafta çoklu host object storage, provider-supported idempotency, fiyat güncelleme, maliyet raporu, metrics/traces ve genişletilebilir profile şemaları önerildi; mevcut tek-host/volume/accounting sınırları açıklandı.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** README'nin son paragrafını oku.

**Değerlendirici nasıl inceleyebilir?** Önerilerin mevcut sınırlarla ilişkisini sorgula; tamamlanmamış zorunlu gereksinimleri 'gelecek hafta' diye ertelemek için kullanılmadığını kontrol et.

## 83. Teslim README: bonuslar

**Orijinal metin — kaynak satır 189–190:**

<!-- case-source: start=189 end=190 -->
```text
Opsiyonel bonus (eklediyseniz)

```

**Biz ne yaptık:** Restart/idempotency/maliyet bonusları README'de açıklanıyor; test ve gerçek/maliyet kapsam sınırları linkleniyor.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** README Hata, retry ve bonuslar ile Başlangıç tahmini ve gerçek maliyet bölümlerini oku.

**Değerlendirici nasıl inceleyebilir?** Üç bonusun implementation/test karşılığını bu rapordaki ayrı maddelerle eşleştir. Fal actual accounting sınırı, case'in tahmini maliyet bonusunu engellemiyor.

## 84. Bütçe: sağlayıcı ücretinin karşılanması

**Orijinal metin — kaynak satır 191–192:**

<!-- case-source: start=191 end=192 -->
```text
Üretim Ortamı ve Bütçe
Sağlayıcı maliyetleri tarafımızca karşılanmaktadır.
```

**Biz ne yaptık:** Sağlayıcı ücretinin kimin tarafından karşılandığı case'in organizasyon bilgisi. Repoda ödeme/kredi sistemi kurulmadı; maliyet kayıtları USD cinsinden tutuluyor.

**Yapıldı mı?** Bilgilendirme / kapsam açıklaması; ayrı bir geliştirme işi değil.

**Sen nasıl inceleyebilirsin?** README bütçe ve maliyet ayrımını incele.

**Değerlendirici nasıl inceleyebilir?** Bunu yeni ödeme entegrasyonu gereksinimi olarak değerlendirme; kredi/ödeme zaten kapsam dışında.

## 85. Bütçe: sağlanan API anahtarları

**Orijinal metin — kaynak satır 193–193:**

<!-- case-source: start=193 end=193 -->
```text
Size özel fal.ai ve OpenRouter API anahtarları e-posta ile iletilecektir.
```

**Biz ne yaptık:** Kullanıcı sağladığı anahtarları ignored .env'ye girdi; sunucuya özel env olarak taşındı. Frontend/Git/loglarda plaintext yok. Anahtarın e-posta ile iletilmesi dış süreç; mail erişimi incelenmedi.

**Yapıldı mı?** Anahtar yapılandırması tamamlandı; e-posta teslimi uygulama kapsamı dışında.

**Sen nasıl inceleyebilirsin?** .env.example'de değişken isimlerini gör; gerçek .env veya secret değerlerini ekran/rapor/Git ile paylaşma.

**Değerlendirici nasıl inceleyebilir?** Secret değerlerini okumadan .gitignore, backend config, server env mode600 ve frontend bundle/config ayrımını incele; deployment proof keys_configured:true gösterir.

## 86. Bütçe: 5/2 USD limit ve mock testler

**Orijinal metin — kaynak satır 194–195:**

<!-- case-source: start=194 end=195 -->
```text
Anahtarların aday başına üst limiti: fal.ai 5 USD, OpenRouter 2 USD. Gereksiz istek göndermemeye dikkat
edin; testlerde mock sağlayıcıyı kullanın.
```

**Biz ne yaptık:** Kabul araçları fal $5/OpenRouter $2 hard limitleri ve $0.50/$0.05 inceleme eşiklerini gözetiyor; az ve sırayla onaylı gerçek çağrılar kullanıldı. Test/startup gerçek üretim çağırmıyor. Dört eski gerçek zincirde OpenRouter bilinen actual toplam $0.0001962; fal tahmini toplam $0.106. Fal actual/bakiye billing yetkisiyle doğrulanmadı.

**Yapıldı mı?** Mock ve planlı bütçe yönetimi tamamlandı; fal gerçek bakiye/tüketim doğrulaması kullanıcı talebiyle kapsam dışı.

**Sen nasıl inceleyebilirsin?** flowkit-approved.json final_budget_summary alanını oku; actual ile estimated değerleri ayrı işaretle. Bu rapor üretim yapmıyor ve anahtarlı provider çağrısı kullanmıyor.

**Değerlendirici nasıl inceleyebilir?** cmd/acceptance bütçe rezervasyon kontrolleri, approved demo kapsamı ve CI empty-key/mocks yapılandırmasını incele. Tahmin eşiğinin altında olmak, yetkisiz provider bakiyesinin doğrulandığı anlamına gelmez.

## 87. Bütçe: limit aşımı ve kontrol sorumluluğu

**Orijinal metin — kaynak satır 196–196:**

<!-- case-source: start=196 end=196 -->
```text
Limiti aşan anahtarlar kapatılır. Bütçe yönetimi değerlendirmenin parçasıdır.
```

**Biz ne yaptık:** Ek üretim yalnız açık kapsamlı kullanıcı onayıyla yapıldı; call/budget ledger ve reserved retry hesabı kabul aracında var. Bilinmeyen ücret sıfır yazılmadı; yetkisiz fal accounting için ek sorgu yapılmadı.

**Yapıldı mı?** Bütçe yönetimi kaydı var; provider tarafından kapatma dış politika, gerçek fal toplamı doğrulanmış değil.

**Sen nasıl inceleyebilirsin?** README bütçe paragrafı ve final_budget_summary içinde çağrı sayısı, review threshold/hard limit ve balances_verified:false alanlarını gör.

**Değerlendirici nasıl inceleyebilir?** Dört üretim/20 eski onaylı anahtarlı HTTP çağrısı özetini incele. Son düzeltme/deploy turunda provider_calls:0 olduğunu doğrula; limits tüketim kanıtını tahminden türetme.

## 88. Bütçe: model gerekçesi ve ucuz çözünürlük

**Orijinal metin — kaynak satır 197–199:**

<!-- case-source: start=197 end=199 -->
```text
Not: Çıktı kalitesi değerlendirilmediği için en ucuz modelleri kullanabilirsiniz (ör. fal'da hızlı/düşük
maliyetli görsel modelleri ve düşük çözünürlük, OpenRouter'da düşük maliyetli metin modelleri). Model
seçimi size aittir; tercihinizin gerekçesini README'de kısaca belirtin.
```

**Biz ne yaptık:** OpenRouter Luna kısa prompt için düşük maliyet; Schnell ucuz/hızlı, Dev aynı profil ile runtime hot-add kanıtı için seçildi. Ürün 512×512, afiş 512×768, tek görsel; kalite iyileştirme üretimi yapılmadı. Model gerekçesi README'de.

**Yapıldı mı?** Tamamlandı.

**Sen nasıl inceleyebilirsin?** README model seçimi paragrafını ve models/workflows parametrelerini incele.

**Değerlendirici nasıl inceleyebilir?** Schnell/Dev defaults, num_images=1, image_size, LLM max_completion_tokens=300 ve kanıttaki gerçek parametreleri karşılaştır. Fiyatlar tarihli katalog snapshot'ıdır; rapor güncel fiyat keşfi yaptığını iddia etmez.

## E1. Siteye giriş ve iki workflow

1. [Siteyi](https://flowkit.canhurkurtoglu.com/) gizli pencerede aç. Şifre ekranı görünmeli. Şifre değerlendiriciye sahibinden özel kanalla verilir.
2. Girişten sonra sol listede **Ürün katalog görseli** ve **Etkinlik afişi** görünmeli.
3. Her birinin JSON tanımını aç. Ürün image `modelRef:flux-schnell`, etkinlik image `modelRef:flux-dev`; dört node/üç edge olmalı.
4. Son çalıştırmalar bölümünden mevcut başarılı run'ları aç. Girdi, prompt, model, süre, deneme ve maliyet sütunlarına bak; görsel bağlantısını aç.
5. JSON editöründe bir taslakta `modelRef:not-registered` ve bilinmeyen şablon/eksik edge oluşturup **Doğrula** kullan. Çoklu bulgu bekle. Teslim workflow'unu değiştirmeden taslağı bırakıp sayfayı yenile.
6. Çıkış yap; korumalı API'ye tekrar erişim giriş gerektirmeli. Mevcut run incelemesi yeni üretim yapmaz.

## E2. API ve gerçek tarihsel kanıtlar

Girişten sonra aynı tarayıcıda bu adresler açılabilir:

| Kontrol | Adres / dosya | Beklenen |
|---|---|---|
| Katalog | [/models](https://flowkit.canhurkurtoglu.com/models) | prompt-writer, flux-schnell, flux-dev; enabled:true |
| Workflow listesi | [/workflows](https://flowkit.canhurkurtoglu.com/workflows) | 2 aktif workflow; ürün revision 3/Schnell, etkinlik revision 1/Dev |
| Run geçmişi | [/runs](https://flowkit.canhurkurtoglu.com/runs) | Denetim snapshot'ında 3 gerçek başarılı run + 1 ücretsiz kontrol run'ı |
| İlk FlowKit Schnell | [/runs/a733fb1e-686e-4d78-8386-09ce5953a36a](https://flowkit.canhurkurtoglu.com/runs/a733fb1e-686e-4d78-8386-09ce5953a36a) | succeeded, Schnell, ilk immutable workflow/model snapshot |
| API'den model değişimi Dev | [/runs/217fb931-3af8-4313-89b2-436985798449](https://flowkit.canhurkurtoglu.com/runs/217fb931-3af8-4313-89b2-436985798449) | succeeded, ürün revision 2, Dev |
| İkinci workflow Dev | [/runs/614b472e-e6c8-4e17-b07d-995f4307b321](https://flowkit.canhurkurtoglu.com/runs/614b472e-e6c8-4e17-b07d-995f4307b321) | succeeded, etkinlik revision 1, Dev |
| Ücretsiz cloud worker | [/runs/edbca753-ebe1-4e96-aef3-90412a1e5f79](https://flowkit.canhurkurtoglu.com/runs/edbca753-ebe1-4e96-aef3-90412a1e5f79) | message/approved:false, input→output, provider request 0 |
| Billing ayarı | [/capabilities](https://flowkit.canhurkurtoglu.com/capabilities) | billing_enabled:false |
| Sağlık | [/health/ready](https://flowkit.canhurkurtoglu.com/health/ready) | status:ready |
| Doğrudan deney | [direct.json](evidence/direct.json) | gerçek OpenRouter completion + Schnell queue, ayrı standalone deney |
| Restart olmadan runtime kabul | [flowkit-approved.json](evidence/flowkit-approved.json) | HTTP mutation gövdeleri, runtime_before/runtime_after_all aynı kimlikler |
| Cloud taşıma/kontrol | [deployment-fixes.json](evidence/deployment-fixes.json) | 3 run objesi orijinaliyle eşit, 3 artifact SHA256, worker/log/auth kontrolleri |

Sonradan yeni run/model oluşturursan sayılar değişebilir; tablodaki sayılar **denetim anının kaydıdır**. Run snapshot'larında gerçek model bilgisi korunmalıdır. Public site JSON'u API'dir; rapordaki kanıt JSON'u önceki kaydedilmiş kabul verisidir. Şifreli API için cURL kullanacaksan gate oturum cookie'si gerekir; bunu rapora/Git'e koyma. Yeni model/workflow mutasyonunu yerel test ortamında denemek, paylaşılan teslim verisini korur.

## E3. Maliyetsiz otomatik test incelemesi

[GitHub Actions](https://github.com/CanhK-2103/trypix-flowkitCase/actions) en kolay yoldur: backend, dashboard ve compose işleri yeşil olmalı. Backend integration testleri PostgreSQL ile çalışmalı; skip olmamalı. Aynı kontrolleri yerelde çalıştırmak için **ayrı test veritabanı** kullanılır; testler bu DB'yi temizler, production DB URL'si kullanılmaz.

Repository kökünde PowerShell:

```powershell
docker compose -p flowkit-tests -f compose.test.yaml up -d --wait
$env:TEST_DATABASE_URL='postgres://flowkit:flowkit@localhost:55432/flowkit_test?sslmode=disable'
go test ./... -count=1 -timeout=6m
go vet ./...
sqlc generate
git diff --exit-code -- internal/storage/dbgen
node tools/generate-openapi.mjs
git diff --exit-code -- docs/openapi.json
```

Frontend:

```powershell
Set-Location web
npm ci
npm test
npm run build
```

Beklenen: Go/integration başarılı; frontend 4 dosyada 13 test başarılı; typecheck/build başarılı; generated SQL/OpenAPI farkı yok. Bu rapor oluşturulurken uygulama kodu değiştirilmediği için tam testler tekrar başlatılmadı; yukarıdaki sonuçlar son uygulama CI ve denetim kayıtlarına aittir.

Belirli bonus/kuralları tek başına incelemek için test DB hazırken repository kökünde:

```powershell
go test ./internal/workflow -run 'TestRejectGraphRules|TestIndependentFindings' -count=1 -v
go test ./internal/engine -run 'TestTransientRetryThenSuccess|TestRetryCap|TestPermanentFailureCorrectNode|TestResumeKnownFalRequest' -count=1 -v
go test ./internal/integration -run 'TestKilledWorkerResumesDurableFalTicket|TestConcurrentIdempotencyAndSnapshot|TestHotAddModelAndWorkflowWithoutRestart' -count=1 -timeout=6m -v
go test ./internal/integration -run 'TestQuoteAndDelayedBillingPersistedWithAPIScheduledRefresh' -count=1 -timeout=6m -v
```

Temiz kurulum için ayrı checkout/.env.example ile `docker compose up` kullanılır. Sadece arayüz/kurulum incelemek için gerçek anahtar gerekmez. Mevcut gerçek anahtarlarla normal worker'ı başlatmak bekleyen provider işlerini otomatik yürütür; gerçek kullanım bu projedeki kullanıcı onay sınırlarına tabidir. `go run ./cmd/direct`, `cmd/acceptance` veya live-demo script'i bu raporun ücretsiz kontrol komutları değildir.

## E4. Sunucuda salt okunur servis, log ve kayıt kontrolü

Sahibi sunucuda şu komutlarla kurulum durumunu inceleyebilir; değerlendirici için zorunlu SSH paylaşımı yoktur:

```sh
cd /opt/flowkit
git rev-parse HEAD
git status --short
docker compose -p flowkit -f compose.yaml -f compose.production.yaml ps -a
docker compose -p flowkit -f compose.yaml -f compose.production.yaml logs --no-color --no-log-prefix --tail=20 api worker db dashboard gate proxy init bootstrap
docker compose -p flowkit -f compose.yaml -f compose.production.yaml exec -T db psql -U flowkit -d flowkit -c "SELECT count(*) AS runs FROM runs; SELECT count(*) AS artifacts FROM artifacts; SELECT count(*) AS provider_requests FROM provider_requests;"
```

Beklenen: API/DB/gate healthy; dashboard/proxy/worker running; init/bootstrap exited(0). Çıktının log satırları JSON ve run_id/node_id/attempt içerir. DB sayıları son kontrol anında 4/3/6; providerless test yeni provider request eklemez. Yalnız proxy 80/443 host portları açık; DB/API host'a publish edilmez. `.env` private/mode600; secret içeriğini yazdırmaya gerek yoktur.

Git HEAD yeni dokümantasyon commit'leriyle ilerleyebilir. [deployment-fixes.json](evidence/deployment-fixes.json) içindeki `code_revision` çalışan uygulamanın kod build'ini gösterir; rapor veya README eklemek uygulama image'ını yeniden build etmeyi gerektirmez. SQL ve log okumaları canlı AI sağlayıcı kullanımı değildir.

## E5. Kod hâkimiyetini sınamak için kısa savunma soruları

| Soru | Açıklayabilmen gereken nokta | Kontrol dosyası |
|---|---|---|
| Run DB'ye yazılıp job kaydı başarısız olursa ne olur? | Run/snapshot/InsertTx tek transaction; rollback yarım kayıt bırakmaz. | [server.go](../internal/api/server.go), transaction rollback testi |
| Yeni Dev'i neden deploy etmeden ekleyebiliyoruz? | Alias→catalog modelId/profile/defaults veri; motor/provider adaptörü aynı. | [validate.go](../internal/workflow/validate.go), provider client, hot-add testi |
| Fal ID'si ne zaman kalıcı olur? | Submit cevabı sonrası, poll/pricing/download ve worker return öncesi DB commit. | [engine.go](../internal/engine/engine.go), killed-worker testi |
| Provider isteği kabul etti ama cevap DB'ye yazılamadıysa? | sending + durable ID yoksa uncertain submission; kör submit tekrarına izin verilmez. | engine sending fazı, TestUncertainSubmissionIsNotRetried |
| Geçici poll hatası yeni image attempt/charge oluşturur mu? | Aynı ticket ile güvenli polling; domain üretim retry'si ve queue recovery ayrı. | TestPollRetryKeepsRequestAndCost, jobs.go |
| Aynı key iki eşzamanlı POST'ta nasıl tek run olur? | Request hash/unique key/transaction; aynı gövde replay, farklı gövde 409. | TestConcurrentIdempotencyAndSnapshot |
| Fal tahmini belli iken neden actual total null? | Tahmin actual değil; tüm request-specific ücretler bilinmeden gerçek total hesaplanmaz. | [cost.go](../internal/engine/cost.go), [billing_test.go](../internal/engine/billing_test.go) |
| Sonradan workflow/model değişirse eski run neyi gösterir? | Immutable workflow/model/initial-estimate snapshot ve gerçek producer provenance. | startRun snapshot, run types, gerçek eski Dev run |
| Şifre koruması neden kullanıcı sistemi sayılmıyor? | Shared deployment gate case'in opsiyonel yayına alma koruması; role/account yönetimi yok. | [gate.go](../internal/gate/gate.go), production override |

Bu cevapları ezberlemek yerine ilgili kodu adım adım gösterebilmen gerekir. Görüşme olmadan bu kişisel şartın tamamlandığı iddia edilemez.
