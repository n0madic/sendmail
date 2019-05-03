package sendmail

import (
	"errors"
	"net"
	"net/smtp"
	"strings"
	"sync/atomic"
)

var portSMTP = "25"

// SendLikeMTA message delivery directly, like Mail Transfer Agent.
func (e *Envelope) SendLikeMTA() <-chan Result {
	var successCount = new(int32)
	mapDomains := make(map[string][]string)
	results := make(chan Result, len(e.Recipients))
	generatedBody, err := e.GenerateMessage()
	if err != nil {
		results <- Result{FatalLevel, err, "Generate message", nil}
	} else {
		for _, recipient := range e.Recipients {
			domain := GetDomainFromAddress(recipient)
			mapDomains[domain] = append(mapDomains[domain], recipient)
		}

		for domain, addresses := range mapDomains {
			rcpts := strings.Join(addresses, ",")
			wg.Add(1)
			go func(domain string, addresses []string) {
				defer wg.Done()
				var hostList []string
				mxrecords, err := net.LookupMX(domain)
				if err != nil {
					results <- Result{WarnLevel, err, "LookupMX", Fields{
						"sender":     e.Header.Get("From"),
						"domain":     domain,
						"recipients": rcpts,
					}}
					// Fallback to A records
					ips, err := net.LookupIP(domain)
					if err != nil {
						results <- Result{WarnLevel, err, "LookupIP", Fields{
							"sender":     e.Header.Get("From"),
							"domain":     domain,
							"recipients": rcpts,
						}}
					} else {
						for _, ip := range ips {
							host := strings.TrimSuffix(ip.String(), ".")
							hostList = append(hostList, host)
						}
					}
				} else {
					for _, mx := range mxrecords {
						host := strings.TrimSuffix(mx.Host, ".")
						hostList = append(hostList, host)
					}
				}
				if len(hostList) == 0 {
					results <- Result{ErrorLevel, errors.New("MX not found"), "Lookup", Fields{
						"sender":     e.Header.Get("From"),
						"domain":     domain,
						"recipients": rcpts,
					}}
				} else {
					for _, host := range hostList {
						fields := Fields{
							"sender":     e.Header.Get("From"),
							"mx":         host,
							"recipients": rcpts,
						}
						err := smtp.SendMail(host+":"+portSMTP, nil,
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
