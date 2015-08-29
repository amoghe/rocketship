package host

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/jinzhu/gorm"

	_ "fmt"

	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"
)

//
// Test Suite
//

type ControllerTestSuite struct {
	db         gorm.DB
	server     *httptest.Server
	controller *Controller
}

func (ts *ControllerTestSuite) SetUpSuite(c *C) {
	db, err := gorm.Open("sqlite3", ":memory:")
	c.Assert(err, IsNil)

	ts.db = db
	ts.controller = NewController(&ts.db)
	ts.server = httptest.NewServer(ts.controller)
}

func (ts *ControllerTestSuite) TearDownSuite(c *C) {
	ts.server.Close()
}

func (ts *ControllerTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)
	ts.db = db
	ts.controller.MigrateDB()
	ts.controller.SeedDB()
}

func (ts *ControllerTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

//
// Helpers
//

func (ts *ControllerTestSuite) requestWithJSONBody(c *C, reqtype, url string, bodystruct interface{}) *http.Request {
	bodybytes, err := json.Marshal(bodystruct)
	c.Assert(err, IsNil)

	req, err := http.NewRequest(reqtype, url, bytes.NewBuffer(bodybytes))
	c.Assert(err, IsNil)

	return req
}

func (ts *ControllerTestSuite) getHostnameFromRequest(c *C) Hostname {
	req, err := http.NewRequest("GET", ts.server.URL+EHostname, nil)
	c.Assert(err, IsNil)

	client := &http.Client{}
	resp, err := client.Do(req)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)

	h := Hostname{}
	err = json.Unmarshal(resbody, &h)
	c.Assert(err, IsNil)

	return h
}

func (ts *ControllerTestSuite) getDomainFromRequest(c *C) Domain {
	req, err := http.NewRequest("GET", ts.server.URL+EDomain, nil)
	c.Assert(err, IsNil)

	client := &http.Client{}
	resp, err := client.Do(req)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)

	d := Domain{}
	err = json.Unmarshal(resbody, &d)
	c.Assert(err, IsNil)

	return d
}

func (ts *ControllerTestSuite) getHostnameFromDB(c *C) Hostname {
	host := Hostname{}
	err := ts.db.First(&host, 1).Error
	c.Assert(err, IsNil)
	return host
}

func (ts *ControllerTestSuite) getDomainFromDB(c *C) Domain {
	dom := Domain{}
	err := ts.db.First(&dom, 1).Error
	c.Assert(err, IsNil)
	return dom
}

//
// Tests
//

func (ts *ControllerTestSuite) TestDefaultEntries(c *C) {
	ifaces := []InterfaceConfig{}
	profile := DHCPProfile{}

	err := ts.db.Find(&ifaces).Error
	c.Assert(err, IsNil)

	err = ts.db.Find(&profile, 1).Error
	c.Assert(err, IsNil)

	// ensure one interface
	c.Assert(len(ifaces), Equals, 1)

	// ensure it is in dhcp mode, and uses the default profile
	c.Assert(ifaces[0].Mode, Equals, ModeDHCP)
	c.Assert(ifaces[0].DHCPProfileID, Equals, profile.ID)
}

func (ts *ControllerTestSuite) TestGetHostname(c *C) {
	h := ts.getHostnameFromRequest(c)
	c.Assert(h.Hostname, Equals, DefaultHostname)
}

func (ts *ControllerTestSuite) TestGetDomain(c *C) {
	d := ts.getDomainFromRequest(c)
	c.Assert(d.Domain, Equals, DefaultDomain)
}

func (ts *ControllerTestSuite) TestPutHostname(c *C) {

	body := Hostname{Hostname: "foobar"}
	req := ts.requestWithJSONBody(c, "PUT", ts.server.URL+EHostname, body)

	client := &http.Client{}
	resp, err := client.Do(req)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)

	h := Hostname{}
	err = json.Unmarshal(resbody, &h)
	c.Assert(err, IsNil)

	// Ensure response has new hostname
	c.Assert(h.Hostname, Equals, "foobar")

	// Ensure it is the same as the DB
	h1 := ts.getHostnameFromDB(c)
	h2 := ts.getHostnameFromRequest(c)
	c.Assert(h1.Hostname, Equals, h2.Hostname)
}

func (ts *ControllerTestSuite) TestPutDomain(c *C) {

	body := Domain{Domain: "foobar"}
	req := ts.requestWithJSONBody(c, "PUT", ts.server.URL+EDomain, body)

	client := &http.Client{}
	resp, err := client.Do(req)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)

	d := Domain{}
	err = json.Unmarshal(resbody, &d)
	c.Assert(err, IsNil)

	// Ensure response has new domain
	c.Assert(d.Domain, Equals, "foobar")

	// Ensure it is the same as the DB
	d1 := ts.getDomainFromDB(c)
	d2 := ts.getDomainFromRequest(c)
	c.Assert(d1.Domain, Equals, d2.Domain)
}
