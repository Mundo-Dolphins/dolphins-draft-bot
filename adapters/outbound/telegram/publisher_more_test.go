package telegram

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type failTransport struct{}

func (failTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network down")
}

func TestPublisherRetryFailure(t *testing.T) {
	client := &http.Client{Transport: failTransport{}}
	pub := NewPublisher("token", "chat", client, 1, time.Millisecond)
	if err := pub.Publish(context.Background(), "hello"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPublisherErrorDoesNotLeakToken(t *testing.T) {
	const token = "123456:SECRET-TOKEN-VALUE"
	client := &http.Client{Transport: failTransport{}}
	pub := NewPublisher(token, "-1001234567890", client, 0, time.Millisecond)
	err := pub.Publish(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "SECRET-TOKEN-VALUE") {
		t.Fatalf("error leaks the bot token: %v", err)
	}
	if !strings.Contains(err.Error(), "network down") {
		t.Fatalf("error lost its cause: %v", err)
	}
}

func TestPublisherRetryPolicyByStatus(t *testing.T) {
	cases := []struct {
		status   int
		wantHits int
	}{
		{http.StatusBadRequest, 1},
		{http.StatusForbidden, 1},
		{http.StatusTooManyRequests, 3},
		{http.StatusInternalServerError, 3},
	}
	for _, tc := range cases {
		hits := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits++
			w.WriteHeader(tc.status)
		}))
		client := &http.Client{Transport: rewriteTransport{target: ts.URL, base: http.DefaultTransport}}
		pub := NewPublisher("token", "chat", client, 2, time.Millisecond)
		if err := pub.Publish(context.Background(), "hello"); err == nil {
			t.Fatalf("status %d: expected error", tc.status)
		}
		ts.Close()
		if hits != tc.wantHits {
			t.Fatalf("status %d: expected %d attempts, got %d", tc.status, tc.wantHits, hits)
		}
	}
}

func TestPublisherBackoffStopsOnContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	client := &http.Client{Transport: rewriteTransport{target: ts.URL, base: http.DefaultTransport}}
	pub := NewPublisher("token", "chat", client, 5, time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := pub.Publish(ctx, "hello"); err == nil {
		t.Fatal("expected error")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("publish did not stop when the context was cancelled")
	}
}
