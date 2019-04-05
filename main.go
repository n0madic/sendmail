package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"net"
	"net/mail"
	"os"
	"strings"

	"github.com/emersion/go-smtp"
	log "github.com/sirupsen/logrus"
)

var (
	sender  string
	verbose bool
)

func main() {
	flag.StringVar(&sender, "f", "", "Set the envelope sender address.")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging for debugging purposes.")

	flag.Parse()

	if !verbose {
		log.SetLevel(log.WarnLevel)
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		log.Fatal("no stdin input")
	}

	body, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}

	if sender == "" {
		sender = msg.Header.Get("From")
		if sender == "" {
			log.Fatal("Header 'From' not in the message")
		}
	}

	recipients, err := msg.Header.AddressList("To")
	if err != nil {
		log.Fatal(err)
	}

	rcpt := func(field string) []*mail.Address {
		if recipient, err := msg.Header.AddressList(field); err == nil {
			return recipient
		}
		return nil
	}
	recipients = append(recipients, rcpt("Cc")...)
	recipients = append(recipients, rcpt("Bcc")...)

	mapMX := make(map[string][]string)
	for _, recipient := range recipients {
		components := strings.Split(recipient.Address, "@")
		mapMX[components[1]] = append(mapMX[components[1]], recipient.Address)
	}

	var successCount int
	for domain, addresses := range mapMX {
		log.Infof("Send mail to %s", addresses)
		mxrecords, err := net.LookupMX(domain)
		if err != nil {
			log.Warn(err)
		} else {
			for _, mx := range mxrecords {
				host := strings.TrimSuffix(mx.Host, ".")
				log.Infof("Connect with %s", host)
				err := smtp.SendMail(host+":25", nil,
					sender,
					addresses,
					bytes.NewReader(body))
				if err == nil {
					log.Info("Send mail OK")
					successCount++
				} else {
					log.Warn(err)
				}
			}
		}
	}

	if successCount == 0 {
		log.Fatal("Failed to send all messages")
	}
	if successCount != len(mapMX) {
		log.Fatal("Not all messages were able to send")
	}
}
