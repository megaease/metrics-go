package middleware

import (
	echo "github.com/labstack/echo/v4"

	"github.com/megaease/metrics-go/metricshub"
	"github.com/megaease/metrics-go/utils/fasttime"
)

func NewMetricsCollector(hub *metricshub.MetricsHub) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			startAt := fasttime.Now()

			err := next(ctx)
			if err != nil {
				ctx.Error(err)
			}
			processTime := fasttime.Since(startAt)
			path := ctx.Path()
			method := ctx.Request().Method
			code := ctx.Response().Status
			bodyBytesReceived := ctx.Request().ContentLength
			bodyBytesSent := ctx.Response().Size

			// We just use the registered router path as the group path.
			groupPath := path

			requestMetric := &metricshub.RequestMetric{
				StatusCode: code,
				Duration:   processTime,
				ReqSize:    uint64(bodyBytesReceived),
				RespSize:   uint64(bodyBytesSent),
			}
			hub.UpdateHTTPRequestMetrics(requestMetric, method, groupPath)

			return err
		}
	}
}
