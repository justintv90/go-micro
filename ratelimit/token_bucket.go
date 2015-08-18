package ratelimit

import (
	"errors"
	"time"

	"github.com/justintv90/go-micro/endpoint"

	"github.com/juju/ratelimit"
)

// ErrLimited is returned in the request path when the rate limiter is
// triggered and the request is rejected.
var ErrLimited = errors.New("rate limit exceeded")

// NewTokenBucketLimiter returns an endpoint.Middleware that acts as a rate
// limiter based on a token-bucket algorithm. Requests that would exceed the
// maximum request rate are simply rejected with an error.
func NewTokenBucketLimiter(tb *ratelimit.Bucket) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func() error {
			if tb.TakeAvailable(1) == 0 {
				return ErrLimited
			}
			return next()
		}
	}
}

// NewTokenBucketThrottler returns an endpoint.Middleware that acts as a
// request throttler based on a token-bucket algorithm. Requests that would
// exceed the maximum request rate are delayed via the parameterized sleep
// function. By default you may pass time.Sleep.
func NewTokenBucketThrottler(tb *ratelimit.Bucket, sleep func(time.Duration)) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func() error {
			sleep(tb.Take(1))
			return next()
		}
	}
}
