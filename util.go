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

// AddressListToSlice convert mail.Address list to slice of strings
func AddressListToSlice(list []*mail.Address) (slice []string) {
	for _, rcpt := range list {
		slice = append(slice, rcpt.Address)
	}
	return
}

// GetDomainFromAddress extract domain from email address
func GetDomainFromAddress(address string) string {
	components := strings.Split(address, "@")
	if len(components) == 2 {
		return components[1]
	}
	return ""
}
