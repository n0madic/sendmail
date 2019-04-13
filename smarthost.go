package sendmail

import (
	"net"
	"net/smtp"
	"strings"
)

// SendSmarthost message delivery through an external mail server.
func (e *Envelope) SendSmarthost(smarthost, login, password string) <-chan Result {
	results := make(chan Result, len(e.Recipients))
	host, _, err := net.SplitHostPort(smarthost)
	if err != nil {
		results <- Result{FatalLevel, err, "Smarthost", Fields{
			"smarthost": smarthost,
		}}
	} else {
		// Set up authentication information.
		var auth smtp.Auth
		if login != "" && password != "" {
			auth = smtp.PlainAuth("", login, password, host)
		}
		generatedBody, err := e.GenerateMessage()
		if err != nil {
			results <- Result{FatalLevel, err, "Generate message", nil}
		} else {
			fields := Fields{
				"sender":     e.Header.Get("From"),
				"smarthost":  smarthost,
				"recipients": strings.Join(e.Recipients, ","),
			}
			go func() {
				// Connect to the server, authenticate, set the sender and recipient,
				// and send the email all in one step.
				err := smtp.SendMail(smarthost, auth,
					e.Header.Get("From"),
					e.Recipients,
					generatedBody)
				if err == nil {
					results <- Result{InfoLevel, nil, "Send mail OK", fields}
				} else {
					results <- Result{ErrorLevel, err, "", fields}
				}
				close(results)
			}()
		}
	}
	return results
}
