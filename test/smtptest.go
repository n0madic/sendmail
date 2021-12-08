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

var once sync.Once

// PortSMTP for tests
const PortSMTP = "2525"

// Login handles a login command with username and password.
func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return &Session{}, nil
}

// AnonymousLogin allowed
func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &Session{}, nil
}

// A Session is returned after successful login.
type Session struct{}

// Mail check sender
func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	if from != "sender@localhost" {
		return fmt.Errorf("unknow sender %s", from)
	}
	return nil
}

// Rcpt check recipients
func (s *Session) Rcpt(to string) error {
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
