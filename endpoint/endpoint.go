package endpoint

import "golang.org/x/net/context"

// Endpoint is the fundamental building block of servers and clients.
// It represents a single RPC method.
type Endpoint func() error

// Middleware is a chainable behavior modifier for endpoints.
type Middleware func(Endpoint) Endpoint

// Endpoint for low-level request
type SEndpoint func(ctx context.Context, address string, req, rsp interface{}) error

// Middleware for low-level request
type SMiddleware func(SEndpoint) SEndpoint
