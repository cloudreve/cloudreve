package middleware

import (
	"strconv"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
)

const rateLimitPrefix = "rate_limit:"

// RateLimit applies a fixed-window rate limit to requests sharing the same
// bucket key. Once `limit` requests arrive within `window`, further requests
// are rejected until the window expires. State lives in the shared KV store,
// so limits hold across cluster nodes when Redis is configured.
//
// The counter is approximate (read-then-write), which is acceptable for
// abuse prevention — it bounds attempts to roughly `limit` per window.
func RateLimit(limit int, window time.Duration, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		kv := dependency.FromContext(c).KV()
		bucket := rateLimitPrefix + key(c)

		count := 0
		if raw, ok := kv.Get(bucket); ok {
			if v, ok := raw.(int); ok {
				count = v
			}
		}

		if count >= limit {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			c.JSON(200, serializer.NewError(serializer.CodeRateLimited, "Too many requests, please try again later.", nil))
			c.Abort()
			return
		}

		_ = kv.Set(bucket, count+1, int(window.Seconds()))
		c.Next()
	}
}

// RateLimitByIP keys the rate limit on the client IP and a static bucket
// name, e.g. RateLimitByIP("login", 10, time.Minute).
func RateLimitByIP(bucket string, limit int, window time.Duration) gin.HandlerFunc {
	return RateLimit(limit, window, func(c *gin.Context) string {
		return bucket + ":" + c.ClientIP()
	})
}
