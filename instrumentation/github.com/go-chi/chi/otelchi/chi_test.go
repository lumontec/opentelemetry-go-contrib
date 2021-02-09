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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	b3prop "go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	otelglobal "go.opentelemetry.io/otel/api/global"
	oteltrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/api/trace/tracetest"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagators"
)

func TestChildSpanFromGlobalTracer(t *testing.T) {
	otelglobal.SetTracerProvider(tracetest.NewTracerProvider())

	c := chi.NewRouter()
	c.Use(Middleware("foobar"))
	c.Get("/user/:id", func(res http.ResponseWriter, req *http.Request) {
		span := oteltrace.SpanFromContext(req.Context())
		_, ok := span.(*tracetest.Span)
		assert.True(t, ok)
		spanTracer := span.Tracer()
		mockTracer, ok := spanTracer.(*tracetest.Tracer)
		require.True(t, ok)
		assert.Equal(t, instrumentationName, mockTracer.Name)
		res.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/user/123", nil)
	res := httptest.NewRecorder()

	c.ServeHTTP(res, req)
}

func TestChildSpanNames(t *testing.T) {
	sr := new(tracetest.StandardSpanRecorder)
	tp := tracetest.NewTracerProvider(tracetest.WithSpanRecorder(sr))

	c := chi.NewRouter()
	c.Use(Middleware("foobar", WithTracerProvider(tp)))
	c.Get("/user/:id", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
	})

	c.Get("/book/:title", func(res http.ResponseWriter, req *http.Request) {
		_, err := res.Write(([]byte)("ok"))
		if err != nil {
			t.Error(err)
		}
	})

	req := httptest.NewRequest("GET", "/user/123", nil)
	res := httptest.NewRecorder()
	c.ServeHTTP(res, req)

	req = httptest.NewRequest("GET", "/book/foo", nil)
	res = httptest.NewRecorder()
	c.ServeHTTP(res, req)

	spans := sr.Completed()
	require.Len(t, spans, 2)
	span := spans[0]
	assert.Equal(t, "/user/123", span.Name()) // TODO: span name should show router template, eg /user/:id
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attrs := span.Attributes()
	assert.Equal(t, label.StringValue("foobar"), attrs["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusOK), attrs["http.status_code"])
	assert.Equal(t, label.StringValue("GET"), attrs["http.method"])
	assert.Equal(t, label.StringValue("/user/123"), attrs["http.target"])
	// TODO: span name should show router template, eg /user/:id
	//assert.Equal(t, label.StringValue("/user/:id"), span.Attributes["http.route"])

	span = spans[1]
	assert.Equal(t, "/book/foo", span.Name()) // TODO: span name should show router template, eg /book/:title
	assert.Equal(t, oteltrace.SpanKindServer, span.SpanKind())
	attrs = span.Attributes()
	assert.Equal(t, label.StringValue("foobar"), attrs["http.server_name"])
	assert.Equal(t, label.IntValue(http.StatusOK), attrs["http.status_code"])
	assert.Equal(t, label.StringValue("GET"), attrs["http.method"])
	assert.Equal(t, label.StringValue("/book/foo"), attrs["http.target"])
	// TODO: span name should show router template, eg /book/:title
	//assert.Equal(t, label.StringValue("/book/:title"), span.Attributes["http.route"])
}

func TestGetSpanNotInstrumented(t *testing.T) {
	c := chi.NewRouter()
	c.Get("/user/:id", func(res http.ResponseWriter, req *http.Request) {
		span := oteltrace.SpanFromContext(req.Context())
		ok := !span.SpanContext().IsValid()
		assert.True(t, ok)
		res.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/user/123", nil)
	res := httptest.NewRecorder()

	c.ServeHTTP(res, req)
}

func TestPropagationWithGlobalPropagators(t *testing.T) {
	tracer := tracetest.NewTracerProvider().Tracer("test-tracer")
	otelglobal.SetTextMapPropagator(propagators.TraceContext{})

	req := httptest.NewRequest("GET", "/user/123", nil)
	res := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	otelglobal.TextMapPropagator().Inject(ctx, req.Header)

	c := chi.NewRouter()
	c.Use(Middleware("foobar"))
	c.Get("/user/:id", func(res http.ResponseWriter, req *http.Request) {
		span := oteltrace.SpanFromContext(req.Context())
		mspan, ok := span.(*tracetest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID())
		res.WriteHeader(http.StatusOK)
	})

	c.ServeHTTP(res, req)
	otelglobal.SetTextMapPropagator(otel.NewCompositeTextMapPropagator())
}

func TestPropagationWithCustomPropagators(t *testing.T) {
	tp := tracetest.NewTracerProvider()
	tracer := tp.Tracer("test-tracer")
	b3 := b3prop.B3{}

	req := httptest.NewRequest("GET", "/user/123", nil)
	res := httptest.NewRecorder()

	ctx, pspan := tracer.Start(context.Background(), "test")
	b3.Inject(ctx, req.Header)

	c := chi.NewRouter()
	c.Use(Middleware("foobar", WithTracerProvider(tp), WithPropagators(b3)))
	c.Get("/user/:id", func(res http.ResponseWriter, req *http.Request) {
		span := oteltrace.SpanFromContext(req.Context())
		mspan, ok := span.(*tracetest.Span)
		require.True(t, ok)
		assert.Equal(t, pspan.SpanContext().TraceID, mspan.SpanContext().TraceID)
		assert.Equal(t, pspan.SpanContext().SpanID, mspan.ParentSpanID())
		res.WriteHeader(http.StatusOK)
	})

	c.ServeHTTP(res, req)
}
