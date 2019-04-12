// Standalone drop-in replacement for sendmail with direct send
package main

import (
	"bufio"
	"bytes"
	"flag"
	"io"
	"os"
	"sync"

	"github.com/n0madic/sendmail"
	log "github.com/sirupsen/logrus"
)

var (
	body      []byte
	httpMode  bool
	httpBind  string
	ignored   bool
	ignoreDot bool
	sender    string
	smtpMode  bool
	smtpBind  string
	subject   string
	verbose   bool
	wg        sync.WaitGroup
)

func main() {
	flag.BoolVar(&ignored, "t", true, "Extract recipients from message headers. IGNORED")
	flag.BoolVar(&ignoreDot, "i", false, "When reading a message from standard input, don't treat a line with only a . character as the end of input.")
	flag.BoolVar(&verbose, "v", false, "Enable verbose logging for debugging purposes.")
	flag.StringVar(&sender, "f", "", "Set the envelope sender address.")
	flag.StringVar(&subject, "s", "", "Specify subject on command line.")

	flag.BoolVar(&httpMode, "http", false, "Enable HTTP server mode.")
	flag.StringVar(&httpBind, "httpBind", "localhost:8080", "TCP address to HTTP listen on.")
	flag.BoolVar(&smtpMode, "smtp", false, "Enable SMTP server mode.")
	flag.StringVar(&smtpBind, "smtpBind", "localhost:25", "TCP or Unix address to SMTP listen on.")

	flag.Parse()

	if !verbose {
		log.SetLevel(log.WarnLevel)
	}

	if httpMode || smtpMode {
		if httpMode {
			go startHTTP(httpBind)
		}
		if smtpMode {
			go startSMTP(smtpBind)
		}
		select {}
	} else {
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
		envelope, err := sendmail.NewEnvelope(sender, flag.Args(), subject, body)
		if err != nil {
			log.Fatal(err)
		}
		errs := envelope.Send()
		for result := range errs {
			switch {
			case result.Level > sendmail.WarnLevel:
				log.WithFields(getLogFields(result.Fields)).Info(result.Message)
			case result.Level == sendmail.WarnLevel:
				log.WithFields(getLogFields(result.Fields)).Warn(result.Error)
			case result.Level < sendmail.WarnLevel:
				log.WithFields(getLogFields(result.Fields)).Fatal(result.Error)
			}
		}
	}
}

func getLogFields(fields sendmail.Fields) log.Fields {
	logFields := log.Fields{}
	for k, v := range fields {
		logFields[k] = v
	}
	return logFields
}
