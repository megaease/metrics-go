package metricshub

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNotify(t *testing.T) {
	cfg := &MetricsHubConfig{
		ServiceName: "test",
		HostName:    "test",
		Labels: map[string]string{
			"env": "dev",
		},
		SlackWebhookURL: "",
	}
	mHub := NewMetricsHub(cfg)

	result := &Result{
		UID:      "zxcfju-1734464397-eace373587dc410f",
		Title:    "NetDisk Download Failed",
		Endpoint: fmt.Sprintf("pvcName: %s", "xxx"),
		Message: fmt.Sprintf("tenantID: %s, srcPath: %s, dstPath: %s, DriveType: %s, error: %v",
			"Y", "/home", "/app", "BaiduPan", "not enough storage space"),
	}

	err := mHub.NotifyResult(result)
	assert.Nil(t, err)
}
