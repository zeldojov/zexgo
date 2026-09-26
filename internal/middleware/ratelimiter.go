package middleware

import (
	"net/http"

	"github.com/zeldojov/zexgo/internal/ratelimiter"
	"github.com/zeldojov/zexgo/internal/utils"
)

func RateLimiter(limiter *ratelimiter.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := utils.GetClientIP(r)

			if !limiter.Allow(ip) {
				http.Error(
					w,
					"too many requests",
					http.StatusTooManyRequests,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
