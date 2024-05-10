package main

import (
	"errors"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

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

func sendMail(timestamp time.Time, message string, title string, errCh chan ReporterError) {

	e := ReporterError{
		Reporter: "mail",
	}

	from := config.Reporter.Mail.From
	to := []string{config.Reporter.Mail.To}
	username := config.Reporter.Mail.User
	password := config.Reporter.Mail.Password

	host := config.Reporter.Mail.Host
	port := strconv.Itoa(config.Reporter.Mail.Port)
	address := host + ":" + port

	subject := title
	body := message

	mail := buildEMail(timestamp, from, to, subject, body)

	auth := smtp.PlainAuth("", username, password, host)

	err := smtp.SendMail(address, auth, from, to, []byte(mail))
	if err != nil {
		log.Error().Err(err).Str("reporter", "Mail").Msg("")
		e.Error = errors.New("failed to send mail")
		errCh <- e
		return
	}

}
