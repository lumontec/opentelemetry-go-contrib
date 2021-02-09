// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelchi

import (
	"fmt"
	"net/http"
	"strconv"

	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/semconv"

	otelcontrib "go.opentelemetry.io/contrib"
)

// instrumentationName is the name of this instrumentation package.
const instrumentationName = "go.opentelemetry.io/contrib/instrumentation/github.com/go-chi/chi/otelchi"

// Middleware returns a go-chi Handler to trace requests to the server.
func Middleware(service string, opts ...Option) func(next http.Handler) http.Handler {
	cfg := newConfig(opts)
	tracer := cfg.TracerProvider.Tracer(
		instrumentationName,
		oteltrace.WithInstrumentationVersion(otelcontrib.SemVersion()),
	)
	return func(next http.Handler) http.Handler {
		fn := func(res http.ResponseWriter, req *http.Request) {
			savedCtx := req.Context()
			defer func() {
				req = req.WithContext(savedCtx)
			}()

			ctx := cfg.Propagators.Extract(savedCtx, req.Header)
			opts := []oteltrace.SpanOption{
				oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", req)...),
				oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(req)...),
				oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(service, "", req)...),
				oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			}
			// TODO: span name should be router template not the actual request path, eg /user/:id vs /user/123
			spanName := req.RequestURI
			if spanName == "" {
				spanName = fmt.Sprintf("HTTP %s route not found", req.Method)
			}
			ctx, span := tracer.Start(ctx, spanName, opts...)
			defer span.End()

			// pass the span through the request context
			req = req.WithContext(ctx)

			// serve the request to the next middleware
			next.ServeHTTP(res, req)

			status, _ := strconv.Atoi(req.Response.Status)
			attrs := semconv.HTTPAttributesFromHTTPStatusCode(status)
			spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(status)
			span.SetAttributes(attrs...)
			span.SetStatus(spanStatus, spanMessage)
		}
		return http.HandlerFunc(fn)
	}
}
