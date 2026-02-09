package enrich

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRDNSOnlyForNewDevices(t *testing.T) {
	calls := 0
	lookup := func(context.Context, string) (string, error) {
		calls++
		return "fw.example.net", nil
	}
	now := time.Unix(1_700_000_000, 0).UTC()

	_, ok := MaybeLookup(Options{Enabled: false, IsNewDevice: true, MgmtIP: "10.0.0.1", Now: now, Lookup: lookup})
	if ok || calls != 0 {
		t.Fatalf("disabled lookup should not run: ok=%v calls=%d", ok, calls)
	}

	_, ok = MaybeLookup(Options{Enabled: true, IsNewDevice: false, MgmtIP: "10.0.0.1", Now: now, Lookup: lookup})
	if ok || calls != 0 {
		t.Fatalf("existing device lookup should not run: ok=%v calls=%d", ok, calls)
	}

	_, ok = MaybeLookup(Options{Enabled: true, IsNewDevice: true, MgmtIP: "host.example", Now: now, Lookup: lookup})
	if ok || calls != 0 {
		t.Fatalf("non-ip lookup should not run: ok=%v calls=%d", ok, calls)
	}

	got, ok := MaybeLookup(Options{Enabled: true, IsNewDevice: true, MgmtIP: "10.0.0.1", Now: now, Lookup: lookup})
	if !ok {
		t.Fatal("expected lookup to run")
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
	if got.Status != "ok" || got.PTRName != "fw.example.net" || got.IP != "10.0.0.1" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestRDNSTimeoutRetry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()

	t.Run("timeout then success retries once", func(t *testing.T) {
		calls := 0
		lookup := func(context.Context, string) (string, error) {
			calls++
			if calls == 1 {
				return "", context.DeadlineExceeded
			}
			return "fw.example.net", nil
		}
		got, ok := MaybeLookup(Options{Enabled: true, IsNewDevice: true, MgmtIP: "10.0.0.2", Now: now, Lookup: lookup})
		if !ok || calls != 2 {
			t.Fatalf("ok=%v calls=%d, want ok=true calls=2", ok, calls)
		}
		if got.Status != "ok" {
			t.Fatalf("status=%q, want ok", got.Status)
		}
	})

	t.Run("timeout twice returns timeout", func(t *testing.T) {
		calls := 0
		lookup := func(context.Context, string) (string, error) {
			calls++
			return "", context.DeadlineExceeded
		}
		got, ok := MaybeLookup(Options{Enabled: true, IsNewDevice: true, MgmtIP: "10.0.0.3", Now: now, Lookup: lookup})
		if !ok || calls != 2 {
			t.Fatalf("ok=%v calls=%d, want ok=true calls=2", ok, calls)
		}
		if got.Status != "timeout" {
			t.Fatalf("status=%q, want timeout", got.Status)
		}
	})

	t.Run("not found maps status", func(t *testing.T) {
		calls := 0
		lookup := func(context.Context, string) (string, error) {
			calls++
			return "", ErrNotFound
		}
		got, ok := MaybeLookup(Options{Enabled: true, IsNewDevice: true, MgmtIP: "10.0.0.4", Now: now, Lookup: lookup})
		if !ok || calls != 1 {
			t.Fatalf("ok=%v calls=%d, want ok=true calls=1", ok, calls)
		}
		if got.Status != "not_found" {
			t.Fatalf("status=%q, want not_found", got.Status)
		}
	})

	t.Run("generic error maps status", func(t *testing.T) {
		calls := 0
		lookup := func(context.Context, string) (string, error) {
			calls++
			return "", errors.New("resolver failure")
		}
		got, ok := MaybeLookup(Options{Enabled: true, IsNewDevice: true, MgmtIP: "10.0.0.5", Now: now, Lookup: lookup})
		if !ok || calls != 2 {
			t.Fatalf("ok=%v calls=%d, want ok=true calls=2", ok, calls)
		}
		if got.Status != "error" {
			t.Fatalf("status=%q, want error", got.Status)
		}
	})
}
