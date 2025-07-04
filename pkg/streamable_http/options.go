package streamable_http

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

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

	// AuthHost is the host that will be used in
	// authorization discovery as the host through which
	// the server is reachable from clients.
	// Can include both hostname and port.
	// If empty, the server will use the hostname and port
	// that the server is listening on.
	AuthHost string
	// AuthScheme is the scheme that will be used in
	// authorization discovery as the scheme through which
	// the server is reachable from clients.
	// For production servers, it should be set to "https".
	AuthScheme string
}

func (o Endpoint) apply(t *streamableHttpTransport) {
	if o.Port != 0 {
		t.port = o.Port
	}
	if o.Hostname != "" {
		t.hostname = o.Hostname
	}
	if o.Path != "" {
		t.path = o.Path
	}

	if o.AuthHost != "" {
		t.authHost = o.AuthHost
	} else {
		t.authHost = t.hostname
		if t.port != 0 {
			t.authHost = fmt.Sprintf("%s:%d", t.hostname, t.port)
		}
	}
	if o.AuthScheme != "" {
		t.authScheme = o.AuthScheme
	} else {
		t.authScheme = "https"
	}
}

type TlsConfig struct {
	TLS *tls.Config
}

func (o TlsConfig) apply(t *streamableHttpTransport) {
	t.tlsConfig = o.TLS
}

type EchoConfigurer struct {
	Configure func(e *echo.Echo)
}

func (o EchoConfigurer) apply(t *streamableHttpTransport) {
	if o.Configure != nil {
		o.Configure(t.e)
	}
}
