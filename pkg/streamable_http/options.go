package streamable_http

import "time"

// KeepStreamAliveInterval is an option for the streamable HTTP transport that sets the keep-alive interval
// for long-running SSE streams.
//
// The default value is 5 seconds. It would result in sending a comment event
// every specified interval to keep the connection alive to prevent proxies to close it due to inactivity.
// Set value to 0 to disable keep-alive messages.
type KeepStreamAliveInterval struct {
	Interval time.Duration
}

func (o KeepStreamAliveInterval) apply(t *streamableHttpTransport) {
	t.keepStreamAliveInterval = o.Interval
}

type Endpoint struct {
	Hostname string
	Port     int
	Path     string
}

func (o Endpoint) apply(t *streamableHttpTransport) {
	t.port = o.Port
	t.hostname = o.Hostname
	t.path = o.Path
}
