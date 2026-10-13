package runner

import (
	"context"
	"testing"
	"time"
)

func TestRealClockNowAndSleep(t *testing.T) {
	c := RealClock{}
	if c.Now().IsZero() {
		t.Fatal("now should not be zero")
	}
	if err := c.Sleep(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Sleep(ctx, time.Second); err == nil {
		t.Fatal("expected canceled")
	}
}
