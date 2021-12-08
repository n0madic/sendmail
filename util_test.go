package sendmail_test

import (
	"bytes"
	"net/mail"
	"reflect"
	"testing"

	"github.com/n0madic/sendmail"
)

func TestGetDumbMessage(t *testing.T) {
	expectedHeader := mail.Header{
		"From": []string{"sender@localhost"},
		"To":   []string{"user@example.com"},
	}
	expectedBody := []byte("TEST\r\n")

	_, err := sendmail.GetDumbMessage("", []string{}, []byte{})
	if err.Error() != "empty recipients list" {
		t.Error("Expected empty recipients list")
	}

	msg, err := sendmail.GetDumbMessage("sender@localhost", []string{"user@example.com"}, []byte("TEST"))
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(msg.Header, expectedHeader) {
		t.Error("Expected", expectedHeader, "got", msg.Header)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(msg.Body)
	if !reflect.DeepEqual(buf.Bytes(), expectedBody) {
		t.Error("Expected", expectedBody, "got", buf.Bytes())
	}
}

func TestAddressListToSlice(t *testing.T) {
	expected := []string{"user@example.com"}

	list := []*mail.Address{{Address: "user@example.com"}}
	slice := sendmail.AddressListToSlice(list)
	if !reflect.DeepEqual(slice, expected) {
		t.Error("Expected", expected, "got", slice)
	}
}

func TestGetDomainFromAddress(t *testing.T) {
	expected := "example.com"

	domain := sendmail.GetDomainFromAddress("user@example.com")
	if domain != expected {
		t.Error("Expected", expected, "got", domain)
	}

	if sendmail.GetDomainFromAddress("example.com") != "" {
		t.Error("Expected empty string")
	}
}
