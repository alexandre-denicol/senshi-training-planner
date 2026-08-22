package auth

import (
	"strconv"
	"testing"
	"time"
)

func TestLoginLimiterCleansExpiredEntries(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	limiter := NewLoginLimiter()
	limiter.now = func() time.Time { return now }

	limiter.RecordFailure("old")
	if len(limiter.attempts) != 1 {
		t.Fatalf("expected one attempt, got %d", len(limiter.attempts))
	}

	now = now.Add(loginLimitWindow + time.Second)
	limiter.RecordFailure("new")

	if _, ok := limiter.attempts["old"]; ok {
		t.Fatal("expected expired entry to be cleaned by later activity")
	}
	if _, ok := limiter.attempts["new"]; !ok {
		t.Fatal("expected new entry to remain")
	}
}

func TestLoginLimiterDefensiveBound(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	limiter := NewLoginLimiter()
	limiter.now = func() time.Time { return now }

	for i := 0; i < loginLimitEntriesMax+50; i++ {
		limiter.RecordFailure("key-" + strconv.Itoa(i))
	}

	if len(limiter.attempts) > loginLimitEntriesMax {
		t.Fatalf("expected attempts map to stay bounded at %d, got %d", loginLimitEntriesMax, len(limiter.attempts))
	}
}

func TestLoginLimiterLockoutBehavior(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	limiter := NewLoginLimiter()
	limiter.now = func() time.Time { return now }

	for i := 0; i < loginLimitMax; i++ {
		limiter.RecordFailure("locked")
	}

	if limiter.Allow("locked") {
		t.Fatal("expected locked key to be denied")
	}

	now = now.Add(loginLimitLock + time.Second)
	if !limiter.Allow("locked") {
		t.Fatal("expected key to be allowed after lock expires")
	}
}
