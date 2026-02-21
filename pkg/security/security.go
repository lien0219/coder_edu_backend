package security

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// CORS 中间件 仅允许白名单中的Origin，支持Credentials
func CORS(allowedOrigins []string) gin.HandlerFunc {
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != "" && originSet[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Secure 中间件
func Secure() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止MIME嗅探
		c.Header("X-Content-Type-Options", "nosniff")
		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")
		// XSS保护
		c.Header("X-XSS-Protection", "1; mode=block")
		// HSTS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		c.Next()
	}
}

// visitor 包装限流器和最后活跃时间，用于定期清理
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter 限流中间件 按IP限流，自动清理过期条目
func RateLimiter(maxRequests int, window time.Duration) gin.HandlerFunc {
	store := make(map[string]*visitor)
	var mu sync.Mutex

	go func() {
		expiry := window * 3
		if expiry < time.Minute {
			expiry = time.Minute
		}
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, v := range store {
				if time.Since(v.lastSeen) > expiry {
					delete(store, ip)
				}
			}
			mu.Unlock()
		}
	}()

	r := rate.Every(window / time.Duration(maxRequests))

	return func(c *gin.Context) {
		key := c.ClientIP()

		mu.Lock()
		v, exists := store[key]
		if !exists {
			v = &visitor{
				limiter: rate.NewLimiter(r, maxRequests),
			}
			store[key] = v
		}
		v.lastSeen = time.Now()
		mu.Unlock()

		if !v.limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}

		c.Next()
	}
}
