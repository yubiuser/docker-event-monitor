package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/rs/zerolog"
)

type args struct {
	Pushover          bool                `arg:"env:PUSHOVER" default:"false" help:"Enable/Disable Pushover Notification (True/False)"`
	PushoverAPIToken  string              `arg:"env:PUSHOVER_APITOKEN" help:"Pushover's API Token/Key"`
	PushoverUserKey   string              `arg:"env:PUSHOVER_USER" help:"Pushover's User Key"`
	Gotify            bool                `arg:"env:GOTIFY" default:"false" help:"Enable/Disable Gotify Notification (True/False)"`
	GotifyURL         string              `arg:"env:GOTIFY_URL" help:"URL of your Gotify server"`
	GotifyToken       string              `arg:"env:GOTIFY_TOKEN" help:"Gotify's App Token"`
	Mail              bool                `arg:"env:MAIL" default:"false" help:"Enable/Disable Mail (SMTP) Notification (True/False)"`
	MailFrom          string              `arg:"env:MAIL_FROM" help:"your.username@provider.com"`
	MailTo            string              `arg:"env:MAIL_TO" help:"recipient@provider.com"`
	MailUser          string              `arg:"env:MAIL_USER" help:"SMTP Username"`
	MailPassword      string              `arg:"env:MAIL_PASSWORD" help:"SMTP Password"`
	MailPort          int                 `arg:"env:MAIL_PORT" default:"587" help:"SMTP Port"`
	MailHost          string              `arg:"env:MAIL_HOST" help:"SMTP Host"`
	Mattermost        bool                `arg:"env:MATTERMOST" default:"false" help:"Enable/Disable Mattermost Notification (True/False)"`
	MattermostURL     string              `arg:"env:MATTERMOST_URL" help:"URL of your Mattermost incoming webhook"`
	MattermostChannel string              `arg:"env:MATTERMOST_CHANNEL" help:"Mattermost channel to post in"`
	MattermostUser    string              `arg:"env:MATTERMOST_USER" default:"Docker Event Monitor" help:"Mattermost user to post as"`
	Delay             time.Duration       `arg:"env:DELAY" default:"500ms" help:"Delay before next message is send"`
	FilterStrings     []string            `arg:"env:FILTER,--filter,separate" help:"Filter docker events using Docker syntax."`
	Filter            map[string][]string `arg:"-"`
	ExcludeStrings    []string            `arg:"env:EXCLUDE,--exclude,separate" help:"Exclude docker events using Docker syntax."`
	Exclude           map[string][]string `arg:"-"`
	LogLevel          string              `arg:"env:LOG_LEVEL" default:"info" help:"Set log level. Use debug for more logging."`
	ServerTag         string              `arg:"env:SERVER_TAG" help:"Prefix to include in the title of notifications. Useful when running docker-event-monitors on multiple machines."`
	Version           bool                `arg:"-v" help:"Print version information."`
}

// Creating a global logger
var logger zerolog.Logger

// hold the supplied run-time arguments globally
var glb_arguments args

// version information, are injected during build process
var (
	version string = "n/a"
	commit  string = "n/a"
	date    string = "0"
	gitdate string = "0"
	branch  string = "n/a"
)

func init() {
	parseArgs()
	configureLogger(glb_arguments.LogLevel)

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
	if glb_arguments.Mattermost {
		if len(glb_arguments.MattermostURL) == 0 {
			logger.Fatal().Msg("Mattermost enabled. Mattermost URL required!")
		}
	}
}

func main() {
	// if the -v flag was set, print version information and exit
	if glb_arguments.Version {
		printVersion()
	}

	// log all supplied arguments
	logArguments()

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
		logger.Fatal().Err(err).Msg("Failed to create new docker client")
	}
	defer cli.Close()

	// receives events from the channel
	event_chan, errs := cli.Events(context.Background(), types.EventsOptions{Filters: filterArgs})

	for {
		select {
		case err := <-errs:
			logger.Fatal().Err(err).Msg("")
		case event := <-event_chan:
			// if logging level is debug, log the event
			logger.Debug().
				Interface("event", event).Msg("")

			// Check if event should be exlcuded from reporting
			if len(glb_arguments.Exclude) > 0 {
				logger.Debug().Msg("Performing check for event exclusion")
				if excludeEvent(event) {
					break //breaks out of the select and waits for the next event to arrive
				}
			}
			processEvent(event)
		}
	}
}

func parseArgs() {
	parser := arg.MustParse(&glb_arguments)

	// Parse (include) filters
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

	// Parse exclude filters
	glb_arguments.Exclude = make(map[string][]string)

	for _, exclude := range glb_arguments.ExcludeStrings {
		pos := strings.Index(exclude, "=")
		if pos == -1 {
			parser.Fail("each filter should be of the form key=value")
		}
		//trim whitespaces
		key := strings.TrimSpace(exclude[:pos])
		val := exclude[pos+1:]
		glb_arguments.Exclude[key] = append(glb_arguments.Exclude[key], val)
	}

}

func configureLogger(LogLevel string) {
	// Configure time/timestamp format
	zerolog.TimeFieldFormat = time.RFC1123Z

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
