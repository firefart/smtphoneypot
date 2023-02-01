package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// The Backend implements SMTP server methods.
type Backend struct{}

func (bkd *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

// A Session is returned after EHLO.
type Session struct {
	Username string
	Password string
	Opts     *smtp.MailOptions
	From     string
	To       string
	Message  *mail.Message
}

func (s *Session) AuthPlain(username, password string) error {
	s.Username = username
	s.Password = password
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.From = from
	s.Opts = opts
	return nil
}

func (s *Session) Rcpt(to string) error {
	s.To = to
	return nil
}

func (s *Session) Data(r io.Reader) error {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return err
	}
	s.Message = msg
	return nil
}

func (s *Session) Reset() {
	if err := s.SaveMail(); err != nil {
		fmt.Println(err)
	}
}

func (s *Session) Logout() error {
	return nil
}

func (s *Session) SaveMail() error {
	filename := cleanFilename(fmt.Sprintf("%s_%s_%d.mail", s.From, s.To, time.Now().Unix()))
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("From: %s\n", s.From)); err != nil {
		return err
	}
	if _, err := f.WriteString(fmt.Sprintf("To: %s\n", s.To)); err != nil {
		return err
	}

	if s.Username != "" {
		if _, err := f.WriteString(fmt.Sprintf("Username: %s\n", s.Username)); err != nil {
			return err
		}
	}
	if s.Password != "" {
		if _, err := f.WriteString(fmt.Sprintf("Password: %s\n", s.Password)); err != nil {
			return err
		}
	}

	if _, err := f.WriteString("\n\nBody:\n"); err != nil {
		return err
	}

	for name, values := range s.Message.Header {
		if _, err := f.WriteString(fmt.Sprintf("%s: %s\n", name, strings.Join(values, ", "))); err != nil {
			return err
		}
	}
	body, err := ioutil.ReadAll(s.Message.Body)
	if err != nil {
		return err
	}
	f.Write(body)

	return nil
}

func cleanFilename(in string) string {
	re := regexp.MustCompile(`[/\\?%*:|"<>]`)
	return re.ReplaceAllString(in, "_")
}

func main() {
	be := &Backend{}

	s := smtp.NewServer(be)

	s.Addr = "0.0.0.0:25"
	s.Domain = "localhost"
	s.ReadTimeout = 1 * time.Minute
	s.WriteTimeout = 1 * time.Minute
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true
	s.EnableAuth(sasl.Login, func(conn *smtp.Conn) sasl.Server {
		return sasl.NewLoginServer(func(username, password string) error {
			sess := conn.Session()
			if sess == nil {
				panic("No session when AUTH is called")
			}

			return sess.AuthPlain(username, password)
		})
	})

	log.Println("Starting server at", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
