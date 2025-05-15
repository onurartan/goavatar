package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Ratelimit config
const (
	requestsPerSecond = 15
	burstSize         = 35
	cleanupInterval   = 2 * time.Minute
	visitorTTL        = 5 * time.Minute
	blockDuration     = 10 * time.Second
)

type Visitor struct {
	limiter      *rate.Limiter
	lastSeen     time.Time
	blockedUntil time.Time
}

var (
	mu       sync.Mutex
	visitors = make(map[string]*Visitor)
)

func getRealIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func getVisitor(ip string) *Visitor {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burstSize)
		v = &Visitor{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		visitors[ip] = v
		return v
	}

	v.lastSeen = time.Now()
	return v
}

func cleanupVisitors() {
	for {
		time.Sleep(cleanupInterval)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > visitorTTL {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

// Rate Limiter Middleware
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getRealIP(r)
		visitor := getVisitor(ip)

		now := time.Now()

		mu.Lock()
		if now.Before(visitor.blockedUntil) {
			waitSeconds := int(visitor.blockedUntil.Sub(now).Seconds()) + 1
			msg := "To many requests. Try Again " + fmt.Sprintf("%d seconds.", waitSeconds)
			mu.Unlock()
			writeError(w, http.StatusTooManyRequests, msg)
			return
		}
		mu.Unlock()

		if !visitor.limiter.Allow() {
			visitor.blockedUntil = now.Add(blockDuration)
			mu.Unlock()
			writeError(w, http.StatusTooManyRequests, "Too many requests. Try again later.")
			return
		}

		next.ServeHTTP(w, r)
	})
}
