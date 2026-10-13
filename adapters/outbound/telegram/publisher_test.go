package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type rewriteTransport struct {
	target string
	base   http.RoundTripper
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(r.target, "http://")
	return r.base.RoundTrip(req)
}

func TestPublisherUsesHTTPFakeServer(t *testing.T) {
	hits := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hits++
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", req.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	httpClient := &http.Client{Transport: rewriteTransport{target: ts.URL, base: http.DefaultTransport}, Timeout: 2 * time.Second}
	pub := NewPublisher("token", "chat", httpClient, 0, time.Millisecond)
	if err := pub.Publish(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("expected one call, got %d", hits)
	}
}
