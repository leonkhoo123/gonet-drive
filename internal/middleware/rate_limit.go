package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips *cache.Cache
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	// Cache limiters for 1 hour, purge every 15 minutes to prevent memory leaks
	return &IPRateLimiter{
		ips: cache.New(1*time.Hour, 15*time.Minute),
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	limiter, exists := i.ips.Get(ip)
	if !exists {
		newLimiter := rate.NewLimiter(i.r, i.b)
		i.ips.Set(ip, newLimiter, cache.DefaultExpiration)
		return newLimiter
	}
	// Reset the expiration on access so active IPs don't get their limiters dropped
	i.ips.Set(ip, limiter, cache.DefaultExpiration)
	return limiter.(*rate.Limiter)
}

var loginLimiter = NewIPRateLimiter(rate.Every(1*time.Second), 5)
var shareVerifyLimiter = NewIPRateLimiter(rate.Every(1*time.Second), 5) // 5 requests per second per IP for share verification

func LoginRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := loginLimiter.GetLimiter(ip)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			return
		}
		c.Next()
	}
}

func ShareVerifyRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := shareVerifyLimiter.GetLimiter(ip)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many attempts, please try again later",
			})
			return
		}
		c.Next()
	}
}
