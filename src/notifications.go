package main

import (
	"sync"
	"time"
)

func sendNotifications(timestamp time.Time, message string, title string) {
	// Sending messages to different services as goroutines concurrently
	// Adding a wait group here to delay execution until all functions return,
	// otherwise delaying in processEvent() would not make any sense

	var wg sync.WaitGroup

	// If there is a server tag, add it to the title
	if len(glb_arguments.ServerTag) > 0 {
		title = "[" + glb_arguments.ServerTag + "] " + title
	}

	if glb_arguments.Pushover {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendPushover(timestamp, message, title)
		}()
	}

	if glb_arguments.Gotify {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendGotify(message, title)
		}()
	}

	if glb_arguments.Mail {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendMail(timestamp, message, title)
		}()
	}

	if glb_arguments.Mattermost {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendMattermost(message, title)
		}()
	}
	wg.Wait()

}
