// Package sendmail is intended for direct sending of emails.
package sendmail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net/mail"
	"os"
	"os/user"
	"sort"
	"strings"
	"sync"
)

var (
	wg sync.WaitGroup
)

// Config of envelope
type Config struct {
	Sender     string
	Recipients []string
	Subject    string
	Body       []byte
	PortSMTP   string
}

// Envelope of message
type Envelope struct {
	*mail.Message
	Recipients []string
	PortSMTP   string
}

// NewEnvelope return new message envelope
func NewEnvelope(config *Config) (Envelope, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(config.Body))
	if err != nil {
		if len(config.Recipients) > 0 {
			msg, err = GetDumbMessage(config.Sender, config.Recipients, config.Body)
		}
		if err != nil {
			return Envelope{}, err
		}
	}

	if config.PortSMTP == "" {
		config.PortSMTP = "25"
	}

	if config.Sender != "" {
		msg.Header["From"] = []string{config.Sender}
	} else {
		config.Sender = msg.Header.Get("From")
		if config.Sender == "" {
			user, err := user.Current()
			if err == nil {
				hostname, err := os.Hostname()
				if err == nil {
					config.Sender = user.Username + "@" + hostname
					msg.Header["From"] = []string{config.Sender}
				}
			}
		}
	}

	if config.Subject != "" {
		msg.Header["Subject"] = []string{"=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(config.Subject))}
	}

	var recipients []string

	if len(config.Recipients) > 0 {
		recipient, err := mail.ParseAddressList(strings.Join(config.Recipients, ","))
		if err == nil {
			recipients = AddressListToSlice(recipient)
		}
	} else {
		recipientsList, err := msg.Header.AddressList("To")
		if err != nil {
			return Envelope{}, err
		}
		rcpt := func(field string) []*mail.Address {
			if recipient, err := msg.Header.AddressList(field); err == nil {
				return recipient
			}
			return nil
		}
		recipientsList = append(recipientsList, rcpt("Cc")...)
		recipientsList = append(recipientsList, rcpt("Bcc")...)
		recipients = AddressListToSlice(recipientsList)
	}

	if len(recipients) == 0 {
		return Envelope{}, errors.New("no recipients listed")
	}

	return Envelope{msg, recipients, config.PortSMTP}, nil
}

// Send message.
// It returns channel for results of send.
// After the end of sending channel are closed.
func (e *Envelope) Send() <-chan Result {
	smartHost := os.Getenv("SENDMAIL_SMART_HOST")
	if smartHost != "" {
		return e.SendSmarthost(
			smartHost,
			os.Getenv("SENDMAIL_SMART_LOGIN"),
			os.Getenv("SENDMAIL_SMART_PASSWORD"),
		)
	}
	return e.SendLikeMTA()
}

// GenerateMessage create body from mail.Message
func (e *Envelope) GenerateMessage() ([]byte, error) {
	if len(e.Header) == 0 {
		return nil, errors.New("empty header")
	}

	buf := bytes.NewBuffer(nil)
	keys := make([]string, 0, len(e.Header))
	for key := range e.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		buf.WriteString(key + ": " + strings.Join(e.Header[key], ",") + "\r\n")
	}
	buf.WriteString("\r\n")

	_, err := buf.ReadFrom(e.Body)
	if err != nil {
		return nil, err
	}

	if !bytes.HasSuffix(buf.Bytes(), []byte("\r\n")) {
		buf.WriteString("\r\n")
	}

	return buf.Bytes(), nil
}
