package espn

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientRetriesOnStatusError(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()
	c := NewClient(ts.URL, &http.Client{Timeout: time.Second}, 2, time.Millisecond, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := c.FetchDraft(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if calls != 3 {
		t.Fatalf("expected retries, got %d", calls)
	}
}

func TestClientParserFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer ts.Close()
	c := NewClient(ts.URL, &http.Client{Timeout: time.Second}, 0, time.Millisecond, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := c.FetchDraft(context.Background()); err == nil {
		t.Fatal("expected parser error")
	}
}

func TestClientBackoffStopsOnContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer ts.Close()
	c := NewClient(ts.URL, &http.Client{Timeout: time.Second}, 5, time.Minute, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := c.FetchDraft(ctx); err == nil {
		t.Fatal("expected error")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("fetch did not stop when the context was cancelled")
	}
}
