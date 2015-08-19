package client

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/justintv90/go-micro/circuitbreaker"

	"github.com/justintv90/go-micro/endpoint"

	"github.com/justintv90/go-micro/broker"
	c "github.com/justintv90/go-micro/context"
	"github.com/justintv90/go-micro/errors"
	"github.com/justintv90/go-micro/registry"
	"github.com/justintv90/go-micro/transport"

	rpc "github.com/youtube/vitess/go/rpcplus"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

type headerRoundTripper struct {
	r http.RoundTripper
}

type rpcClient struct {
	once sync.Once
	opts options
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func newRpcClient(opt ...Option) Client {
	var opts options

	for _, o := range opt {
		o(&opts)
	}

	if opts.transport == nil {
		opts.transport = transport.DefaultTransport
	}

	if opts.broker == nil {
		opts.broker = broker.DefaultBroker
	}

	var once sync.Once

	return &rpcClient{
		once: once,
		opts: opts,
	}
}

func (t *headerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-Client-Version", "1.0")
	return t.r.RoundTrip(r)
}

func (r *rpcClient) call(ctx context.Context, address string, request Request, response interface{}) error {
	msg := &transport.Message{
		Header: make(map[string]string),
	}

	md, ok := c.GetMetadata(ctx)
	if ok {
		for k, v := range md {
			msg.Header[k] = v
		}
	}

	msg.Header["Content-Type"] = request.ContentType()

	c, err := r.opts.transport.Dial(address)
	if err != nil {
		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}

	client := rpc.NewClientWithCodec(newRpcPlusCodec(msg, c))
	defer client.Close()
	return client.Call(ctx, request.Method(), request.Request(), response)
}

func (r *rpcClient) stream(ctx context.Context, address string, request Request, responseChan interface{}) (Streamer, error) {
	msg := &transport.Message{
		Header: make(map[string]string),
	}

	md, ok := c.GetMetadata(ctx)
	if ok {
		for k, v := range md {
			msg.Header[k] = v
		}
	}

	msg.Header["Content-Type"] = request.ContentType()

	c, err := r.opts.transport.Dial(address, transport.WithStream())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
	}

	client := rpc.NewClientWithCodec(newRpcPlusCodec(msg, c))
	call := client.StreamGo(request.Method(), request.Request(), responseChan)

	return &rpcStream{
		request: request,
		call:    call,
		client:  client,
	}, nil
}

func (r *rpcClient) CallRemote(ctx context.Context, address string, request Request, response interface{}) error {
	return r.call(ctx, address, request, response)
}

// TODO: Call(..., opts *Options) error {
func (r *rpcClient) Call(ctx context.Context, request Request, response interface{}) error {
	service, err := registry.GetService(request.Service())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	if len(service.Nodes) == 0 {
		return errors.NotFound("go.micro.client", "Service not found")
	}

	n := rand.Int() % len(service.Nodes)
	node := service.Nodes[n]

	address := node.Address
	if node.Port > 0 {
		address = fmt.Sprintf("%s:%d", address, node.Port)
	}

	// Endpoint construction

	var ep endpoint.Endpoint

	ep = func() error {
		return r.call(ctx, address, request, response)
	}

	commandName := fmt.Sprintf("%s_%s", service.Name, request.Method())

	callback := func(err error) error {
		return err
	}
	// Including middlewares
	ep = circuitbreaker.Hystrix(commandName, ep, callback, 1000, 25, 100, 0, 0)(ep)

	return ep()
}

func (r *rpcClient) StreamRemote(ctx context.Context, address string, request Request, responseChan interface{}) (Streamer, error) {
	return r.stream(ctx, address, request, responseChan)
}

func (r *rpcClient) Stream(ctx context.Context, request Request, responseChan interface{}) (Streamer, error) {
	service, err := registry.GetService(request.Service())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	if len(service.Nodes) == 0 {
		return nil, errors.NotFound("go.micro.client", "Service not found")
	}

	n := rand.Int() % len(service.Nodes)
	node := service.Nodes[n]

	address := node.Address
	if node.Port > 0 {
		address = fmt.Sprintf("%s:%d", address, node.Port)
	}

	return r.stream(ctx, address, request, responseChan)
}

func (r *rpcClient) Publish(ctx context.Context, p Publication) error {
	md, ok := c.GetMetadata(ctx)
	if !ok {
		md = make(map[string]string)
	}
	md["Content-Type"] = p.ContentType()

	// encode message body
	var body []byte

	switch p.ContentType() {
	case "application/octet-stream":
		b, err := proto.Marshal(p.Message().(proto.Message))
		if err != nil {
			return err
		}
		body = b
	case "application/json":
		b, err := json.Marshal(p.Message())
		if err != nil {
			return err
		}
		body = b
	}

	r.once.Do(func() {
		r.opts.broker.Connect()
	})

	return r.opts.broker.Publish(p.Topic(), &broker.Message{
		Header: md,
		Body:   body,
	})
}

func (r *rpcClient) NewPublication(topic string, message interface{}) Publication {
	return r.NewProtoPublication(topic, message)
}

func (r *rpcClient) NewProtoPublication(topic string, message interface{}) Publication {
	return newRpcPublication(topic, message, "application/octet-stream")
}
func (r *rpcClient) NewRequest(service, method string, request interface{}) Request {
	return r.NewProtoRequest(service, method, request)
}

func (r *rpcClient) NewProtoRequest(service, method string, request interface{}) Request {
	return newRpcRequest(service, method, request, "application/octet-stream")
}

func (r *rpcClient) NewJsonRequest(service, method string, request interface{}) Request {
	return newRpcRequest(service, method, request, "application/json")
}
