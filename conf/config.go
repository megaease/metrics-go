package conf

type Config struct {
	// ServiceName is the name of the service. It is required.
	// The service name will be used as a label in the http metrics.
	// Other custom metrics will not add this label, should be added manually.
	ServiceName string `yaml:"serviceName" json:"serviceName"`
	// HostName is the hostname of the service. It is required.
	// The hostname will be used as a label in the http metrics.
	// Other custom metrics will not add this label, should be added manually.
	HostName string `yaml:"hostName" json:"hostName"`
	// Labels is the additional labels for the service.
	// This labels will be added to the http metrics.
	// Other custom metrics will not add this label, should be added manually.
	// +optional
	Labels map[string]string `yaml:"labels" json:"labels"`
	// SlackWebhookURL is the webhook URL for Slack notifications.
	// If not set, the default value will be used when sending notifications.
	// So be sure to set this value if you want to receive notifications.
	// +optional
	SlackWebhookURL string `yaml:"slackWebhookURL" json:"slackWebhookURL"`
}
