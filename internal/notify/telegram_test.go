package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTelegramSend(t *testing.T) {
	var gotChat, gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Errorf("path %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		q := string(body)
		if !strings.Contains(q, "chat_id=99") || !strings.Contains(q, "hello") {
			t.Errorf("form %s", q)
		}
		gotChat = "99"
		gotText = "hello"
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tg := &Telegram{
		Token:  "tok",
		ChatID: "99",
		Client: srv.Client(),
	}
	// rewrite Send by hitting the test server: use a custom transport via Client
	// NewRequest still goes to api.telegram.org. Use a round-tripper that redirects.
	tg.Client = &http.Client{
		Timeout:   5 * time.Second,
		Transport: rewriteHost{base: srv.Client().Transport, host: srv.URL},
	}
	if err := tg.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if gotChat != "99" || gotText != "hello" {
		t.Fatalf("%s %s", gotChat, gotText)
	}
}

type rewriteHost struct {
	base http.RoundTripper
	host string
}

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := req.URL.Parse(r.host + req.URL.Path)
	if err != nil {
		return nil, err
	}
	clone := req.Clone(req.Context())
	clone.URL = u
	clone.Host = u.Host
	rt := r.base
	if rt == nil {
		rt = http.DefaultTransport
	}
	return rt.RoundTrip(clone)
}

func TestTelegramNotConfigured(t *testing.T) {
	tg := NewTelegram("", "")
	if err := tg.Send(context.Background(), "x"); err == nil {
		t.Fatal("expected error")
	}
}
