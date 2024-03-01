package main

import "github.com/gregdel/pushover"

func sendPushover(message string, title string) {
	// Create a new pushover app with an API token
	app := pushover.New(glb_arguments.PushoverAPIToken)

	// Create a new recipient (user key)
	recipient := pushover.NewRecipient(glb_arguments.PushoverUserKey)

	// Create the message to send
	pushmessage := pushover.NewMessageWithTitle(message, title)

	// Send the message to the recipient
	response, err := app.SendMessage(pushmessage, recipient)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Pushover").Msg("")
		return
	}
	if response != nil {
		logger.Debug().Str("reporter", "Pushover").Msgf("%s", response)
	}

	if (*response).Status == 1 {
		// Pushover returns 1 if the message request to the API was valid
		// https://pushover.net/api#response
		logger.Debug().Str("reporter", "Pushover").Msgf("Pushover message delivered")
		return
	}

	// if response Status !=1
	logger.Error().Str("reporter", "Pushover").Msg("Pushover message not delivered")

}
