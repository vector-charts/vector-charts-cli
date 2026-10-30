package output

import (
	"strings"
	"testing"
	"time"
)

func TestTimePastAndFuture(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-90 * time.Minute).Format(time.RFC3339)
	got := Time(past)
	if !strings.HasSuffix(got, " ago") || !strings.Contains(got, "1h") {
		t.Fatalf("past: got %q", got)
	}

	future := now.Add(3 * 24 * time.Hour).Add(time.Hour).UnixMilli()
	got = Time(float64(future))
	if !strings.HasPrefix(got, "in ") || !strings.Contains(got, "3d") {
		t.Fatalf("future: got %q", got)
	}

	if Time(nil) != "" || Time("") != "" || Time(float64(0)) != "" {
		t.Fatal("expected empty for nil/zero")
	}
}

func TestEpochMSStillDate(t *testing.T) {
	ts := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	got := EpochMS(ts.Format(time.RFC3339))
	if got != "2026-03-15" {
		t.Fatalf("got %q", got)
	}
}

func TestEstimateRemaining(t *testing.T) {
	started := time.Now().Add(-10 * time.Minute).UnixMilli()
	got := EstimateRemaining(50, float64(started))
	if !strings.Contains(got, "remaining") || !strings.Contains(got, "10m") {
		t.Fatalf("got %q", got)
	}
	if EstimateRemaining(0, float64(started)) != "" || EstimateRemaining(100, float64(started)) != "" {
		t.Fatal("expected empty for 0%/100%")
	}
}
