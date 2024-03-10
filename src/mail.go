package main

import (
	"net/smtp"
	"strconv"
	"strings"
	"time"
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
