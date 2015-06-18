package dhcp

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jinzhu/gorm"

	_ "fmt"
	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"
)

func init() {
	Suite(&TestSuite{})
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

//
// Test Suite
//

type TestSuite struct {
	db              gorm.DB
	server          *httptest.Server
	controller      *Controller
	commitChan      chan func() error
	commitErrorChan chan error
}

func (ts *TestSuite) SetUpSuite(c *C) {
	db, err := gorm.Open("sqlite3", ":memory:")
	c.Assert(err, IsNil)

	ts.db = db
	ts.commitChan = make(chan func() error)
	ts.commitErrorChan = make(chan error)

	ts.controller = NewController(&ts.db, ts.commitChan, ts.commitErrorChan)
	ts.server = httptest.NewServer(ts.controller)

	// Fake the commit processor
	go func() {
		for _ = range ts.commitChan {
			ts.commitErrorChan <- nil
		}
	}()
}

func (ts *TestSuite) TearDownSuite(c *C) {
	ts.server.Close()
	close(ts.commitChan)
	close(ts.commitErrorChan)
}

func (ts *TestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", ":memory:")
	c.Assert(err, IsNil)
	ts.db = db
	ts.controller.MigrateDB()
}

//
// Helpers
//

func (ts *TestSuite) requestWithJSONBody(c *C, reqtype string, bodystruct interface{}) *http.Request {
	bodybytes, err := json.Marshal(bodystruct)
	c.Assert(err, IsNil)

	req, err := http.NewRequest(reqtype, ts.server.URL+EHostname, bytes.NewBuffer(bodybytes))
	c.Assert(err, IsNil)

	return req
}

func (ts *TestSuite) getHostnameFromRequest(c *C) Hostname {
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

func (ts *TestSuite) getDomainFromRequest(c *C) Domain {
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

func (ts *TestSuite) getHostnameFromDB(c *C) Hostname {
	host := Hostname{}
	err := ts.db.First(&host, 1).Error
	c.Assert(err, IsNil)
	return host
}

func (ts *TestSuite) getDomainFromDB(c *C) Domain {
	dom := Domain{}
	err := ts.db.First(&dom, 1).Error
	c.Assert(err, IsNil)
	return dom
}

//
// Tests
//

func (ts *TestSuite) TestDefaultEntries(c *C) {
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

func (ts *TestSuite) TestGetHostname(c *C) {
	h := ts.getHostnameFromRequest(c)
	c.Assert(h.Hostname, Equals, DefaultHostname)
}

func (ts *TestSuite) TestGetDomain(c *C) {
	d := ts.getDomainFromRequest(c)
	c.Assert(d.Domain, Equals, DefaultDomain)
}

func (ts *TestSuite) TestPutHostname(c *C) {

	body := Hostname{Hostname: "foobar"}
	req := ts.requestWithJSONBody(c, "PUT", body)

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

func (ts *TestSuite) TestPutDomain(c *C) {

	body := Domain{Domain: "foobar"}
	req := ts.requestWithJSONBody(c, "PUT", body)

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

func (ts *TestSuite) TestHostnameFileContents(c *C) {
	contents, err := ts.controller.hostnameFileContents()
	c.Assert(err, IsNil)
	c.Assert(string(contents), Equals, DefaultHostname+"\n")
}

func (ts *TestSuite) TestSaveInterfaceConfigShouldFailWithInvalidIPs(c *C) {
	err := ts.db.Create(&InterfaceConfig{Name: "test", Mode: ModeStatic}).Error
	c.Assert(err, Not(IsNil))

	// missing addr
	err = ts.db.Create(&InterfaceConfig{
		Name:    "test",
		Mode:    ModeStatic,
		Gateway: "1.2.3.4",
		Netmask: "255.255.255.0"}).Error
	c.Assert(err, Not(IsNil))

	// missing gateway
	err = ts.db.Create(&InterfaceConfig{
		Name:    "test",
		Mode:    ModeStatic,
		Address: "1.2.3.4",
		Netmask: "255.255.255.0"}).Error
	c.Assert(err, Not(IsNil))

	// missing netmask
	err = ts.db.Create(&InterfaceConfig{
		Name:    "test",
		Mode:    ModeStatic,
		Address: "5.6.7.8",
		Gateway: "1.2.3.4"}).Error
	c.Assert(err, Not(IsNil))

	// invalid netmask
	err = ts.db.Create(&InterfaceConfig{
		Name:    "test",
		Mode:    ModeStatic,
		Address: "5.6.7.8",
		Netmask: "255.255.100.0",
		Gateway: "1.2.3.4"}).Error
	c.Assert(err, Not(IsNil))

	// gateway not within mask
	err = ts.db.Create(&InterfaceConfig{
		Name:    "test",
		Mode:    ModeStatic,
		Address: "192.168.168.8",
		Netmask: "255.255.255.0",
		Gateway: "1.2.3.4"}).Error
	c.Assert(err, Not(IsNil))
}

func (ts *TestSuite) TestInterfaceFileGeneration(c *C) {

	err := ts.db.Create(&InterfaceConfig{
		Name:    "test1",
		Mode:    ModeStatic,
		Address: "192.168.168.8",
		Netmask: "255.255.255.0",
		Gateway: "192.168.168.1"}).Error
	c.Assert(err, IsNil)

	err = ts.db.Create(&InterfaceConfig{
		Name:          "test2",
		Mode:          ModeDHCP,
		DHCPProfileID: 1,
	}).Error
	c.Assert(err, IsNil)

	filestr, err := ts.controller.interfacesConfigFileContents()
	c.Assert(err, IsNil)

	expectedLines := []string{
		"# This file is AUTOGENERATED.",
		"#",
		"",
		"auto lo",
		"iface lo inet loopback",
		"",
		"auto eth0",
		"iface eth0inetdhcp",
		"",
		"auto test1",
		"iface test1inetstatic",
		"address 192.168.168.8",
		"netmask 255.255.255.0",
		"gateway 192.168.168.1",
		"",
		"auto test2",
		"iface test2inetdhcp",
	}

	filebytes := bytes.NewBufferString(filestr)
	for _, expstr := range expectedLines {
		b := filebytes.Next(len(expstr) + 1)
		c.Assert(string(b), Equals, expstr+"\n")
	}
}
