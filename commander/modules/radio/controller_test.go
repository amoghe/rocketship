package radio

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"testing"

	"rocketship/regulog"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"

	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"
)

//
// Test Suite
//

type RadioTestSuite struct {
	db         gorm.DB
	controller *Controller
}

// Register the test suite with gocheck.
func init() {
	Suite(&RadioTestSuite{})
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

func (ts *RadioTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)

	ts.db = db
	ts.controller = NewController(&ts.db, regulog.NewNull(""))
	ts.controller.MigrateDB()
	ts.controller.SeedDB()
}

func (ts *RadioTestSuite) TearDownTest(c *C) {
	ts.controller.db.Close()
}

//
// Vars
//

//
// Tests
//

func (ts *RadioTestSuite) TestDbSeed(c *C) {
	cfg := []RadioConfig{}
	err := ts.db.Find(&cfg).Error
	c.Assert(err, IsNil)
	c.Assert(cfg, HasLen, 1)
}

func (ts *RadioTestSuite) TestEmailRecipientAdd(c *C) {
	var (
		email = "add_asdf@gmail.com"
		rec   = httptest.NewRecorder()
	)

	// Add recipient email via handler and verify response.
	addRecipient := func(tableStruct interface{}, c *C) {
		req := newJsonPostRequest("", EmailRecipient{Email: email}, c)
		ts.controller.AddInfoRecipient(web.C{}, rec, req)

		body, err := ioutil.ReadAll(rec.Body)
		c.Assert(err, IsNil)

		ent := EmailRecipient{}
		c.Assert(json.Unmarshal(body, &ent), IsNil)
		c.Assert(ent.Email, Equals, email)
	}

	addRecipient(InfoRecipient{}, c)
	addRecipient(WarnRecipient{}, c)
	addRecipient(ErrorRecipient{}, c)
}

func (ts *RadioTestSuite) TestEmailRecipientGet(c *C) {

	for _, e := range []interface{}{
		&InfoRecipient{Email: "get_a@b.com"},
		&WarnRecipient{Email: "get_x@y.com"},
		&ErrorRecipient{Email: "get_c@d.com"},
	} {
		// Ensure it is saved to DB
		c.Assert(ts.db.Create(e).Error, IsNil)
	}

	for _, f := range []func(web.C, http.ResponseWriter, *http.Request){
		ts.controller.GetInfoRecipients,
		ts.controller.GetWarnRecipients,
		ts.controller.GetErrorRecipients,
	} {
		rec := httptest.NewRecorder()
		f(web.C{}, rec, &http.Request{})

		body, err := ioutil.ReadAll(rec.Body)
		c.Assert(err, IsNil)

		res := []EmailRecipient{}
		c.Assert(json.Unmarshal(body, &res), IsNil)
		c.Assert(res, HasLen, 1)
	}
}

func (ts *RadioTestSuite) TestEmailRecipientDel(c *C) {
	for _, e := range []interface{}{
		&InfoRecipient{Email: "del_a@b.com"},
		&WarnRecipient{Email: "del_x@y.com"},
		&ErrorRecipient{Email: "del_c@d.com"},
	} {
		// Ensure it is saved to DB
		c.Assert(ts.db.Create(e).Error, IsNil)
	}

	for _, f := range []func(web.C, http.ResponseWriter, *http.Request){
		ts.controller.DeleteInfoRecipient,
		ts.controller.DeleteWarnRecipient,
		ts.controller.DeleteErrorRecipient,
	} {
		rec := httptest.NewRecorder()
		f(web.C{URLParams: map[string]string{"id": "1"}}, rec, &http.Request{})

		body, err := ioutil.ReadAll(rec.Body)
		c.Assert(err, IsNil)
		c.Assert(rec.Code, Equals, http.StatusOK)

		res := EmailRecipient{}
		c.Assert(json.Unmarshal(body, &res), IsNil)
		c.Assert(res.ID, Equals, 1)
	}

	for _, tableStruct := range []interface{}{
		InfoRecipient{},
		WarnRecipient{},
		ErrorRecipient{},
	} {
		cnt := 0
		err := ts.db.Model(tableStruct).Count(&cnt).Error
		c.Assert(err, IsNil)
		c.Assert(cnt, Equals, 0)
	}
}

func (ts *RadioTestSuite) TestCannotOverwriteEmailRecipient(c *C) {
	for iter, e := range []EmailRecipient{
		{ID: 1, Email: "a@b.com"},
		{ID: 1, Email: "x@4.com"}, // reuse ID
		{ID: 1, Email: "w@f.com"}, // reuse ID
	} {
		rec := httptest.NewRecorder()
		req := newJsonPostRequest("", e, c)

		ts.controller.AddInfoRecipient(web.C{}, rec, req)
		c.Assert(rec.Code, Equals, http.StatusOK)

		count := 0
		c.Assert(ts.db.Model(InfoRecipient{}).Count(&count).Error, IsNil)
		c.Assert(count, Equals, iter+1)
	}
}

func (ts *RadioTestSuite) TestValidation(c *C) {
	for _, e := range []string{
		"@b.com",
		"x@.com",
		"a@()f.com",
		"()@f.com",
		".x@f.com",
	} {
		// c.Log("Testing that ", e, " fails validation")
		c.Assert(ts.db.Create(&InfoRecipient{Email: e}).Error, Not(IsNil))
	}
}

func (ts *RadioTestSuite) TestConfigFileContents(c *C) {
	for _, x := range []interface{}{
		&InfoRecipient{Email: "a@info.com"},

		&WarnRecipient{Email: "a@warn.com"},
		&WarnRecipient{Email: "b@warn.com"},

		&ErrorRecipient{Email: "a@error.com"},
		&ErrorRecipient{Email: "b@error.com"},
		&ErrorRecipient{Email: "c@error.com"},
	} {
		c.Assert(ts.db.Create(x).Error, IsNil)
	}

	rcfg := RadioConfig{}
	c.Assert(ts.db.Find(&rcfg).Error, IsNil)

	mail, _ := mail.ParseAddress("test@rocketship.com")
	rcfg.DefaultFrom = *mail
	rcfg.ServerAddress = "8.8.8.8"
	rcfg.ServerPort = 4123

	c.Assert(ts.db.Save(&rcfg).Error, IsNil)

	_, err := ts.controller.radioConfFileContents()
	c.Assert(err, IsNil)

	// TODO parse string and test for presence of known substrings?
}

//
// Helpers
//

func newJsonPostRequest(url string, s interface{}, c *C) *http.Request {
	jsonBytes, err := json.Marshal(s)
	c.Assert(err, IsNil)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	c.Assert(err, IsNil)

	return req
}
