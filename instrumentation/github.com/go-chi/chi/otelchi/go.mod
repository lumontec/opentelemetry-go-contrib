module go.opentelemetry.io/contrib/instrumentation/github.com/go-chi/chi/otelchi

go 1.14

replace (
	go.opentelemetry.io/contrib => ../../../../../
	go.opentelemetry.io/contrib/propagators => ../../../../../propagators
)

require (
	github.com/go-chi/chi v1.5.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib v0.13.0
	go.opentelemetry.io/contrib/propagators v0.13.0
	go.opentelemetry.io/otel v0.16.0
)
