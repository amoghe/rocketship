package fqdn

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jinzhu/gorm"
	"gopkg.in/check.v1"

	_ "fmt"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	check.Suite(&TestSuite{})
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type TestSuite struct {
	db              gorm.DB
	server          *httptest.Server
	controller      *Controller
	commitChan      chan func() error
	commitErrorChan chan error
}

//
// Helpers
//

func (ts *TestSuite) SetUpSuite(c *check.C) {
	db, err := gorm.Open("sqlite3", ":memory:")
	c.Assert(err, check.IsNil)

	ts.db = db
	ts.commitChan = make(chan func() error)
	ts.commitErrorChan = make(chan error)

	ts.controller = NewController(&ts.db, ts.commitChan, ts.commitErrorChan)
	ts.controller.MigrateDB()

	ts.server = httptest.NewServer(ts.controller)

	// Fake the commit processor
	go func() {
		for _ = range ts.commitChan {
			ts.commitErrorChan <- nil
		}
	}()
}

func (ts *TestSuite) SetUpTest(c *check.C) {
	db, err := gorm.Open("sqlite3", ":memory:")
	c.Assert(err, check.IsNil)
	ts.db = db
	ts.controller.MigrateDB()
}

func (ts *TestSuite) TearDownSuite(c *check.C) {
	ts.server.Close()
	close(ts.commitChan)
	close(ts.commitErrorChan)
}

func (ts *TestSuite) getHostnameFromRequest(c *check.C) Hostname {
	req, err := http.NewRequest("GET", ts.server.URL+EHostname, nil)
	c.Assert(err, check.IsNil)

	client := &http.Client{}
	resp, err := client.Do(req)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, check.IsNil)

	h := Hostname{}
	err = json.Unmarshal(resbody, &h)
	c.Assert(err, check.IsNil)

	return h
}

func (ts *TestSuite) getDomainFromRequest(c *check.C) Domain {
	req, err := http.NewRequest("GET", ts.server.URL+EDomain, nil)
	c.Assert(err, check.IsNil)

	client := &http.Client{}
	resp, err := client.Do(req)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, check.IsNil)

	d := Domain{}
	err = json.Unmarshal(resbody, &d)
	c.Assert(err, check.IsNil)

	return d
}

func (ts *TestSuite) getHostnameFromDB(c *check.C) Hostname {
	host := Hostname{}
	err := ts.db.First(&host, 1).Error
	c.Assert(err, check.IsNil)
	return host
}

func (ts *TestSuite) getDomainFromDB(c *check.C) Domain {
	dom := Domain{}
	err := ts.db.First(&dom, 1).Error
	c.Assert(err, check.IsNil)
	return dom
}

//
// Tests
//

func (ts *TestSuite) TestGetHostname(c *check.C) {
	h := ts.getHostnameFromRequest(c)
	c.Assert(h.Hostname, check.Equals, DefaultHostname)
}

func (ts *TestSuite) TestGetDomain(c *check.C) {
	d := ts.getDomainFromRequest(c)
	c.Assert(d.Domain, check.Equals, DefaultDomain)
}

func (ts *TestSuite) TestPutHostname(c *check.C) {

	body := Hostname{Hostname: "foobar"}
	bodybytes, err := json.Marshal(body)
	c.Assert(err, check.IsNil)

	req, err := http.NewRequest("PUT", ts.server.URL+EHostname, bytes.NewBuffer(bodybytes))
	c.Assert(err, check.IsNil)

	client := &http.Client{}
	resp, err := client.Do(req)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, check.IsNil)

	h := Hostname{}
	err = json.Unmarshal(resbody, &h)
	c.Assert(err, check.IsNil)

	// Ensure response has new hostname
	c.Assert(h.Hostname, check.Equals, "foobar")

	// Ensure it is the same as the DB
	h1 := ts.getHostnameFromDB(c)
	h2 := ts.getHostnameFromRequest(c)
	c.Assert(h1.Hostname, check.Equals, h2.Hostname)
}

func (ts *TestSuite) TestPutDomain(c *check.C) {

	body := Domain{Domain: "foobar"}
	bodybytes, err := json.Marshal(body)
	c.Assert(err, check.IsNil)

	req, err := http.NewRequest("PUT", ts.server.URL+EDomain, bytes.NewBuffer(bodybytes))
	c.Assert(err, check.IsNil)

	client := &http.Client{}
	resp, err := client.Do(req)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, check.IsNil)

	d := Domain{}
	err = json.Unmarshal(resbody, &d)
	c.Assert(err, check.IsNil)

	// Ensure response has new domain
	c.Assert(d.Domain, check.Equals, "foobar")

	// Ensure it is the same as the DB
	d1 := ts.getDomainFromDB(c)
	d2 := ts.getDomainFromRequest(c)
	c.Assert(d1.Domain, check.Equals, d2.Domain)
}

func (ts *TestSuite) TestHostnameFileContents(c *check.C) {
	contents, err := ts.controller.hostnameFileContents()
	c.Assert(err, check.IsNil)
	c.Assert(string(contents), check.Equals, DefaultHostname+"\n")
}
