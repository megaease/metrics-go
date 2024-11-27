# Metrics-Go

metrics-go is a powerful and flexible library for collecting, managing, and exposing application metrics in Go. Designed for general-purpose use, it supports Prometheus as the primary backend for monitoring and provides tools to seamlessly integrate with web frameworks like Echo and Gin.

## Features

- Prometheus Integration: Expose metrics easily via a `/metrics` like endpoint.
- Granular HTTP Metrics: Track HTTP request durations, sizes, and statuses.
- Framework-Specific Middleware: Built-in support for Gin and Echo.
- Custom Metrics: Extend functionality to add business-specific metrics.
- Real-Time Monitoring: Collect exponentially-weighted rate, percentile latency metrics (e.g., m1, m5, p99, p95).

## Install

```bash
go get github.com/megaease/metrics-go
```

## Quick Start

The MetricsHub serves as the central component to manage and update metrics:

```go
import (
	"github.com/labstack/echo/v4"
	"github.com/megaease/metrics-go/metricshub"
	"github.com/megaease/metrics-go/middleware"
)

config := &metricshub.MetricsHubConfig{
    ServiceName: "vm-operator-echo",
    HostName:    "sprite-run-serverless-01",
}
// 1. Initialize MetricsHub
mHub := metricshub.NewMetricsHub(config)

// 2. Expose Metrics via HTTP
app := echo.New()
app.Use(middleware.NewEchoMetricsCollector(mHub))
app.GET("/metrics", echo.WrapHandler(mHub.HTTPHandler()))
app.Start(fmt.Sprintf(":%d", 8080))
```

The full examples can be found in the [examples](./examples) directory.

## Advanced Usage

TODO

## Community

- [Join Slack Workspace](https://cloud-native.slack.com/messages/easegress) for requirement, issue and development.
- [MegaEase on Twitter](https://twitter.com/megaease)

## Contributing

The project welcomes contributions and suggestions that abide by the [CNCF Code of Conduct](./CODE_OF_CONDUCT.md).

## License

Easegress is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.
