package middleware

import (
	"net/http"
	"slices"

	"github.com/zeldojov/zexgo/internal/ratelimiter"
	"github.com/zeldojov/zexgo/internal/store"
)

type ErrorHandlers struct {
	NotFound            http.Handler
	InternalServerError http.Handler
	MethodNotAllowed    http.Handler
}

type Middleware struct {
	errors ErrorHandlers
}

type MiddlewareFunc func(http.Handler) http.Handler

func CreateChain(middlewares ...Middleware) Middleware {
	return func(handler http.Handler) http.Handler {
		for _, middleware := range slices.Backward(middlewares) {
			handler = middleware(handler)
		}

		return handler
	}
}

func NewPublicChain(st *store.Store, limiter *ratelimiter.RateLimiter) Middleware {
	return CreateChain(
		AllowedMethods,

		func(next http.Handler) http.Handler {
			return RateLimiter(limiter)(next)
		},

		func(next http.Handler) http.Handler {
			return Session(st, next)
		},

		func(next http.Handler) http.Handler {
			return ValidateSession(st, next)
		},

		CSRF,
	)
}

func NewGuestChain(
	st *store.Store,
	limiter *ratelimiter.RateLimiter,
) Middleware {
	return CreateChain(
		AllowedMethods,

		func(next http.Handler) http.Handler {
			return RateLimiter(limiter)(next)
		},

		func(next http.Handler) http.Handler {
			return Session(st, next)
		},

		func(next http.Handler) http.Handler {
			return ValidateSession(st, next)
		},

		CSRF,
		Guest,
	)
}

func NewAuthChain(
	st *store.Store,
	limiter *ratelimiter.RateLimiter,
) Middleware {
	return CreateChain(
		AllowedMethods,
		func(next http.Handler) http.Handler {
			return RateLimiter(limiter)(next)
		},

		func(next http.Handler) http.Handler {
			return Session(st, next)
		},

		func(next http.Handler) http.Handler {
			return ValidateSession(st, next)
		},

		CSRF,
		Auth(),
	)
}
