package rest

import (
	"context"
	"errors"
	"net/http"

	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/lucsky/cuid"
)

// RequestTracing handles the request tracing context
func RequestTracing(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		requestSource := r.Header.Get(values.HeaderRequestSource)
		if requestSource == "" {
			errM := errors.New("X-Request-Source is empty")

			writeErrorResponse(w, errM, values.Error, errM.Error())
			return
		}

		requestID := r.Header.Get(values.HeaderRequestID)
		if requestID == "" {
			requestID = cuid.New()
		}

		tracingContext := tracing.Context{
			RequestID:     requestID,
			RequestSource: requestSource,
		}

		ctx = context.WithValue(ctx, values.ContextTracingKey, tracingContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	}

	return http.HandlerFunc(fn)
}
