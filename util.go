package sendmail

import (
	"bytes"
	"errors"
	"net/mail"
	"strings"
)

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
