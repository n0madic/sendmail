// Package sendmail is intended for direct sending of emails.
package sendmail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"os/user"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	wg sync.WaitGroup
)

// Envelope of message
type Envelope struct {
	*mail.Message
	recipientsList []*mail.Address
}

// Level type of result
type Level uint32

const (
	// FatalLevel level.
	FatalLevel Level = iota
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the application.
	InfoLevel
)

// Fields type, used for expand information.
type Fields map[string]interface{}

// Result of send
type Result struct {
	Level   Level
	Error   error
	Message string
	Fields  Fields
}

// NewEnvelope return new message envelope
func NewEnvelope(sender string, recipients []string, subject string, body []byte) (Envelope, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		if len(recipients) > 0 {
			msg, err = GetDumbMessage(sender, recipients, body)
		}
		if err != nil {
			return Envelope{}, err
		}
	}

	if sender != "" {
		msg.Header["From"] = []string{sender}
	} else {
		sender = msg.Header.Get("From")
		if sender == "" {
			user, err := user.Current()
			if err == nil {
				hostname, err := os.Hostname()
				if err == nil {
					sender = user.Username + "@" + hostname
					msg.Header["From"] = []string{sender}
				}
			}
		}
	}

	if subject != "" {
		msg.Header["Subject"] = []string{"=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject))}
	}

	var recipientsList []*mail.Address

	if len(recipients) > 0 {
		recipient, err := mail.ParseAddressList(strings.Join(recipients, ","))
		if err == nil {
			recipientsList = recipient
		}
	} else {
		recipientsList, err = msg.Header.AddressList("To")
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
	}

	if len(recipientsList) == 0 {
		return Envelope{}, errors.New("No recipients listed")
	}

	return Envelope{msg, recipientsList}, nil
}

// GetDumbMessage create simple mail.Message from raw data
func GetDumbMessage(sender string, recipients []string, body []byte) (*mail.Message, error) {
	if len(recipients) == 0 {
		return nil, errors.New("Empty recipients list")
	}
	buf := bytes.NewBuffer(nil)
	if sender != "" {
		buf.WriteString("From: " + sender + "\r\n")
	}
	buf.WriteString("To: " + strings.Join(recipients, ",") + "\r\n")
	buf.WriteString("\r\n")
	buf.Write(body)
	buf.WriteString("\r\n")
	return mail.ReadMessage(buf)
}

// Send message.
// It returns channel for results of send.
// After the end of sending channel are closed.
func (e *Envelope) Send() <-chan Result {
	return e.SendLikeMTA()
}

// SendLikeMTA message delivery directly, like Mail Transfer Agent.
func (e *Envelope) SendLikeMTA() <-chan Result {
	var successCount = new(int32)
	mapDomains := make(map[string][]string)
	results := make(chan Result, len(e.recipientsList))
	generatedBody, err := e.GenerateMessage()
	if err != nil {
		results <- Result{FatalLevel, err, "Generate message", nil}
	} else {
		for _, recipient := range e.recipientsList {
			components := strings.Split(recipient.Address, "@")
			mapDomains[components[1]] = append(mapDomains[components[1]], recipient.Address)
		}

		for domain, addresses := range mapDomains {
			rcpts := strings.Join(addresses, ",")
			wg.Add(1)
			go func(domain string, addresses []string) {
				defer wg.Done()
				mxrecords, err := net.LookupMX(domain)
				if err != nil {
					results <- Result{WarnLevel, err, "", Fields{
						"sender":     e.Header.Get("From"),
						"domain":     domain,
						"recipients": rcpts,
					}}
				} else {
					for _, mx := range mxrecords {
						host := strings.TrimSuffix(mx.Host, ".")
						fields := Fields{
							"sender":     e.Header.Get("From"),
							"mx":         host,
							"recipients": rcpts,
						}
						err := smtp.SendMail(host+":25", nil,
							e.Header.Get("From"),
							addresses,
							generatedBody)
						if err == nil {
							results <- Result{InfoLevel, nil, "Send mail OK", fields}
							atomic.AddInt32(successCount, 1)
							return
						}
						results <- Result{WarnLevel, err, "", fields}
					}
				}
			}(domain, addresses)
		}
	}
	go func() {
		wg.Wait()
		fields := Fields{
			"sender":  e.Header.Get("From"),
			"success": *successCount,
			"total":   int32(len(mapDomains)),
		}
		if *successCount == 0 {
			results <- Result{ErrorLevel, errors.New("Failed to deliver to all recipients"), "", fields}
		} else if *successCount != int32(len(mapDomains)) {
			results <- Result{ErrorLevel, errors.New("Failed to deliver to some recipients"), "", fields}
		}
		close(results)
	}()
	return results
}

// GenerateMessage create body from mail.Message
func (e *Envelope) GenerateMessage() ([]byte, error) {
	if len(e.Header) == 0 {
		return nil, errors.New("Empty header")
	}
	buf := bytes.NewBuffer(nil)
	for key, value := range e.Header {
		buf.WriteString(key + ": " + strings.Join(value, ",") + "\r\n")
	}
	_, err := buf.ReadFrom(e.Body)
	if err != nil {
		return nil, err
	}
	buf.WriteString("\r\n")
	return buf.Bytes(), nil
}
