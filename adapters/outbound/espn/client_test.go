package espn

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClientFetchDraftWithHTTPTestServer(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "draft_payload.json"))
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture)
	}))
	defer ts.Close()
	client := NewClient(ts.URL, &http.Client{Timeout: 2 * time.Second}, 1, time.Millisecond, slog.New(slog.NewTextHandler(io.Discard, nil)))
	snap, err := client.FetchDraft(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Picks) == 0 {
		t.Fatal("expected picks")
	}
}
