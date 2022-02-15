package notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/mudrex/onyx/pkg/logger"
)

type slackRequestBody struct {
	Text string `json:"text"`
}

func Notify(hook, message string) error {
	if hook == "" {
		logger.Info(message)
		return nil
	}

	slackBody, _ := json.Marshal(slackRequestBody{Text: message})
	req, err := http.NewRequest(http.MethodPost, hook, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return errors.New("unable to send notification to slack")
	}

	return nil
}
