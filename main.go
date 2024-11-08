package main

import (
	"fmt"
	"io"
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

// compile time check that struct implements the interface
var _ smtp.AuthSession = (*Session)(nil)

func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Anonymous, sasl.External, sasl.OAuthBearer, sasl.Plain}
}

// Auth is the handler for supported authenticators.
func (s *Session) Auth(mech string) (sasl.Server, error) {
	switch mech {
	case sasl.Anonymous:
		return sasl.NewAnonymousServer(func(trace string) error {
			return nil
		}), nil
	case sasl.External:
		return sasl.NewExternalServer(func(identity string) error {
			return nil
		}), nil
	case sasl.OAuthBearer:
		return sasl.NewOAuthBearerServer(func(opts sasl.OAuthBearerOptions) *sasl.OAuthBearerError {
			s.Username = opts.Username
			s.Password = opts.Token
			return nil
		}), nil
	case sasl.Plain:
		return sasl.NewPlainServer(func(identity, username, password string) error {
			s.Username = username
			s.Password = password
			return nil
		}), nil
	default:
		return sasl.NewPlainServer(nil), fmt.Errorf("invalid mech %s", mech)
	}
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.From = from
	s.Opts = opts
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
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

	if s.Message != nil {
		if _, err := f.WriteString("\n\nBody:\n"); err != nil {
			return err
		}
		for name, values := range s.Message.Header {
			if _, err := f.WriteString(fmt.Sprintf("%s: %s\n", name, strings.Join(values, ", "))); err != nil {
				return err
			}
		}
		body, err := io.ReadAll(s.Message.Body)
		if err != nil {
			return err
		}
		f.Write(body)
	}

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
	log.Println("Starting server at", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
