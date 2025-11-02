package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// visitorInfo holds rate limiter and last seen time for a visitor
type visitorInfo struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter manages rate limiters for different IP addresses
type IPRateLimiter struct {
	visitors map[string]*visitorInfo
	mu       sync.RWMutex
	r        rate.Limit // requests per second
	b        int        // burst size
}

// NewIPRateLimiter creates a new IP-based rate limiter
// r is the rate (requests per second)
// b is the burst size (max requests that can be made in a burst)
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		visitors: make(map[string]*visitorInfo),
		r:        r,
		b:        b,
	}

	// Start cleanup goroutine to remove old visitors
	go limiter.cleanupVisitors()

	return limiter
}

// getVisitor returns the rate limiter for the given IP
func (i *IPRateLimiter) getVisitor(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	v, exists := i.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(i.r, i.b)
		i.visitors[ip] = &visitorInfo{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	// Update last seen time
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes visitors that haven't been seen for 5 minutes
func (i *IPRateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		i.mu.Lock()
		for ip, v := range i.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(i.visitors, ip)
			}
		}
		i.mu.Unlock()
	}
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
// Example usage:
//   - RateLimitMiddleware(100, 10) = 100 requests per second with burst of 10
//   - RateLimitMiddleware(5, 2) = 5 requests per second with burst of 2
func RateLimitMiddleware(requestsPerSecond float64, burst int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate.Limit(requestsPerSecond), burst)

	return func(c *gin.Context) {
		// Get client IP
		ip := c.ClientIP()

		// Get rate limiter for this IP
		limiter := limiter.getVisitor(ip)

		// Check if request is allowed
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByMinute is a convenience function for rate limiting by minute
// Example: RateLimitByMinute(60, 10) = 60 requests per minute with burst of 10
func RateLimitByMinute(requestsPerMinute int, burst int) gin.HandlerFunc {
	requestsPerSecond := float64(requestsPerMinute) / 60.0
	return RateLimitMiddleware(requestsPerSecond, burst)
}
