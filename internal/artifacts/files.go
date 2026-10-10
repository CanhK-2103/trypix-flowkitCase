package artifacts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CanhK-2103/trypix-flowkitCase/internal/domain"
)

const MaxSize = 20 * 1024 * 1024

type Files struct {
	Root      string
	HTTP      *http.Client
	AllowHTTP bool
}

func New(root string) *Files {
	return &Files{Root: root, HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (f *Files) Download(ctx context.Context, source string, a domain.Artifact) (domain.Artifact, error) {
	u, err := url.Parse(source)
	if err != nil || u.User != nil || (u.Scheme != "https" && !(f.AllowHTTP && u.Scheme == "http")) {
		return a, fmt.Errorf("invalid artifact source URL")
	}
	if strings.ContainsAny(a.ID, "/\\.") {
		return a, fmt.Errorf("invalid artifact ID")
	}
	if err := os.MkdirAll(f.Root, 0750); err != nil {
		return a, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
	if err != nil {
		return a, err
	}
	res, err := f.HTTP.Do(req)
	if err != nil {
		return a, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return a, fmt.Errorf("artifact download HTTP %d", res.StatusCode)
	}
	mime := strings.Split(res.Header.Get("Content-Type"), ";")[0]
	if mime != "image/png" && mime != "image/jpeg" && mime != "image/webp" {
		return a, fmt.Errorf("unsupported artifact MIME: %s", mime)
	}
	temp, err := os.CreateTemp(f.Root, "download-*")
	if err != nil {
		return a, err
	}
	defer os.Remove(temp.Name())
	n, copyErr := io.Copy(temp, io.LimitReader(res.Body, MaxSize+1))
	closeErr := temp.Close()
	if copyErr != nil {
		return a, copyErr
	}
	if closeErr != nil {
		return a, closeErr
	}
	if n == 0 || n > MaxSize {
		return a, fmt.Errorf("artifact exceeds 20 MiB or is empty")
	}
	name := a.ID + ".image"
	if err := os.Rename(temp.Name(), filepath.Join(f.Root, name)); err != nil {
		return a, err
	}
	a.FileName = name
	a.MIME = mime
	a.Size = n
	return a, nil
}
