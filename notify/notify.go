package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/megaease/metrics-go/conf"
)

// Result is used to form the notification message
type Result struct {
	// UID is the unique identifier of the result
	UID string
	// Title is the title of the result
	Title string
	// Status is the status of the result
	Status Status
	// Endpoint is the endpoint of the result, usually is the URL or command
	Endpoint string
	// Message is the message of the result
	Message string
	// TimeStamp is the time of the result
	TimeStamp time.Time
}

func NotifyMessage(cfg *conf.Config, msg string) error {
	if cfg.SlackWebhookURL == "" {
		return fmt.Errorf("Slack webhook is empty")
	}

	req, err := http.NewRequest(http.MethodPost, cfg.SlackWebhookURL, bytes.NewBuffer([]byte(msg)))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Close = true

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error response from Slack - code [%d] - msg [%s]", resp.StatusCode, string(buf))
	}
	return nil
}

func NotifyResult(cfg *conf.Config, result *Result) error {
	return nil
}

func toSlack(cfg *conf.Config, r Result) string {
	serviceName := getServiceHostName(cfg)
	jsonMsg := `
	{
		"text": "%s",
		"blocks": [
			{
				"type": "section",
				"text": {
					"type": "mrkdwn",
					"text": "%s"
				}
			},
			{
				"type": "context",
				"elements": [
					{
						"type": "image",
						"image_url": "` + DefaultIconURL + `",
						"alt_text": "` + serviceName + `"
					},
					{
						"type": "mrkdwn",
						"text": "` + serviceName + ` %s"
					}
				]
			}
		]
	}
	`

	body := fmt.Sprintf("*%s*\\n>%s %s\\n>%s",
		r.Title, r.Status.Emoji(), r.Endpoint, jsonEscape(r.Message))
	context := slackTimeFormation(r.TimeStamp, " report at ", DefaultTimeFormat)
	summary := fmt.Sprintf("%s %s - %s", r.Title, r.Status.Emoji(), jsonEscape(r.Message))
	output := fmt.Sprintf(jsonMsg, summary, body, context)
	if !json.Valid([]byte(output)) {
		log.Printf("ToSlack() for %s: Invalid JSON: %s", r.UID, output)
	}
	return output
}

func getServiceHostName(cfg *conf.Config) string {
	return fmt.Sprintf("%s@%s", cfg.ServiceName, cfg.HostName)
}

func jsonEscape(str string) string {
	b, err := json.Marshal(str)
	if err != nil {
		return str
	}
	s := string(b)
	return s[1 : len(s)-1]
}

func slackTimeFormation(t time.Time, act string, format string) string {
	return fmt.Sprintf("<!date^%d^%s{date_num} {time_secs}|%s%s>",
		t.Unix(), act, act, t.UTC().Format(format))
}