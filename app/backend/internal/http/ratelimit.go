package http

import (
	"net"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     float64
	capacity float64
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

func newRateLimiter(rate float64) *rateLimiter {
	if rate <= 0 {
		return nil
	}
	capacity := rate * 2
	if capacity < 5 {
		capacity = 5
	}
	return &rateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     rate,
		capacity: capacity,
	}
}

func (l *rateLimiter) Allow(key string, now time.Time) bool {
	if l == nil {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &tokenBucket{
			tokens:     l.capacity - 1,
			lastRefill: now,
		}
		l.buckets[key] = bucket
		return true
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	if elapsed > 0 {
		bucket.tokens += elapsed * l.rate
		if bucket.tokens > l.capacity {
			bucket.tokens = l.capacity
		}
		bucket.lastRefill = now
	}

	if bucket.tokens < 1 {
		return false
	}

	bucket.tokens -= 1
	return true
}

func clientIPAddress(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
