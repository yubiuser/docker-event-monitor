package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// Adapted from https://github.com/mdeheij/mattergo

// Message is a chat message to be sent using a webhook
type MattermostMessage struct {
	Username string `json:"username"`
	Channel  string `json:"channel"`
	Text     string `json:"text"`
}

// Send a message to a Mattermost chat channel
func sendMattermost(message string, title string) {

	m := MattermostMessage{
		Username: glb_arguments.MattermostUser,
		Channel:  glb_arguments.MattermostChannel,
		Text:     "##### " + title + "\n" + message,
	}

	messageJSON, err := json.Marshal(m)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Mattermost").Msg("Faild to marshal JSON")
		return
	}

	req, err := http.NewRequest("POST", glb_arguments.MattermostURL, bytes.NewBuffer(messageJSON))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Mattermost").Msg("Faild to build request")
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Mattermost").Msg("Faild to send request")
		return
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Mattermost").Msg("")
		return
	}

	logger.Debug().Str("reporter", "Mattermost").Msgf("Mattermost response statusCode: %d", statusCode)
	logger.Debug().Str("reporter", "Mattermost").Msgf("Mattermost response body: %s", string(respBody))

	// Log non successfull status codes
	if statusCode == 200 {
		logger.Debug().Str("reporter", "Mattermost").Msg("Mattermost message delivered")
	} else {
		logger.Error().Str("reporter", "Mattermost").Msg("Pushing Mattermost message failed.")
		logger.Error().Str("reporter", "Mattermost").Msgf("Mattermost response body: %s", string(respBody))
	}

}
