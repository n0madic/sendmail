package sendmail

import (
	"bytes"
	"net/mail"
	"reflect"
	"testing"
)

func TestGetDumbMessage(t *testing.T) {
	expectedHeader := mail.Header{
		"From": []string{"sender@localhost"},
		"To":   []string{"user@example.com"},
	}
	expectedBody := []byte("TEST\r\n")

	_, err := GetDumbMessage("", []string{}, []byte{})
	if err.Error() != "Empty recipients list" {
		t.Error("Expected empty recipients list")
	}

	msg, err := GetDumbMessage("sender@localhost", []string{"user@example.com"}, []byte("TEST"))
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
	slice := AddressListToSlice(list)
	if !reflect.DeepEqual(slice, expected) {
		t.Error("Expected", expected, "got", slice)
	}
}

func TestGetDomainFromAddress(t *testing.T) {
	expected := "example.com"

	domain := GetDomainFromAddress("user@example.com")
	if domain != expected {
		t.Error("Expected", expected, "got", domain)
	}

	if GetDomainFromAddress("example.com") != "" {
		t.Error("Expected empty string")
	}
}
