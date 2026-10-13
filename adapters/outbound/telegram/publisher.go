package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxResponseBytes bounds how much of a Telegram response is read.
const maxResponseBytes = 1 << 20

type Publisher struct {
	token   string
	chatID  string
	http    *http.Client
	retries int
	backoff time.Duration
}

func NewPublisher(token, chatID string, httpClient *http.Client, retries int, backoff time.Duration) *Publisher {
	return &Publisher{token: token, chatID: chatID, http: httpClient, retries: retries, backoff: backoff}
}

// Publish sends message to the configured chat. Transport errors and 429/5xx
// responses are retried; other 4xx responses (bad token, unknown chat, ...)
// will not succeed on retry and fail immediately.
func (p *Publisher) Publish(ctx context.Context, message string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", p.token)
	payload := map[string]string{"chat_id": p.chatID, "text": message}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var lastErr error
	for attempt := 0; attempt <= p.retries; attempt++ {
		retry, err := p.send(ctx, endpoint, buf)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retry || attempt == p.retries {
			break
		}
		if err := sleep(ctx, p.backoff); err != nil {
			return err
		}
	}
	return lastErr
}

func (p *Publisher) send(ctx context.Context, endpoint string, body []byte) (retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, p.redact(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.http.Do(req)
	if err != nil {
		return ctx.Err() == nil, p.redact(err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	retry = resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
	return retry, p.redact(fmt.Errorf("telegram returned status %d: %s", resp.StatusCode, respBody))
}

// redact removes the bot token from err. net/http errors embed the full request
// URL, which for the Telegram API contains the token.
func (p *Publisher) redact(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	if p.token == "" {
		return err
	}
	if msg := err.Error(); strings.Contains(msg, p.token) {
		return errors.New(strings.ReplaceAll(msg, p.token, "***"))
	}
	return err
}

// sleep waits for d or until ctx is cancelled.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
