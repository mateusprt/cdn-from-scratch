package ratelimit

import (
	"sync"

	"golang.org/x/time/rate"
)

type Limiter struct {
	mutex             sync.Mutex
	limiters          map[string]*rate.Limiter
	requestsPerSecond float64
	burst             int
}

func New(requestsPerSecond float64, burst int) *Limiter {
	return &Limiter{
		limiters:          make(map[string]*rate.Limiter),
		requestsPerSecond: requestsPerSecond,
		burst:             burst,
	}
}

func (l *Limiter) Allow(key string) bool {
	limiter := l.getOrCreate(key)
	return limiter.Allow()
}

func (l *Limiter) getOrCreate(key string) *rate.Limiter {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	limiter, exists := l.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(l.requestsPerSecond), l.burst)
		l.limiters[key] = limiter
	}
	return limiter
}
