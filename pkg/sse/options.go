package sse

import "time"

type KeepAliveOption struct {
	Interval time.Duration
}

func (o KeepAliveOption) apply(t *sseTransport) {
	t.keepAliveInterval = o.Interval
}
