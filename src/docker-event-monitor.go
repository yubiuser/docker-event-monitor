package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/gregdel/pushover"

	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type args struct {
	Pushover         bool                `arg:"env:PUSHOVER" default:"false" help:"Enable/Disable Pushover Notification (True/False)"`
	PushoverAPIToken string              `arg:"env:PUSHOVER_APITOKEN" help:"Pushover's API Token/Key"`
	PushoverUserKey  string              `arg:"env:PUSHOVER_USER" help:"Pushover's User Key"`
	Gotify           bool                `arg:"env:GOTIFY" default:"false" help:"Enable/Disable Gotify Notification (True/False)"`
	GotifyURL        string              `arg:"env:GOTIFY_URL" help:"URL of your Gotify server"`
	GotifyToken      string              `arg:"env:GOTIFY_TOKEN" help:"Gotify's App Token"`
	Delay            time.Duration       `arg:"env:DELAY" default:"500ms" help:"Delay before next message is send"`
	FilterStrings    []string            `arg:"env:FILTER,--filter,separate" help:"Filter docker events using Docker syntax."`
	Filter           map[string][]string `arg:"-"`
	LogLevel         string              `arg:"env:LOG_LEVEL" default:"info" help:"Set log level. Use debug for more logging."`
}

// hold the supplied run-time arguments globally
var glb_arguments args

func init() {
	parseArgs()

	configureLogger(glb_arguments.LogLevel)

	if glb_arguments.Pushover {
		if len(glb_arguments.PushoverAPIToken) == 0 {
			log.Fatalln("Pushover enabled. Pushover API token required!")
		}
		if len(glb_arguments.PushoverUserKey) == 0 {
			log.Fatalln("Pushover enabled. Pushover user key required!")
		}
	}
	if glb_arguments.Gotify {
		if len(glb_arguments.GotifyURL) == 0 {
			log.Fatalln("Gotify enabled. Gotify URL required!")
		}
		if len(glb_arguments.GotifyToken) == 0 {
			log.Fatalln("Gotify enabled. Gotify APP token required!")
		}
	}
}

func main() {
	var wg sync.WaitGroup

	log.Infof("Starting docker event monitor")

	if glb_arguments.Pushover {
		log.Infof("Using Pushover API Token %s", glb_arguments.PushoverAPIToken)
		log.Infof("Using Pushover User Key %s", glb_arguments.PushoverUserKey)
	} else {
		log.Info("Pushover notification disabled")
	}

	if glb_arguments.Gotify {
		log.Infof("Using Gotify APP Token %s", glb_arguments.GotifyToken)
		log.Infof("Using Gotify URL %s", glb_arguments.GotifyURL)
	} else {
		log.Info("Gotify notification disabled")
	}
	if glb_arguments.Delay > 0 {
		log.Infof("Using delay of %v", glb_arguments.Delay)
	}

	filterArgs := filters.NewArgs()
	for key, values := range glb_arguments.Filter {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}
	log.Debugf("filterArgs = %v", filterArgs)

	sendNotifications(time.Now().Format("02-01-2006 15:04:05"), "Starting docker event monitor", &wg)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}

	// receives events from the channel
	event_chan, errs := cli.Events(context.Background(), types.EventsOptions{Filters: filterArgs})

	for {
		select {
		case err := <-errs:
			log.Fatal(err)
		case event := <-event_chan:
			processEvent(&event, &wg)
			// Adding a small configurable delay here
			// Sometimes events are pushed through the channel really quickly, but
			// they arrive on the clients in wrong order (probably due to message delivery time)
			// This affects mostly Pushover
			// Consuming the events with a small delay solves the issue
			if glb_arguments.Delay > 0 {
				time.Sleep(glb_arguments.Delay)
			}
		}
	}
}

func sendNotifications(message, title string, wg *sync.WaitGroup) {
	// Sending messages to different services as goroutines concurrently
	// Adding a wait group here to delay execution until all functions return,
	// otherwise the delay in main() would not use its full time

	if glb_arguments.Pushover {
		wg.Add(1)
		go sendPushover(message, title, wg)
	}

	if glb_arguments.Gotify {
		wg.Add(1)
		go sendGotify(message, title, wg)
	}
	wg.Wait()

}

func sendGotify(message, title string, wg *sync.WaitGroup) {
	defer wg.Done()

	response, err := http.PostForm(glb_arguments.GotifyURL+"/message?token="+glb_arguments.GotifyToken,
		url.Values{"message": {message}, "title": {title}})
	if err != nil {
		log.Error(err)
		return
	}

	defer response.Body.Close()

	statusCode := response.StatusCode

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("Gotify response statusCode: %d", statusCode)
	log.Debugf("Gotify response body: %s", string(body))

	// Log non successfull status codes
	if statusCode == 200 {
		log.Debugf("Gotify message delivered")
	} else {
		log.Errorf("Pushing gotify message failed.")
		log.Errorf("Gotify response body: %s", string(body))
	}

}

func sendPushover(message, title string, wg *sync.WaitGroup) {
	defer wg.Done()
	// Create a new pushover app with an API token
	app := pushover.New(glb_arguments.PushoverAPIToken)

	// Create a new recipient (user key)
	recipient := pushover.NewRecipient(glb_arguments.PushoverUserKey)

	// Create the message to send
	pushmessage := pushover.NewMessageWithTitle(message, title)

	// Send the message to the recipient
	response, err := app.SendMessage(pushmessage, recipient)
	if err != nil {
		log.Error(err)
		return
	}
	if response != nil {
		log.Debugf("%s", response)
	}

	if (*response).Status == 1 {
		// Pushover returns 1 if the message request to the API was valid
		// https://pushover.net/api#response
		log.Debugf("Pushover message delivered")
		return
	}

	// if response Status !=1
	log.Errorf("Pushover message not delivered")

}

func processEvent(event *events.Message, wg *sync.WaitGroup) {
	// the Docker Events endpoint will return a struct events.Message
	// https://pkg.go.dev/github.com/docker/docker/api/types/events#Message

	var message string

	// if logging level is Debug, log the event
	log.Debugf("%#v", event)

	//event_timestamp := time.Unix(event.Time, 0).Format("02-01-2006 15:04:05")

	//some events don't return Actor.ID or Actor.Attributes["image"]
	var ID, image string
	if len(event.Actor.ID) > 0 {
		ID = strings.TrimPrefix(event.Actor.ID, "sha256:")[:8] //remove prefix + limit ID legth
	}
	if len(event.Actor.Attributes["image"]) > 0 {
		image = event.Actor.Attributes["image"]
	}

	// Prepare message
	if len(ID) == 0 {
		if len(image) == 0 {
			message = fmt.Sprintf("Object '%s' reported: %s", cases.Title(language.English, cases.Compact).String(event.Type), event.Action)
		} else {
			message = fmt.Sprintf("Object '%s' from image %s reported: %s", cases.Title(language.English, cases.Compact).String(event.Type), image, event.Action)
		}
	} else {
		if len(image) == 0 {
			message = fmt.Sprintf("Object '%s' with ID %s reported: %s", cases.Title(language.English, cases.Compact).String(event.Type), ID, event.Action)
		} else {
			message = fmt.Sprintf("Object '%s' with ID %s from image %s reported: %s", cases.Title(language.English, cases.Compact).String(event.Type), ID, image, event.Action)
		}
	}

	log.Info(message)

	sendNotifications(message, "New Docker Event", wg)

}

func parseArgs() {
	parser := arg.MustParse(&glb_arguments)

	glb_arguments.Filter = make(map[string][]string)

	for _, filter := range glb_arguments.FilterStrings {
		pos := strings.Index(filter, "=")
		if pos == -1 {
			parser.Fail("each filter should be of the form key=value")
		}
		key := filter[:pos]
		val := filter[pos+1:]
		glb_arguments.Filter[key] = append(glb_arguments.Filter[key], val)
	}

}

func configureLogger(LogLevel string) {
	// set log level
	if l, err := log.ParseLevel(LogLevel); err == nil {
		log.SetLevel(l)
	} else {
		log.Fatal(err)
	}

	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	// set log formatting
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})

}
