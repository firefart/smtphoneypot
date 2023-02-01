package main

import (
	"log"
	"strings"

	"github.com/emersion/go-smtp"
)

func main() {
	c, err := smtp.Dial("127.0.0.1:25")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	to := []string{"recipient@example.net"}
	msg := strings.NewReader("To: recipient@example.net\r\n" +
		"Subject: discount Gophers!\r\n" +
		"\r\n" +
		"This is the email body.\r\n")
	err = c.SendMail("sender@example.org", to, msg)
	if err != nil {
		log.Fatal(err)
	}
}
