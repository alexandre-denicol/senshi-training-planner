package auth

import (
	"sync"
	"time"
)

const (
	loginLimitWindow     = 15 * time.Minute
	loginLimitLock       = 15 * time.Minute
	loginLimitMax        = 5
	loginLimitEntriesMax = 4096
)

type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
	now      func() time.Time
}

type loginAttempt struct {
	count       int
	windowEnds  time.Time
	lockedUntil time.Time
}

func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{
		attempts: map[string]loginAttempt{},
		now:      time.Now,
	}
}

func (l *LoginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.cleanupLocked(now)

	attempt := l.attempts[key]
	if !attempt.lockedUntil.IsZero() && now.Before(attempt.lockedUntil) {
		return false
	}

	if now.After(attempt.windowEnds) {
		delete(l.attempts, key)
	}

	return true
}

func (l *LoginLimiter) RecordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.cleanupLocked(now)

	attempt := l.attempts[key]
	if now.After(attempt.windowEnds) {
		attempt = loginAttempt{windowEnds: now.Add(loginLimitWindow)}
	}

	attempt.count++
	if attempt.count >= loginLimitMax {
		attempt.lockedUntil = now.Add(loginLimitLock)
	}
	l.attempts[key] = attempt
	l.enforceBoundLocked()
}

func (l *LoginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, key)
}

func (l *LoginLimiter) cleanupLocked(now time.Time) {
	for key, attempt := range l.attempts {
		if attempt.expired(now) {
			delete(l.attempts, key)
		}
	}
}

func (l *LoginLimiter) enforceBoundLocked() {
	for len(l.attempts) > loginLimitEntriesMax {
		var oldestKey string
		var oldestTime time.Time
		for key, attempt := range l.attempts {
			expiresAt := attempt.expiresAt()
			if oldestKey == "" || expiresAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = expiresAt
			}
		}
		delete(l.attempts, oldestKey)
	}
}

func (a loginAttempt) expired(now time.Time) bool {
	return now.After(a.expiresAt())
}

func (a loginAttempt) expiresAt() time.Time {
	if a.lockedUntil.After(a.windowEnds) {
		return a.lockedUntil
	}
	return a.windowEnds
}
