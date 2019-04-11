package sendmail

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/mail"
	"os"
	"os/user"
	"strings"
	"sync"
	"sync/atomic"

	smtp "github.com/emersion/go-smtp"
	log "github.com/sirupsen/logrus"
)

var (
	wg sync.WaitGroup
)

// Envelope of message
type Envelope struct {
	*mail.Message
	recipientsList []*mail.Address
}

// NewEnvelope return new message envelope
func NewEnvelope(sender string, recipients []string, subject string, body []byte) (Envelope, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		if len(recipients) > 0 {
			log.Info(err)
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
			log.Warn("No header 'From' in the message")
			user, err := user.Current()
			if err != nil {
				log.Warn(err)
			} else {
				hostname, err := os.Hostname()
				if err != nil {
					log.Warn(err)
				} else {
					sender = user.Username + "@" + hostname
					log.Info("Use <" + sender + "> as sender address")
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

// SendLikeMTA message delivery directly, like Mail Transfer Agent
func (e *Envelope) SendLikeMTA() (err error) {
	generatedBody, err := e.GenerateMessage()
	if err != nil {
		return err
	}

	mapDomains := make(map[string][]string)
	for _, recipient := range e.recipientsList {
		components := strings.Split(recipient.Address, "@")
		mapDomains[components[1]] = append(mapDomains[components[1]], recipient.Address)
	}

	var successCount = new(int32)
	for domain, addresses := range mapDomains {
		rcpts := strings.Join(addresses, ",")
		wg.Add(1)
		go func(domain string, addresses []string) {
			defer wg.Done()
			mxrecords, err := net.LookupMX(domain)
			if err != nil {
				log.WithField("recipients", rcpts).Warn(err)
			} else {
				for _, mx := range mxrecords {
					host := strings.TrimSuffix(mx.Host, ".")
					err := smtp.SendMail(host+":25", nil,
						e.Header.Get("From"),
						addresses,
						generatedBody)
					if err == nil {
						log.WithFields(log.Fields{
							"mx":         host,
							"recipients": rcpts,
						}).Info("Send mail OK")
						atomic.AddInt32(successCount, 1)
						return
					}
					log.WithFields(log.Fields{
						"mx":         host,
						"recipients": rcpts,
					}).Warn(err)
				}
			}
		}(domain, addresses)
	}
	wg.Wait()
	if *successCount == 0 {
		err = errors.New("Failed to deliver to all recipients")
	} else if *successCount != int32(len(mapDomains)) {
		err = errors.New("Failed to deliver to some recipients")
	}
	return err
}

// GenerateMessage create body from mail.Message
func (e *Envelope) GenerateMessage() (io.Reader, error) {
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
	return buf, nil
}
