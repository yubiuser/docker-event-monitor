package main

import (
	"context"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/gregdel/pushover"

	"github.com/rs/zerolog"

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
	Mail             bool                `arg:"env:MAIL" default:"false" help:"Enable/Disable Mail (SMTP) Notification (True/False)"`
	MailFrom         string              `arg:"env:MAIL_FROM" help:"your.username@provider.com"`
	MailTo           string              `arg:"env:MAIL_TO" help:"recipient@provider.com"`
	MailUser         string              `arg:"env:MAIL_USER" help:"SMTP Username"`
	MailPassword     string              `arg:"env:MAIL_PASSWORD" help:"SMTP Password"`
	MailPort         int                 `arg:"env:MAIL_PORT" default:"587" help:"SMTP Port"`
	MailHost         string              `arg:"env:MAIL_HOST" help:"SMTP Host"`
	Delay            time.Duration       `arg:"env:DELAY" default:"500ms" help:"Delay before next message is send"`
	FilterStrings    []string            `arg:"env:FILTER,--filter,separate" help:"Filter docker events using Docker syntax."`
	Filter           map[string][]string `arg:"-"`
	LogLevel         string              `arg:"env:LOG_LEVEL" default:"info" help:"Set log level. Use debug for more logging."`
	ServerTag        string              `arg:"env:SERVER_TAG" help:"Prefix to include in the title of notifications. Useful when running docker-event-monitors on multiple machines."`
	Version          bool                `arg:"-v"`
}

// Creating a global logger
var logger zerolog.Logger

// hold the supplied run-time arguments globally
var glb_arguments args

// version information
var (
	version = "n/a"
	commit  = "n/a"
	date    = "n/a"
	gitdate = "n/a"
	branch  = "n/a"
)

func init() {
	parseArgs()
	configureLogger(glb_arguments.LogLevel)

	// if the -v flag was set, print version information and exit
	if glb_arguments.Version {
		logger.Info().
			Str("Version", version).
			Str("Branch", branch).
			Str("Commit", commit).
			Str("Compile_date", date).
			Str("Git_date", gitdate).
			Msg("Version Information")
		os.Exit(0)
	}

	if glb_arguments.Pushover {
		if len(glb_arguments.PushoverAPIToken) == 0 {
			logger.Fatal().Msg("Pushover enabled. Pushover API token required!")
		}
		if len(glb_arguments.PushoverUserKey) == 0 {
			logger.Fatal().Msg("Pushover enabled. Pushover user key required!")
		}
	}
	if glb_arguments.Gotify {
		if len(glb_arguments.GotifyURL) == 0 {
			logger.Fatal().Msg("Gotify enabled. Gotify URL required!")
		}
		if len(glb_arguments.GotifyToken) == 0 {
			logger.Fatal().Msg("Gotify enabled. Gotify APP token required!")
		}
	}
	if glb_arguments.Mail {
		if len(glb_arguments.MailUser) == 0 {
			logger.Fatal().Msg("E-Mail notification enabled. SMTP username required!")
		}
		if len(glb_arguments.MailTo) == 0 {
			logger.Fatal().Msg("E-Mail notification enabled. Recipient address required!")
		}
		if len(glb_arguments.MailFrom) == 0 {
			glb_arguments.MailFrom = glb_arguments.MailUser
		}
		if len(glb_arguments.MailPassword) == 0 {
			logger.Fatal().Msg("E-Mail notification enabled. SMTP Password required!")
		}
		if len(glb_arguments.MailHost) == 0 {
			logger.Fatal().Msg("E-Mail notification enabled. SMTP host address required!")
		}
	}
}

func main() {

	logger.Info().
		Dict("options", zerolog.Dict().
			Dict("reporter", zerolog.Dict().
				Dict("Pushover", zerolog.Dict().
					Bool("enabled", glb_arguments.Pushover).
					Str("PushoverAPIToken", glb_arguments.PushoverAPIToken).
					Str("PushoverUserKey", glb_arguments.PushoverUserKey),
				).
				Dict("Gotify", zerolog.Dict().
					Bool("enabled", glb_arguments.Gotify).
					Str("GotifyURL", glb_arguments.GotifyURL).
					Str("GotifyToken", glb_arguments.GotifyToken),
				).
				Dict("Mail", zerolog.Dict().
					Bool("enabled", glb_arguments.Mail).
					Str("MailFrom", glb_arguments.MailFrom).
					Str("MailTo", glb_arguments.MailTo).
					Str("MailHost", glb_arguments.MailHost).
					Str("MailUser", glb_arguments.MailUser).
					Int("Port", glb_arguments.MailPort),
				),
			).
			Str("Delay", glb_arguments.Delay.String()).
			Str("Loglevel", glb_arguments.LogLevel).
			Str("ServerTag", glb_arguments.ServerTag).
			Str("Filter", strings.Join(glb_arguments.FilterStrings, " ")),
		).
		Dict("version", zerolog.Dict().
			Str("Version", version).
			Str("Branch", branch).
			Str("Commit", commit).
			Str("Compile_date", date).
			Str("Git_date", gitdate),
		).
		Msg("Docker event monitor started")

	timestamp := time.Now()
	startup_message := buildStartupMessage(timestamp)
	sendNotifications(timestamp, startup_message, "Starting docker event monitor")

	filterArgs := filters.NewArgs()
	for key, values := range glb_arguments.Filter {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}
	defer cli.Close()

	// receives events from the channel
	event_chan, errs := cli.Events(context.Background(), types.EventsOptions{Filters: filterArgs})

	for {
		select {
		case err := <-errs:
			logger.Fatal().Err(err).Msg("")
		case event := <-event_chan:
			processEvent(&event)
		}
	}
}

func buildStartupMessage(timestamp time.Time) string {
	var startup_message_builder strings.Builder

	startup_message_builder.WriteString("Docker event monitor started at " + timestamp.Format(time.RFC1123Z) + "\n")

	if glb_arguments.Pushover {
		startup_message_builder.WriteString("Notify via Pushover, using API Token " + glb_arguments.PushoverAPIToken + " and user key " + glb_arguments.PushoverUserKey)
	} else {
		startup_message_builder.WriteString("Pushover notification disabled")
	}

	if glb_arguments.Gotify {
		startup_message_builder.WriteString("\nNotify via Gotify, using URL " + glb_arguments.GotifyURL + " and APP Token " + glb_arguments.GotifyToken)
	} else {
		startup_message_builder.WriteString("\nGotify notification disabled")
	}
	if glb_arguments.Mail {
		startup_message_builder.WriteString("\nNotify via E-Mail from " + glb_arguments.MailFrom + " to " + glb_arguments.MailTo + " using host " + glb_arguments.MailHost + " and port " + strconv.Itoa(glb_arguments.MailPort))
	} else {
		startup_message_builder.WriteString("\nE-Mail notification disabled")
	}

	if glb_arguments.Delay > 0 {
		startup_message_builder.WriteString("\nUsing delay of " + glb_arguments.Delay.String())
	} else {
		startup_message_builder.WriteString("\nDelay disabled")
	}

	startup_message_builder.WriteString("\nLog level: " + glb_arguments.LogLevel)

	if glb_arguments.ServerTag != "" {
		startup_message_builder.WriteString("\nServerTag: " + glb_arguments.ServerTag)
	} else {
		startup_message_builder.WriteString("\nServerTag: none")
	}

	return startup_message_builder.String()
}

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
			sendPushover(message, title)
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
	wg.Wait()

}

func buildEMail(timestamp time.Time, from string, to []string, subject string, body string) string {
	var msg strings.Builder
	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + strings.Join(to, ";") + "\r\n")
	msg.WriteString("Date: " + timestamp.Format(time.RFC1123Z) + "\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("\r\n" + body + "\r\n")

	return msg.String()
}

func sendMail(timestamp time.Time, message string, title string) {

	from := glb_arguments.MailFrom
	to := []string{glb_arguments.MailTo}
	username := glb_arguments.MailUser
	password := glb_arguments.MailPassword

	host := glb_arguments.MailHost
	port := strconv.Itoa(glb_arguments.MailPort)
	address := host + ":" + port

	subject := title
	body := message

	mail := buildEMail(timestamp, from, to, subject, body)

	auth := smtp.PlainAuth("", username, password, host)

	err := smtp.SendMail(address, auth, from, to, []byte(mail))
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Mail").Msg("")
		return
	}
}

func sendGotify(message string, title string) {

	response, err := http.PostForm(glb_arguments.GotifyURL+"/message?token="+glb_arguments.GotifyToken,
		url.Values{"message": {message}, "title": {title}})
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Gotify").Msg("")
		return
	}

	defer response.Body.Close()

	statusCode := response.StatusCode

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Error().Err(err).Str("reporter", "Gotify").Msg("")
		return
	}

	logger.Debug().Str("reporter", "Gotify").Msgf("Gotify response statusCode: %d", statusCode)
	logger.Debug().Str("reporter", "Gotify").Msgf("Gotify response body: %s", string(body))

	// Log non successfull status codes
	if statusCode == 200 {
		logger.Debug().Str("reporter", "Gotify").Msgf("Gotify message delivered")
	} else {
		logger.Error().Str("reporter", "Gotify").Msgf("Pushing gotify message failed.")
		logger.Error().Str("reporter", "Gotify").Msgf("Gotify response body: %s", string(body))
	}

}

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

func processEvent(event *events.Message) {
	// the Docker Events endpoint will return a struct events.Message
	// https://pkg.go.dev/github.com/docker/docker/api/types/events#Message

	var msg_builder, title_builder strings.Builder
	var ActorID, ActorImage, ActorName, TitleID string

	// Adding a small configurable delay here
	// Sometimes events are pushed through the event channel really quickly, but they arrive on the notification clients in
	// wrong order (probably due to message delivery time), e.g. Pushover is susceptible for this.
	// Finishing this function not before a certain time before draining the next event from the event channel in main() solves the issue
	timer := time.NewTimer(glb_arguments.Delay)

	// if logging level is Debug, log the event
	logger.Debug().Msgf("%#v", event)

	//some events don't return Actor.ID, Actor.Attributes["image"] or Actor.Attributes["name"]
	if len(event.Actor.ID) > 0 && strings.HasPrefix(event.Actor.ID, "sha256:") {
		ActorID = strings.TrimPrefix(event.Actor.ID, "sha256:")[:8] //remove prefix + limit ActorID legth
	}
	if len(event.Actor.Attributes["image"]) > 0 {
		ActorImage = event.Actor.Attributes["image"]
	} else {
		// try to recover image name from org.opencontainers.image info
		if len(event.Actor.Attributes["org.opencontainers.image.title"]) > 0 && len(event.Actor.Attributes["org.opencontainers.image.version"]) > 0 {
			ActorImage = event.Actor.Attributes["org.opencontainers.image.title"] + ":" + event.Actor.Attributes["org.opencontainers.image.version"]
		}
	}
	if len(event.Actor.Attributes["name"]) > 0 {
		// in case the ActorName is only an hash
		if strings.HasPrefix(event.Actor.Attributes["name"], "sha256:") {
			ActorName = strings.TrimPrefix(event.Actor.Attributes["name"], "sha256:")[:8] //remove prefix + limit ActorName legth
		} else {
			ActorName = event.Actor.Attributes["name"]
		}
	}

	// Check possible image and container name
	// The order of the checks is important, because we want name rather than ActorID
	// as identifier in the title
	if len(ActorID) > 0 {
		msg_builder.WriteString("ID: " + ActorID + "\n")
		TitleID = ActorID
	}
	if len(ActorImage) > 0 {
		msg_builder.WriteString("Image: " + ActorImage + "\n")
		// Not using ActorImage as possible title, because it's too long
	}
	if len(ActorName) > 0 {
		msg_builder.WriteString("Name: " + ActorName + "\n")
		TitleID = ActorName
	}

	// Build title
	title_builder.WriteString(cases.Title(language.English, cases.Compact).String(string(event.Type)))
	if len(TitleID) > 0 {
		title_builder.WriteString(" " + TitleID)
	}
	title_builder.WriteString(": " + string(event.Action))

	// Get event timestamp
	timestamp := time.Unix(event.Time, 0)
	msg_builder.WriteString("Time: " + timestamp.Format(time.RFC1123Z) + "\n")

	// Append possible docker compose context
	if len(event.Actor.Attributes["com.docker.compose.project.working_dir"]) > 0 {
		msg_builder.WriteString("Docker compose context: " + event.Actor.Attributes["com.docker.compose.project.working_dir"] + "\n")
	}
	if len(event.Actor.Attributes["com.docker.compose.service"]) > 0 {
		msg_builder.WriteString("Docker compose service: " + event.Actor.Attributes["com.docker.compose.service"] + "\n")
	}

	// Build message and title
	title := title_builder.String()
	message := strings.TrimRight(msg_builder.String(), "\n")

	// Log message
	logger.Info().
		Str("eventType", string(event.Type)).
		Str("ActorID", ActorID).
		Str("eventAction", string(event.Action)).
		Str("ActorImage", ActorImage).
		Str("ActorName", ActorName).
		Str("DockerComposeContext", event.Actor.Attributes["com.docker.compose.project.working_dir"]).
		Str("DockerComposeService", event.Actor.Attributes["com.docker.compose.service"]).
		Msg(title)

	// send notifications to various reporters
	// function will finish when all reporters finished
	sendNotifications(timestamp, message, title)

	// block function until time (delay) triggers
	// if sendNotifications is faster than the delay, function blocks here until delay is over
	// if sendNotifications takes longer than the delay, trigger already fired and no delay is added
	<-timer.C

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
	// UNIX Time is faster and smaller than most timestamps
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Change logging level when debug flag is set
	if LogLevel == "debug" {
		logger = zerolog.New(os.Stdout).
			Level(zerolog.DebugLevel).
			With().
			Timestamp().
			Str("service", "docker event monitor").
			Logger()
	} else {
		logger = zerolog.New(os.Stdout).
			Level(zerolog.InfoLevel).
			With().
			Str("service", "docker event monitor").
			Timestamp().
			Logger()
	}
}
