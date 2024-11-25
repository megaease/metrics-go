package middleware

import (
	echo "github.com/labstack/echo/v4"
	"strings"

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

			groupPath := path

			// TODO: Support more complex path.
			if strings.Contains(path, "/:") {
				groupPath = strings.Split(path, "/:")[0]
			}
			requestMetric := &metricshub.RequestMetric{
				StatusCode: code,
				Duration:   processTime,
				ReqSize:    uint64(bodyBytesReceived),
				RespSize:   uint64(bodyBytesSent),
			}
			hub.UpdateInternalMetrics(requestMetric, method, groupPath)

			return err
		}
	}
}
