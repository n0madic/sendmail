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

	for _, recipient := range recipients {
		components := strings.Split(recipient.Address, "@")
		mxrecords, err := net.LookupMX(components[1])
		if err != nil {
			log.Fatal(err)
		}
		for _, mx := range mxrecords {
			host := strings.TrimSuffix(mx.Host, ".")
			log.Infof("Connect to %s", host)

			err := smtp.SendMail(host+":25", nil, sender, []string{recipient.Address}, bytes.NewReader(body))
			if err == nil {
				log.Info("Send mail OK")
				os.Exit(0)
			}
			log.Warn(err)
		}
	}
	log.Fatal("Could not send message")
}
