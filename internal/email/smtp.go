package email

import (
	"fmt"
	"net/smtp"
)

type SMTPSender struct {
	Host     string
	Port     int
	Username string
	Password string
}

func (s *SMTPSender) Send(to, subject, body string) error {
	auth := smtp.PlainAuth(
		"",
		s.Username,
		s.Password,
		s.Host,
	)

	message := []byte(
		"From: " + s.Username + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			body + "\r\n",
	)

	address := fmt.Sprintf("%s:%d", s.Host, s.Port)

	return smtp.SendMail(
		address,
		auth,
		s.Username,
		[]string{to},
		message,
	)
}
