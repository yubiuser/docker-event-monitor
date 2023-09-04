package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	PushoverDelay    time.Duration       `arg:"env:PUSHOVER_DELAY" default:"500ms" help:"Delay before next Pushover message is send"`
	Gotify           bool                `arg:"env:GOTIFY" default:"false" help:"Enable/Disable Gotify Notification (True/False)"`
	GotifyURL        string              `arg:"env:GOTIFY_URL" help:"URL of your Gotify server"`
	GotifyToken      string              `arg:"env:GOTIFY_TOKEN" help:"Gotify's App Token"`
	FilterStrings    []string            `arg:"env:FILTER,--filter,separate" help:"Filter docker events using Docker syntax."`
	Filter           map[string][]string `arg:"-"`
	LogLevel         string              `arg:"env:LOG_LEVEL" default:"info" help:"Set log level. Use debug for more logging."`
}

func main() {
	args := parseArgs()

	log.Infof("Starting docker event monitor")

	if args.Pushover {
		log.Infof("Using Pushover API Token %s", args.PushoverAPIToken)
		log.Infof("Using Pushover User Key %s", args.PushoverUserKey)
		log.Infof("Using Pushover delay of %v", args.PushoverDelay)
	} else {
		log.Info("Pushover notification disabled")
	}

	if args.Gotify {
		log.Infof("Using Gotify APP Token %s", args.GotifyToken)
		log.Infof("Using Gotify URL %s", args.GotifyURL)
	} else {
		log.Info("Gotify notification disabled")
	}

	if args.Pushover {
		sendPushover(args, time.Now().Format("02-01-2006 15:04:05"), "Starting docker event monitor")
	}
	if args.Gotify {
		sendGotify(args, time.Now().Format("02-01-2006 15:04:05"), "Starting docker event monitor")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Panic(err)
	}

	filterArgs := filters.NewArgs()
	for key, values := range args.Filter {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	log.Debugf("filterArgs = %v", filterArgs)

	// receives events
	event_chan, errs := cli.Events(context.Background(), types.EventsOptions{Filters: filterArgs})

	for {
		select {
		case err := <-errs:
			log.Panic(err)
		case event := <-event_chan:
			processEvent(args, event)
			// Adding a small configurable delay here
			// Sometimes events are pushed through the channel really quickly, but
			// they arrive on the Pushover clients in wrong order (probably due to message delivery time)
			// Consuming the events with a small delay solves the issue
			if args.Pushover {
				time.Sleep(args.PushoverDelay)
			}
		}
	}
}

func sendGotify(args args, message, title string) {
	http.PostForm(args.GotifyURL+"/message?token="+args.GotifyToken,
		url.Values{"message": {message}, "title": {title}})

}
func sendPushover(args args, message, title string) bool {

	// Create a new pushover app with an API token
	app := pushover.New(args.PushoverAPIToken)

	// Create a new recipient (user key)
	recipient := pushover.NewRecipient(args.PushoverUserKey)

	// Create the message to send
	pushmessage := pushover.NewMessageWithTitle(message, title)

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

func processEvent(args args, event events.Message) {
	// the Docker Events endpoint will return a struct events.Message
	// https://pkg.go.dev/github.com/docker/docker/api/types/events#Message

	var message string

	// if logging level is Debug or higher, log the event
	if log.IsLevelEnabled(log.DebugLevel) {
		log.Debugf("%#v", event)
	}

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

	if args.Pushover {
		delivered := sendPushover(args, message, "New Docker Event")
		if delivered {
			log.Debugf("Pushover message delivered")
		} else {
			log.Warnf("Pushover message not delivered")
		}
	}

	if args.Gotify {
		sendGotify(args, message, "New Docker Event")
	}
}

func parseArgs() args {
	var args args
	parser := arg.MustParse(&args)

	configureLogger(args.LogLevel)

	args.Filter = make(map[string][]string)

	for _, filter := range args.FilterStrings {
		pos := strings.Index(filter, "=")
		if pos == -1 {
			parser.Fail("each filter should be of the form key=value")
		}
		key := filter[:pos]
		val := filter[pos+1:]
		args.Filter[key] = append(args.Filter[key], val)
	}

	return args
}

func configureLogger(LogLevel string) {
	// set log level
	if l, err := log.ParseLevel(LogLevel); err == nil {
		log.SetLevel(l)
	} else {
		log.Panic(err)
	}

	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	// set log formatting
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})

}
