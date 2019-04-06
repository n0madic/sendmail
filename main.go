package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"net"
	"net/mail"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/emersion/go-smtp"
	log "github.com/sirupsen/logrus"
)

var (
	sender  string
	verbose bool
	wg      sync.WaitGroup
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

	mapDomains := make(map[string][]string)
	for _, recipient := range recipients {
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
						sender,
						addresses,
						bytes.NewReader(body))
					if err == nil {
						log.WithFields(log.Fields{
							"host":       host,
							"recipients": rcpts,
						}).Info("Send mail OK")
						atomic.AddInt32(successCount, 1)
					} else {
						log.WithFields(log.Fields{
							"host":       host,
							"recipients": rcpts,
						}).Warn(err)
					}
				}
			}
		}(domain, addresses)
	}
	wg.Wait()

	if *successCount == 0 {
		log.Fatal("Failed to deliver to all recipients")
	}
	if *successCount != int32(len(mapDomains)) {
		log.WithFields(log.Fields{
			"total":   len(mapDomains),
			"success": *successCount,
		}).Fatal("Failed to deliver to some recipients")
	}
}
