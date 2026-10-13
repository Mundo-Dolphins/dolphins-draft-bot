package espn

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

// maxResponseBytes bounds how much of an ESPN response is read.
const maxResponseBytes = 8 << 20

type Client struct {
	url     string
	http    *http.Client
	retries int
	backoff time.Duration
	parser  Parser
	logger  *slog.Logger
}

func NewClient(url string, httpClient *http.Client, retries int, backoff time.Duration, logger *slog.Logger) *Client {
	return &Client{url: url, http: httpClient, retries: retries, backoff: backoff, parser: NewParser(), logger: logger}
}

func (c *Client) FetchDraft(ctx context.Context) (domain.DraftSnapshot, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		c.logger.Debug("fetching espn draft", "url", c.url, "attempt", attempt+1, "max_attempts", c.retries+1)
		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
		if errReq != nil {
			return domain.DraftSnapshot{}, errReq
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			c.logger.Warn("espn request failed", "attempt", attempt+1, "error", err)
		} else {
			body, errRead := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
			_ = resp.Body.Close()
			if errRead != nil {
				lastErr = errRead
				c.logger.Warn("failed reading espn response body", "attempt", attempt+1, "error", errRead)
			} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				lastErr = fmt.Errorf("espn returned status %d", resp.StatusCode)
				c.logger.Warn("espn returned non-success status", "attempt", attempt+1, "status", resp.StatusCode, "body_bytes", len(body))
			} else {
				c.logger.Debug("espn response received", "attempt", attempt+1, "status", resp.StatusCode, "body_bytes", len(body))
				snap, kind, errParse := c.parser.Parse(body)
				if errParse == nil {
					c.logger.Info("espn payload parsed", "kind", string(kind), "picks", len(snap.Picks))
					return snap, nil
				}
				lastErr = errParse
				c.logger.Warn("failed to parse espn payload", "attempt", attempt+1, "error", errParse)
			}
		}
		if attempt < c.retries {
			c.logger.Debug("waiting before espn retry", "backoff", c.backoff.String())
			if err := sleep(ctx, c.backoff); err != nil {
				return domain.DraftSnapshot{}, err
			}
		}
	}
	c.logger.Error("espn fetch failed after retries", "attempts", c.retries+1, "error", lastErr)
	return domain.DraftSnapshot{}, lastErr
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
