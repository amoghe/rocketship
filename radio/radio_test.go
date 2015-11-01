package radio

import (
	"testing"

	"bytes"
	"encoding/json"
	"net/http"
	"net/mail"
	"reflect"

	. "gopkg.in/check.v1"
)

var testRadio = Radio{
	config: Config{
		EmailConfig: EmailSettings{
			DefaultFrom: mail.Address{Address: "testmaster@example.com", Name: "A"},
			InfoRecipients: []mail.Address{
				{Address: "test1@info.com"},
				{Address: "test2@info.com"},
			},
			WarnRecipients: []mail.Address{
				{Address: "test1@warn.com"},
				{Address: "test2@warn.com"},
			},
			ErrorRecipients: []mail.Address{
				{Address: "test1@error.com"},
				{Address: "test2@error.com"},
			},
			ServerAddress: "smtp.gmail.com",
			ServerPort:    587,
			AuthHost:      "smtp.gmail.com",
			AuthUsername:  "xxx",
			AuthPassword:  "yyy",
		},
	},
}

func Test(t *testing.T) {
	Suite(&TestSuite{})
	TestingT(t)
}

type TestSuite struct{}

func (t *TestSuite) TestParseInvalidMessageFromRequest(c *C) {
	reqbody, err := json.Marshal(MessageRequest{Severity: "BAD", Subject: "sub", Body: "foobar"})
	if err != nil {
		c.Error(err)
	}

	testreq, err := http.NewRequest("POST", "/dontcare", bytes.NewBuffer(reqbody))
	if err != nil {
		c.Error(err)
	}

	_, err = testRadio.parseMessageFromRequest(testreq)
	if err == nil {
		c.Error("Expected parse error")
	}
}

func (t *TestSuite) TestParseValidMessageFromRequest(c *C) {
	reqbody, err := json.Marshal(MessageRequest{Severity: "INFO", Subject: "sub", Body: "foobar"})
	if err != nil {
		c.Error(err)
	}

	testreq, err := http.NewRequest("POST", "/dontcare", bytes.NewBuffer(reqbody))
	if err != nil {
		c.Error(err)
	}

	msg, err := testRadio.parseMessageFromRequest(testreq)
	if err != nil {
		c.Error("Unexpected parse error")
	}

	if !reflect.DeepEqual(msg.To, testRadio.config.EmailConfig.InfoRecipients) {
		c.Error("Expected 'To' to be the same as 'info recipients'")
	}
}
