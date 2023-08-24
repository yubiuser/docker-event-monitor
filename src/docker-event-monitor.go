package main

import (
	"context"
	"fmt"
  "os"
  "time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
  "github.com/docker/docker/api/types/filters"
  "github.com/docker/docker/api/types/events"

	"github.com/gregdel/pushover"
  log "github.com/sirupsen/logrus"
  "github.com/jnovack/flag"
)

// todo
// Argument Parsing - filters
// memory leak?
// generalize to all event/typs

// Global variables
var PUSHOVER_APITOKEN, PUSHOVER_USER string
var PUSHOVER_DELAY int64

func sendPushover(message,title string) bool {


	// Create a new pushover app with a API token
	app := pushover.New(PUSHOVER_APITOKEN)

	// Create a new recipient (user key)
	recipient := pushover.NewRecipient(PUSHOVER_USER)

	// Create the message to send
	pushmessage := pushover.NewMessageWithTitle(message,title)

	// Send the message to the recipient
	response, err := app.SendMessage(pushmessage, recipient)
	if err != nil {
		log.Panic(err)
	}
	if response != nil {
		log.Debugf("%s", response)
	}

	if (*response).Status == 1 {
		// Pushover returns 1 if the message request to the API was valid
		// https://pushover.net/api#response
		return true
	}
	// default to false if response Status !=1
	return false
}

func processEvent(event events.Message) {
	// the Docker Events endpoint will return a struct events.Message
	// https://pkg.go.dev/github.com/docker/docker/api/types/events#Message

	timestamp := time.Unix(event.Time, 0).Format("02-01-2006 15:04:05")
	status := event.Status
	from := event.From
	ID := event.ID[:8]

	message := fmt.Sprintf("%s New status: %s",timestamp, status)
	title := fmt.Sprintf("Container %s from image %s",ID, from)

	log.Infof("Container %s from image %s with new status: %s",ID,from,status)


	delivered := sendPushover(message,title)
	if delivered == true {
		log.Debugf("Message delivered")
	} else  {
		log.Warnf("Message not delivered")
	}
}

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Set log level - defaults to log.InfoLevel)
	//log.SetLevel(log.DebugLevel)

	// Parse flags/config/env variables
	flag.StringVar(&PUSHOVER_APITOKEN, "PUSHOVER_APITOKEN", "", "Pushover's API token")
	flag.StringVar(&PUSHOVER_USER, "PUSHOVER_USER", "", "Pushover's user key")
	flag.Int64Var(&PUSHOVER_DELAY, "PUSHOVER_DELAY",0, "Delay before sending next Pushover message")
	flag.Parse()
  }

func main() {
	// set filters
	// https://docs.docker.com/engine/reference/commandline/events/
	filter := filters.NewArgs()
	filter.Add("type", "container")
	filter.Add("event", "start")
	filter.Add("event", "stop")
	filter.Add("event", "create")
	filter.Add("event", "die")
	filter.Add("event", "kill")

	log.Infof("Starting docker event monitor")
	log.Infof("Using Pushover API Token: %s", PUSHOVER_APITOKEN)
	log.Infof("Using Pushover User Key %s", PUSHOVER_USER)
	log.Infof("Using Pushover delay of %d ms", PUSHOVER_DELAY)

	sendPushover(time.Now().Format("02-01-2006 15:04:05"), "Starting docker event monitor")


	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Panic(err)
	}

  // receives events
	event_chan, errs := cli.Events(context.Background(), types.EventsOptions{Filters: filter})


	for {
		select {
			case err := <-errs:log.Panic(err)
			case event := <-event_chan:
				processEvent(event)
					// Adding a small configurable delay here
					// Sometimes events are pushed through the channel really quickly, but
					// they arrive on the Pushover clients in wrong order (probably due to message delivery time)
					// Consuming the events with a small delay solves the issue
					time.Sleep(time.Duration(PUSHOVER_DELAY) * time.Millisecond)
		}
	}
}
