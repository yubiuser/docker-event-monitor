package main

import (
	"encoding/json"
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

func sendPushover(timestamp time.Time, message string, title string, errCh chan ReporterError) {
	// Send a message to Pushover

	m := PushoverMessage{
		Token:     glb_arguments.PushoverAPIToken,
		User:      glb_arguments.PushoverUserKey,
		Title:     title,
		Message:   message,
		Timestamp: strconv.FormatInt(timestamp.Unix(), 10),
	}

	e := ReporterError{
		Reporter: "Pushover",
	}

	messageJSON, err := json.Marshal(m)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Pushover").Msg("Faild to marshal JSON")
		e.Error = err
		errCh <- e
		return
	}

	err = sendhttpMessage("Pushover", "https://api.pushover.net/1/messages.json", messageJSON)
	if err != nil {
		e.Error = err
		errCh <- e
		return
	}

}
