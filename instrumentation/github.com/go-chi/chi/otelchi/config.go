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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
)

// config is a group of options for this instrumentation.
type config struct {
	TracerProvider trace.TracerProvider
	Propagators    otel.TextMapPropagator
}

// Option applies an option value for a config.
type Option interface {
	Apply(*config)
}

// newConfig returns a config configured with all the passed Options.
func newConfig(opts []Option) *config {
	c := &config{
		Propagators:    global.TextMapPropagator(),
		TracerProvider: global.TracerProvider(),
	}
	for _, o := range opts {
		o.Apply(c)
	}
	return c
}

type propagatorsOption struct{ p otel.TextMapPropagator }

func (o propagatorsOption) Apply(c *config) {
	c.Propagators = o.p
}

// WithPropagators returns an Option to use the Propagators when extracting
// and injecting trace context from HTTP requests.
func WithPropagators(p otel.TextMapPropagator) Option {
	return propagatorsOption{p: p}
}

type tracerProviderOption struct{ tp trace.TracerProvider }

func (o tracerProviderOption) Apply(c *config) {
	c.TracerProvider = o.tp
}

// WithTracerProvider returns an Option to use the TracerProvider when
// creating a Tracer.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return tracerProviderOption{tp: tp}
}