package test

import (
	"fmt"
	"io"
	"log"
	"net/mail"
	"sync"

	smtp "github.com/emersion/go-smtp"
)

// The Backend implements SMTP server methods.
type Backend struct{}

func (bkd *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

var once sync.Once

// PortSMTP for tests
const PortSMTP = "2525"

// A Session is returned after successful login.
type Session struct{}

// AuthPlain check stub
func (s *Session) AuthPlain(username, password string) error {
	return nil
}

// Mail check sender
func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	if from != "sender@localhost" {
		return fmt.Errorf("unknow sender %s", from)
	}
	return nil
}

// Rcpt check recipients
func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	if to != "recipient@localhost" {
		return fmt.Errorf("unknow recipient %s", to)
	}
	return nil
}

// Data receives the message body
func (s *Session) Data(r io.Reader) error {
	_, err := mail.ReadMessage(r)
	return err
}

// Reset session
func (s *Session) Reset() {}

// Logout session
func (s *Session) Logout() error {
	return nil
}

// StartSMTP server
func StartSMTP() {
	once.Do(func() {
		s := smtp.NewServer(&Backend{})
		s.Addr = "localhost:" + PortSMTP
		log.Fatalln(s.ListenAndServe())
	})
}
