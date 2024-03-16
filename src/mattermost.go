package main

import (
	"encoding/json"
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
		Username: glb_arguments.MattermostUser,
		Channel:  glb_arguments.MattermostChannel,
		Text:     "##### " + title + "\n" + message,
	}

	e := ReporterError{
		Reporter: "Mattermost",
	}

	messageJSON, err := json.Marshal(m)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Mattermost").Msg("Faild to marshal JSON")
		e.Error = err
		errCh <- e
		return
	}

	err = sendhttpMessage("Mattermost", glb_arguments.MattermostURL, messageJSON)
	if err != nil {
		e.Error = err
		errCh <- e
		return
	}

}
