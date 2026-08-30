package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNeedsUpdate(t *testing.T) {
	if !NeedsUpdate("dev", "v1.0.0") {
		t.Fatal("dev should update")
	}
	if NeedsUpdate("v1.0.0", "v1.0.0") {
		t.Fatal("same version")
	}
	if !NeedsUpdate("1.0.0", "v1.1.0") {
		t.Fatal("should update")
	}
	if NeedsUpdate("v1.0.0", "") {
		t.Fatal("empty latest")
	}
}

func TestParseSHA256SUMS(t *testing.T) {
	got := ParseSHA256SUMS("abc123  gamayun-linux-amd64\ndef  ./gamayun-linux-arm64\n")
	if got["gamayun-linux-amd64"] != "abc123" {
		t.Fatalf("%v", got)
	}
	if got["gamayun-linux-arm64"] != "def" {
		t.Fatalf("%v", got)
	}
}

func TestClientUpdate(t *testing.T) {
	payload := []byte("new-binary")
	sum := sha256.Sum256(payload)
	hexSum := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/repos/acme/gamayun/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","assets":[` +
			`{"name":"gamayun-linux-amd64","browser_download_url":"` + srv.URL + `/gamayun-linux-amd64"},` +
			`{"name":"SHA256SUMS","browser_download_url":"` + srv.URL + `/SHA256SUMS"}]}`))
	})
	mux.HandleFunc("/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hexSum + "  gamayun-linux-amd64\n"))
	})
	mux.HandleFunc("/gamayun-linux-amd64", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})

	dir := t.TempDir()
	dest := filepath.Join(dir, "gamayun")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	restarted := false
	c := &Client{
		HTTP:    srv.Client(),
		API:     srv.URL,
		Repo:    "acme/gamayun",
		Current: "v1.0.0",
		Dest:    dest,
		GOARCH:  "amd64",
		Restart: func() error { restarted = true; return nil },
	}
	if err := c.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-binary" {
		t.Fatalf("got %q", got)
	}
	if !restarted {
		t.Fatal("expected restart")
	}
}

func TestClientAlreadyLatest(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	mux.HandleFunc("/repos/acme/gamayun/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0","assets":[]}`))
	})
	c := &Client{
		HTTP:    srv.Client(),
		API:     srv.URL,
		Repo:    "acme/gamayun",
		Current: "v1.0.0",
		Dest:    filepath.Join(t.TempDir(), "bin"),
		GOARCH:  "amd64",
	}
	if err := c.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestClientMissingRepo(t *testing.T) {
	c := &Client{}
	if err := c.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "repo unknown") {
		t.Fatalf("got %v", err)
	}
}

func TestClientUpdateFailsWithoutChecksumsAsset(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/repos/acme/gamayun/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","assets":[` +
			`{"name":"gamayun-linux-amd64","browser_download_url":"` + srv.URL + `/gamayun-linux-amd64"}]}`))
	})
	downloaded := false
	mux.HandleFunc("/gamayun-linux-amd64", func(w http.ResponseWriter, r *http.Request) {
		downloaded = true
		_, _ = w.Write([]byte("new-binary"))
	})

	dir := t.TempDir()
	dest := filepath.Join(dir, "gamayun")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &Client{
		HTTP:    srv.Client(),
		API:     srv.URL,
		Repo:    "acme/gamayun",
		Current: "v1.0.0",
		Dest:    dest,
		GOARCH:  "amd64",
		Restart: func() error { t.Fatal("must not restart"); return nil },
	}
	err := c.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no SHA256SUMS asset") {
		t.Fatalf("got %v", err)
	}
	if downloaded {
		t.Fatal("must not download binary without SHA256SUMS")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("dest changed: %q", got)
	}
}

func TestClientUpdateFailsWithMissingChecksumEntry(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/repos/acme/gamayun/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","assets":[` +
			`{"name":"gamayun-linux-amd64","browser_download_url":"` + srv.URL + `/gamayun-linux-amd64"},` +
			`{"name":"SHA256SUMS","browser_download_url":"` + srv.URL + `/SHA256SUMS"}]}`))
	})
	mux.HandleFunc("/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("deadbeef  gamayun-linux-arm64\n"))
	})
	mux.HandleFunc("/gamayun-linux-amd64", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("new-binary"))
	})

	dir := t.TempDir()
	dest := filepath.Join(dir, "gamayun")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &Client{
		HTTP:    srv.Client(),
		API:     srv.URL,
		Repo:    "acme/gamayun",
		Current: "v1.0.0",
		Dest:    dest,
		GOARCH:  "amd64",
		Restart: func() error { t.Fatal("must not restart"); return nil },
	}
	err := c.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no checksum for gamayun-linux-amd64") {
		t.Fatalf("got %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("dest changed: %q", got)
	}
}

func TestClientSendsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Current: "test"}
	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	body.Close()
	if gotAuth != "" {
		t.Fatalf("empty token sent Authorization %q", gotAuth)
	}

	c.Token = "ghs_test"
	body, err = c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	body.Close()
	if gotAuth != "Bearer ghs_test" {
		t.Fatalf("got %q", gotAuth)
	}
}
