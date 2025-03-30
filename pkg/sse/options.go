package sse

import "time"

type KeepAliveOption struct {
	Interval time.Duration
}

func (o KeepAliveOption) apply(t *sseTransport) {
	t.keepAliveInterval = o.Interval
}

type PortOption struct {
	Port int
}

func (o PortOption) apply(t *sseTransport) {
	t.port = o.Port
}

func WithPort(port int) SSETransportOption {
	return PortOption{
		Port: port,
	}
}

func WithKeepAliveInterval(interval time.Duration) SSETransportOption {
	return KeepAliveOption{
		Interval: interval,
	}
}
