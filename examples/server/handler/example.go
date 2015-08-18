package handler

import (
	log "github.com/golang/glog"
	bucket "github.com/juju/ratelimit"
	"github.com/justintv90/go-micro/circuitbreaker"
	"github.com/justintv90/go-micro/endpoint"
	"github.com/justintv90/go-micro/ratelimit"
	c "github.com/myodc/go-micro/context"
	example "github.com/myodc/go-micro/examples/server/proto/example"
	"github.com/myodc/go-micro/server"

	"golang.org/x/net/context"
)

type Example struct{}

var testBucket = bucket.NewBucketWithRate(10, 100)

func (e *Example) Call(ctx context.Context, req *example.Request, rsp *example.Response) error {

	// Endpoint construction
	var ep endpoint.Endpoint
	ep = func() error {
		md, _ := c.GetMetadata(ctx)
		log.Infof("Received Example.Call request with metadata: %v", md)
		rsp.Msg = server.Config().Id() + ": Hello " + req.Name
		return nil
	}

	// Including middlewares
	ep = ratelimit.NewTokenBucketLimiter(testBucket)(ep)
	ep = circuitbreaker.Hystrix("test_command", ep, nil, 0, 50, 1000, 0, 0)(ep)

	// Excuting
	ep()

	return nil

}

func (e *Example) Stream(ctx context.Context, req *example.StreamingRequest, response func(interface{}) error) error {
	log.Infof("Received Example.Stream request with count: %d", req.Count)
	for i := 0; i < int(req.Count); i++ {
		log.Infof("Responding: %d", i)

		r := &example.StreamingResponse{
			Count: int64(i),
		}

		if err := response(r); err != nil {
			return err
		}
	}

	return nil
}
