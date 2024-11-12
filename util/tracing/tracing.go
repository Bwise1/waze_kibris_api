package tracing

import (
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/lucsky/cuid"
)

type Context struct {
	// RequestID specifies the request ID if empty, a new request ID should be generated
	RequestID     string
	RequestSource string
}

// New creates a new tracing context
func New() *Context {
	return &Context{
		RequestID:     cuid.New(),
		RequestSource: "service-name",
	}
}

// OutgoingHeaders returns the tracing information for response headers
func (tc *Context) OutgoingHeaders() map[string]string {
	return map[string]string{
		values.HeaderRequestID:     tc.RequestID,
		values.HeaderRequestSource: tc.RequestSource,
	}
}
