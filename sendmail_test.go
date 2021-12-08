package sendmail_test

import (
	"bytes"
	"encoding/base64"
	"reflect"
	"strings"
	"testing"

	"github.com/n0madic/sendmail"
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
	expectedMessage := []byte{70, 114, 111, 109, 58, 32, 115, 101, 110, 100, 101, 114, 64, 108, 111, 99,
		97, 108, 104, 111, 115, 116, 13, 10, 83, 117, 98, 106, 101, 99, 116, 58, 32, 61, 63, 85, 84, 70,
		45, 56, 63, 66, 63, 99, 51, 86, 105, 97, 109, 86, 106, 100, 65, 61, 61, 13, 10, 84, 111, 58, 32,
		114, 101, 99, 105, 112, 105, 101, 110, 116, 64, 108, 111, 99, 97, 108, 104, 111, 115, 116, 13,
		10, 13, 10, 84, 69, 83, 84, 13, 10}

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
	if !reflect.DeepEqual(message, expectedMessage) {
		t.Error("Expected", expectedMessage, "got", message)
	}
}
