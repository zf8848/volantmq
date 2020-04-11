package transport

import (
	"errors"
	"sync"

	"go.uber.org/zap"

	"github.com/zf8848/volantmq/auth"
	"github.com/zf8848/volantmq/metrics"
	"github.com/zf8848/volantmq/types"
)

// Config is base configuration object used by all transports
type Config struct {
	// AuthManager
	AuthManager *auth.Manager

	Host string
	// Port tcp port to listen on
	Port string
}

// InternalConfig used by server implementation to configure internal specific needs
type InternalConfig struct {
	Handler
	AcceptPool types.Pool
	Metrics    metrics.Bytes
}

type baseConfig struct {
	InternalConfig
	config       Config
	onConnection sync.WaitGroup // nolint: structcheck
	onceStop     sync.Once      // nolint: structcheck
	quit         chan struct{}  // nolint: structcheck
	log          *zap.SugaredLogger
	protocol     string
}

// Provider is interface that all of transports must implement
type Provider interface {
	Protocol() string
	Serve() error
	Close() error
	Port() string
	Ready() error
	Alive() error
}

var (
	// ErrListenerIsOff ...
	ErrListenerIsOff = errors.New("listener is off")
)

// Port return tcp port used by transport
func (c *baseConfig) Port() string {
	return c.config.Port
}

// Protocol return protocol name used by transport
func (c *baseConfig) Protocol() string {
	return c.protocol
}

func (c *baseConfig) baseReady() error {
	select {
	case <-c.quit:
		return ErrListenerIsOff
	default:
	}

	return nil
}

// handleConnection is for the broker to handle an incoming connection from a client
func (c *baseConfig) handleConnection(conn Conn) {
	if c == nil {
		c.log.Error("Invalid connection type")
		return
	}

	var err error

	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	// To establish a connection, we must
	// 1. Read and decode the message.ConnectMessage from the wire
	// 2. If no decoding errors, then authenticate using username and password.
	//    Otherwise, write out to the wire message.ConnackMessage with
	//    appropriate error.
	// 3. If authentication is successful, then either create a new session or
	//    retrieve existing session
	// 4. Write out to the wire a successful message.ConnackMessage message

	// Read the CONNECT message from the wire, if error, then check to see if it's
	// a CONNACK error. If it's CONNACK error, send the proper CONNACK error back
	// to client. Exit regardless of error type.
	// conn.Conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(c.ConnectTimeout))) // nolint: errcheck, gas

	err = c.OnConnection(conn, c.config.AuthManager)
}
