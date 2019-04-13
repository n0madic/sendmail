package sendmail

import (
	"errors"
	"net"
	"net/smtp"
	"strings"
	"sync/atomic"
)

// SendLikeMTA message delivery directly, like Mail Transfer Agent.
func (e *Envelope) SendLikeMTA() <-chan Result {
	var successCount = new(int32)
	mapDomains := make(map[string][]string)
	results := make(chan Result, len(e.recipientsList))
	generatedBody, err := e.GenerateMessage()
	if err != nil {
		results <- Result{FatalLevel, err, "Generate message", nil}
	} else {
		for _, recipient := range e.recipientsList {
			components := strings.Split(recipient.Address, "@")
			mapDomains[components[1]] = append(mapDomains[components[1]], recipient.Address)
		}

		for domain, addresses := range mapDomains {
			rcpts := strings.Join(addresses, ",")
			wg.Add(1)
			go func(domain string, addresses []string) {
				defer wg.Done()
				mxrecords, err := net.LookupMX(domain)
				if err != nil {
					results <- Result{WarnLevel, err, "", Fields{
						"sender":     e.Header.Get("From"),
						"domain":     domain,
						"recipients": rcpts,
					}}
				} else {
					for _, mx := range mxrecords {
						host := strings.TrimSuffix(mx.Host, ".")
						fields := Fields{
							"sender":     e.Header.Get("From"),
							"mx":         host,
							"recipients": rcpts,
						}
						err := smtp.SendMail(host+":25", nil,
							e.Header.Get("From"),
							addresses,
							generatedBody)
						if err == nil {
							results <- Result{InfoLevel, nil, "Send mail OK", fields}
							atomic.AddInt32(successCount, 1)
							return
						}
						results <- Result{WarnLevel, err, "", fields}
					}
				}
			}(domain, addresses)
		}
	}
	go func() {
		wg.Wait()
		fields := Fields{
			"sender":  e.Header.Get("From"),
			"success": *successCount,
			"total":   int32(len(mapDomains)),
		}
		if *successCount == 0 {
			results <- Result{ErrorLevel, errors.New("Failed to deliver to all recipients"), "", fields}
		} else if *successCount != int32(len(mapDomains)) {
			results <- Result{ErrorLevel, errors.New("Failed to deliver to some recipients"), "", fields}
		}
		close(results)
	}()
	return results
}
