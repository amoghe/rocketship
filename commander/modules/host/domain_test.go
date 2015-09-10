package host

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"rocketship/regulog"

	"github.com/jinzhu/gorm"

	. "gopkg.in/check.v1"
)

type DomainTestSuite struct {
	db         gorm.DB
	controller *Controller
}

func (ts *DomainTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)
	ts.db = db

	ts.controller = NewController(&ts.db, regulog.NewNull("test"))
	ts.controller.MigrateDB()
	ts.controller.SeedDB()
}

func (ts *DomainTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

//
// Tests
//

func (ts *DomainTestSuite) TestGetDomain(c *C) {
	req, err := http.NewRequest("GET", "/dont/care", nil)
	c.Assert(err, IsNil)

	rec := httptest.NewRecorder()

	ts.controller.GetDomain(rec, req)
	c.Assert(rec.Code, Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(rec.Body)
	c.Assert(err, IsNil)

	d := Domain{}
	err = json.Unmarshal(resbody, &d)
	c.Assert(err, IsNil)

	c.Assert(d.Domain, Equals, DefaultDomain)
}

func (ts *DomainTestSuite) TestPutDomain(c *C) {

	jsonbody := `{"Domain": "foobar"}`
	req, err := http.NewRequest("PUT", "/dont/care", bytes.NewBufferString(jsonbody))
	rec := httptest.NewRecorder()

	ts.controller.PutDomain(rec, req)
	c.Assert(rec.Code, Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(rec.Body)
	c.Assert(err, IsNil)

	d := Domain{}
	err = json.Unmarshal(resbody, &d)
	c.Assert(err, IsNil)

	// Ensure response has new domain
	c.Assert(d.Domain, Equals, "foobar")
	// Ensure it is the same as the DB
	c.Assert(ts.getDomainFromDB(c).Domain, Equals, "foobar")
}

// Helpers

func (ts *DomainTestSuite) getDomainFromDB(c *C) Domain {
	dom := Domain{}
	err := ts.db.First(&dom, 1).Error
	c.Assert(err, IsNil)
	return dom
}
