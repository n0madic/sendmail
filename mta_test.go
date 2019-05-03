package sendmail

import (
	"testing"

	"github.com/n0madic/sendmail/test"
)

func init() {
	portSMTP = test.PortSMTP
}
func TestSendLikeMTA(t *testing.T) {
	go test.StartSMTP()

	for _, config := range testConfigs {
		envelope, err := NewEnvelope(&config.initial)
		if err != nil {
			t.Error(err)
			return
		}
		errs := envelope.SendLikeMTA()
		for result := range errs {
			if result.Level < 2 {
				t.Error(result.Error)
			}
		}
	}
}
