package sendmail_test

import (
	"bytes"
	"encoding/base64"
	"reflect"
	"strings"
	"testing"

	"github.com/n0madic/sendmail"
	"github.com/n0madic/sendmail/test"
)

type testData struct {
	initial  sendmail.Config
	expected sendmail.Config
}

var testConfigs = []*testData{
	{
		initial: sendmail.Config{
			Sender:     "sender@localhost",
			Recipients: []string{"recipient@localhost"},
			Subject:    "subject",
			Body:       []byte("TEST"),
			PortSMTP:   test.PortSMTP,
		},
		expected: sendmail.Config{
			Sender:     "sender@localhost",
			Recipients: []string{"recipient@localhost"},
			Subject:    "subject",
			Body:       []byte("TEST"),
		},
	},
	{
		initial: sendmail.Config{
			Sender:     "",
			Recipients: []string{},
			Subject:    "",
			Body: []byte(`From: sender@localhost
To: recipient@localhost
Subject: subject

TEST`,
			),
			PortSMTP: test.PortSMTP,
		},
		expected: sendmail.Config{
			Sender:     "sender@localhost",
			Recipients: []string{"recipient@localhost"},
			Subject:    "subject",
			Body:       []byte("TEST"),
		},
	},
}

func TestNewEnvelope(t *testing.T) {
	for _, config := range testConfigs {
		envelope, err := sendmail.NewEnvelope(&config.initial)
		if err != nil {
			t.Error(err)
			return
		}

		if envelope.Header["From"][0] != config.expected.Sender {
			t.Error("Expected", config.expected.Sender, "got", envelope.Header["From"][0])
		}

		if !reflect.DeepEqual(envelope.Header["To"], config.expected.Recipients) {
			t.Error("Expected", config.expected.Recipients, "got", envelope.Header["To"])
		}

		subject := []byte(envelope.Header["Subject"][0])
		if bytes.Contains(subject, []byte("=?UTF-8?B?")) {
			subject, err = base64.StdEncoding.DecodeString(
				strings.Replace(envelope.Header["Subject"][0], "=?UTF-8?B?", "", 1),
			)
			if err != nil {
				t.Error(err)
				return
			}
		}
		if string(subject) != config.expected.Subject {
			t.Error("Expected", config.expected.Subject, "got", subject)
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(envelope.Body)
		if !reflect.DeepEqual(bytes.TrimSuffix(buf.Bytes(), []byte("\r\n")), config.expected.Body) {
			t.Error("Expected", config.expected.Body, "got", bytes.TrimSpace(buf.Bytes()))
		}
	}
}

func TestGenerateMessage(t *testing.T) {
	expectedMessage := "From: sender@localhost\r\nSubject: =?UTF-8?B?c3ViamVjdA==\r\nTo: recipient@localhost\r\n\r\nTEST\r\n"

	envelope, err := sendmail.NewEnvelope(&testConfigs[0].initial)
	if err != nil {
		t.Error(err)
		return
	}
	message, err := envelope.GenerateMessage()
	if err != nil {
		t.Error(err)
		return
	}
	if !bytes.Equal(message, []byte(expectedMessage)) {
		t.Errorf("EXPECTED:\n%s\nGOT:\n%s", expectedMessage, message)
	}
}
