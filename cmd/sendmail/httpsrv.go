package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/n0madic/sendmail"
	log "github.com/sirupsen/logrus"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if httpToken != "" && r.Header.Get("Token") != httpToken {
			w.WriteHeader(http.StatusUnauthorized)
			log.Errorf("Attempt to unauthorized send with token %s", r.Header.Get("Token"))
			fmt.Fprint(w, "Unauthorized")
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		}
		var recipients []string
		if r.URL.Query().Get("to") != "" {
			recipients = strings.Split(r.URL.Query().Get("to"), ",")
		}
		envelope, err := sendmail.NewEnvelope(&sendmail.Config{
			Sender:     r.URL.Query().Get("from"),
			Recipients: recipients,
			Subject:    r.URL.Query().Get("subject"),
			Body:       body,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		} else {
			senderDomain := sendmail.GetDomainFromAddress(envelope.Header["From"][0])
			if len(smtpDomains) > 0 && !smtpDomains.Contains(senderDomain) {
				w.WriteHeader(http.StatusUnauthorized)
				log.Errorf("Attempt to unauthorized send with domain %s", senderDomain)
				fmt.Fprint(w, "Unauthorized sender domain")
				return
			}
			errs := envelope.Send()
			for result := range errs {
				switch {
				case result.Level > sendmail.WarnLevel:
					log.WithFields(getLogFields(result.Fields)).Info(result.Message)
					fmt.Fprint(w, "Send mail OK")
				case result.Level == sendmail.WarnLevel:
					log.WithFields(getLogFields(result.Fields)).Warn(result.Error)
				case result.Level < sendmail.WarnLevel:
					log.WithFields(getLogFields(result.Fields)).Warn(result.Error)
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprint(w, result.Error)
				}
			}
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Sorry, only POST method are supported.")
	}
}

func startHTTP(bindAddr string) {
	http.HandleFunc("/", handler)

	log.Info("Starting HTTP server at ", bindAddr)
	log.Fatal(http.ListenAndServe(bindAddr, nil))
}
