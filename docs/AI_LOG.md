# AI çalışma kaydı

## 10 Ekim 2026 - Başlangıç

- Codex case PDF'sinin beş sayfasını metin ve görsel olarak inceledi; kabul koşullarını docs/ACCEPTANCE.md dosyasına aktardı.
- Kullanıcı Go ile ilerlemeyi onayladı, API anahtarlarının ulaştığını ve teslim tarihinin 12 Ekim olduğunu belirtti.
- Doğrudan sağlayıcı denemesi için resmi OpenRouter chat completions ve fal queue/model dokümanları incelendi.
- Yerelde Go PATH üzerinde bulunamadı; Go 1.27.2 resmi arşivden SHA-256 kontrolüyle proje içindeki ignored .tools dizinine kuruldu. Docker istemcisi var, daemon çalışmıyor.
- PDF incelemesinde ilk denenen pymupdf paketi ortamda yoktu. pypdf metin çıkarımı ve Poppler görsel render kullanıldı. Windows metin encoding hatası UTF-8 ile giderildi.
- Henüz gerçek sağlayıcı denemesi yapılmadı. Çalışma zamanı anahtarları boş.
- İlk derlemede .env okuyucusundaki bir rune literal'inin gereksiz çift tırnak kaçışı Go derleyicisi tarafından reddedildi. Kaçış kaldırıldı; ardından go test ./... ve go vet ./... başarılı oldu.
- Yerel HTTP fixture testleri: tam sağlayıcı sırası ve auth, dönen queue URL'lerini takip, terminal hata, bilinmeyen durum, HTTP 503'te doğrudan araçta yeniden submit yapılmaması, model/boş/kesilmiş LLM yanıtı ve güvenilmeyen URL reddi. Disk testi: fal ID'nin kalıcı güncellenmesi, secret redaksiyonu ve geçici dosya temizliği.
- Payload önizlemesi -dry-run ile başarılı; sağlayıcı ağ çağrısı sayısı sıfır. Bu testler henüz workflow motorunun kabul testleri değildir.

Kodun anlaşılması ve tasarım kararlarının aday tarafından gözden geçirilmesi ilerleyen aşamalarda ayrıca kaydedilecek. Mimari hatalar fark edilirse gerçek olaylar olarak eklenecek.
