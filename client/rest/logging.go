package client

import "fmt"

// Logger is an interface wrapping logging calls required by the client.
type Logger interface {
	Fatalf(string, ...interface{})
}

// fatalf wraps fatal level logging calls from the client. If a logger is
// provided, the message is sent there, otherwise the call panics. Will never
// return.
func (c *Client) fatalf(format string, args ...interface{}) {
	if c.options.logger != nil {
		c.options.logger.Fatalf(format, args...)
	} else {
		panic(fmt.Sprintf(format, args...))
	}
}
