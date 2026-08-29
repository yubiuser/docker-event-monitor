package main

import (
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"
)

// Adapted from https://github.com/mdeheij/mattergo

// Message is a chat message to be sent using a webhook
type MattermostMessage struct {
	Username string `json:"username"`
	Channel  string `json:"channel"`
	Text     string `json:"text"`
}

// Send a message to a Mattermost chat channel
func sendMattermost(message string, title string, errCh chan ReporterError) {

	m := MattermostMessage{
		Username: config.Reporter.Mattermost.User,
		Channel:  config.Reporter.Mattermost.Channel,
		Text:     "##### " + title + "\n" + message,
	}

	e := ReporterError{
		Reporter: "mattermost",
	}

	messageJSON, err := json.Marshal(m)
	if err != nil {
		log.Error().Err(err).Str("reporter", "Mattermost").Msg("Failed to marshal JSON")
		e.Error = errors.New("failed to marshal JSON")
		errCh <- e
		return
	}

	err = sendhttpMessage("Mattermost", config.Reporter.Mattermost.URL, messageJSON)
	if err != nil {
		e.Error = err
		errCh <- e
		return
	}

}
