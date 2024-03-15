package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
)

type PushoverMessage struct {
	Token     string `json:"token"`
	User      string `json:"user"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func sendPushover(timestamp time.Time, message string, title string) {
	// Send a message to Pushover

	m := PushoverMessage{
		Token:     glb_arguments.PushoverAPIToken,
		User:      glb_arguments.PushoverUserKey,
		Title:     title,
		Message:   message,
		Timestamp: strconv.FormatInt(timestamp.Unix(), 10),
	}

	messageJSON, err := json.Marshal(m)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Pushover").Msg("Faild to marshal JSON")
		return
	}

	// Create request
	req, err := http.NewRequest("POST", "https://api.pushover.net/1/messages.json", bytes.NewBuffer(messageJSON))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Pushover").Msg("Faild to build request")
		return
	}

	// define custom httpClient with a default timeout
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	// Send request
	resp, err := netClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Pushover").Msg("Faild to send request")
		return
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Pushover").Msg("")
		return
	}

	// Log non successfull status codes
	if statusCode == 200 {
		logger.Debug().
			Str("reporter", "Pushover").
			Int("statusCode", statusCode).
			Str("responseBody", string(respBody)).
			Msg("Message delivered")
	} else {
		logger.Error().
			Str("reporter", "Pushover").
			Int("statusCode", statusCode).
			Str("responseBody", string(respBody)).
			Msg("Pushing message failed.")
	}

}
