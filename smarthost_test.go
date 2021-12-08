package sendmail_test

import (
	"testing"

	"github.com/n0madic/sendmail"
	"github.com/n0madic/sendmail/test"
)

func TestSendSmarthost(t *testing.T) {
	go test.StartSMTP()

	for _, config := range testConfigs {
		envelope, err := sendmail.NewEnvelope(&config.initial)
		if err != nil {
			t.Error(err)
			return
		}
		errs := envelope.SendSmarthost("localhost:"+test.PortSMTP, "", "")
		for result := range errs {
			if result.Level < 2 {
				t.Error(result.Error)
			}
		}
	}
}
