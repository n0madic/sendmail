package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"io"
	"net"
	"net/mail"
	"os"
	"os/user"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/emersion/go-smtp"
	log "github.com/sirupsen/logrus"
)

var (
	body        []byte
	extractRcpt bool
	ignoreDot   bool
	recipients  []*mail.Address
	sender      string
	subject     string
	verbose     bool
	wg          sync.WaitGroup
)

func main() {
	flag.BoolVar(&ignoreDot, "i", false, "When reading a message from standard input, don't treat a line with only a . character as the end of input.")
	flag.BoolVar(&extractRcpt, "t", false, "Extract recipients from message headers. This requires that no recipients be specified on the command line.")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging for debugging purposes.")
	flag.StringVar(&sender, "f", "", "Set the envelope sender address.")
	flag.StringVar(&subject, "s", "", "Specify subject on command line.")

	flag.Parse()

	if !verbose {
		log.SetLevel(log.WarnLevel)
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		log.Fatal("no stdin input")
	}

	bio := bufio.NewReader(os.Stdin)
	for {
		line, err := bio.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if !ignoreDot && bytes.Equal(bytes.Trim(line, "\n"), []byte(".")) {
			break
		}
		body = append(body, line...)
	}
	if len(body) == 0 {
		log.Fatal("Empty message body")
	}

	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		if sender != "" && flag.NArg() > 0 {
			log.Info(err)
			buf := bytes.NewBuffer(nil)
			buf.WriteString("From: " + sender + "\r\n")
			buf.WriteString("To: " + strings.Join(flag.Args(), ",") + "\r\n")
			if subject != "" {
				var coder = base64.StdEncoding
				buf.WriteString("Subject: =?UTF-8?B?" +
					coder.EncodeToString([]byte(subject)) +
					"?=\r\n")
			}
			buf.WriteString("\r\n")
			buf.Write(body)
			buf.WriteString("\r\n")
			msg, err = mail.ReadMessage(buf)
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	if sender == "" {
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
				}
			}
		}
	}

	if flag.NArg() > 0 {
		recipient, err := mail.ParseAddressList(strings.Join(flag.Args(), ","))
		if err == nil {
			recipients = recipient
		}
	} else if extractRcpt {
		recipients, err = msg.Header.AddressList("To")
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
	}

	if len(recipients) == 0 {
		log.Fatal("No recipients listed")
	}

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
							"mx":         host,
							"recipients": rcpts,
						}).Info("Send mail OK")
						atomic.AddInt32(successCount, 1)
						break
					} else {
						log.WithFields(log.Fields{
							"mx":         host,
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
