package client

import (
	"github.com/justintv90/go-micro/broker"
	"github.com/justintv90/go-micro/registry"
	"github.com/justintv90/go-micro/transport"
)

type options struct {
	broker    broker.Broker
	registry  registry.Registry
	transport transport.Transport
}

func Broker(b broker.Broker) Option {
	return func(o *options) {
		o.broker = b
	}
}

func Registry(r registry.Registry) Option {
	return func(o *options) {
		o.registry = r
	}
}

func Transport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}
