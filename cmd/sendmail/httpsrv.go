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
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err)
		}
		var recipients []string
		if r.URL.Query().Get("to") != "" {
			recipients = strings.Split(r.URL.Query().Get("to"), ",")
		}
		envelope, err := sendmail.NewEnvelope(
			r.URL.Query().Get("from"),
			recipients,
			r.URL.Query().Get("subject"),
			body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			err = envelope.Send()
		}
		fmt.Fprint(w, err)
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
