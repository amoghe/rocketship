package host

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"rocketship/regulog"
	"strings"

	. "gopkg.in/check.v1"

	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
)

type HostnameTestSuite struct {
	db         gorm.DB
	controller *Controller
}

func (ts *HostnameTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)

	// Comment this to enable db logs during tests
	db.SetLogger(log.New(ioutil.Discard, "", 0))
	ts.db = db

	ts.controller = NewController(&ts.db, regulog.NewNull("test"))
	ts.controller.MigrateDB()
	ts.controller.SeedDB()
}

func (ts *HostnameTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

//
// Tests
//

func (ts *HostnameTestSuite) TestMinimumHostnameLength(c *C) {
	err := ts.db.Create(&Hostname{Hostname: ""}).Error
	c.Assert(err, Not(IsNil))
}

func (ts *HostnameTestSuite) TestInvalidCharsInHostname(c *C) {
	err := ts.db.Create(&Hostname{Hostname: "asdf.asdf"}).Error
	c.Assert(err, Not(IsNil))

	err = ts.db.Create(&Hostname{Hostname: "asdf/asdf"}).Error
	c.Assert(err, Not(IsNil))

	err = ts.db.Create(&Hostname{Hostname: "asdf asdf"}).Error
	c.Assert(err, Not(IsNil))
}

func (ts *HostnameTestSuite) TestHostnameFileContents(c *C) {
	contents, err := ts.controller.hostnameFileContents()
	c.Assert(err, IsNil)
	c.Assert(string(contents), Equals, DefaultHostname+"\n")
}

func (ts *HostnameTestSuite) TestEtcHostsFileContents(c *C) {
	contents, err := ts.controller.etcHostsFileContents()
	c.Log(string(contents))
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(contents), "127.0.0.1 localhost"), Equals, true)
}

func (ts *HostnameTestSuite) TestGetHostname(c *C) {
	req, err := http.NewRequest("PUT", "/dont/care", bytes.NewBufferString(""))
	c.Assert(err, IsNil)

	rec := httptest.NewRecorder()

	ts.controller.GetHostname(rec, req)
	c.Assert(err, IsNil)
	c.Assert(rec.Code, Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(rec.Body)
	c.Assert(err, IsNil)

	h := Hostname{}
	err = json.Unmarshal(resbody, &h)
	c.Assert(err, IsNil)

	c.Assert(ts.getHostnameFromDB(c).Hostname, Equals, h.Hostname)
}

func (ts *HostnameTestSuite) TestPutHostname(c *C) {

	jsonStr := `{"Hostname": "foobar"}`
	req, err := http.NewRequest("PUT", "/dont/care", bytes.NewBufferString(jsonStr))
	c.Assert(err, IsNil)

	rec := httptest.NewRecorder()

	ts.controller.PutHostname(rec, req)
	c.Assert(err, IsNil)
	c.Assert(rec.Code, Equals, http.StatusOK)

	resbody, err := ioutil.ReadAll(rec.Body)
	c.Assert(err, IsNil)

	h := Hostname{}
	err = json.Unmarshal(resbody, &h)
	c.Assert(err, IsNil)

	// Ensure response has new hostname
	c.Assert(h.Hostname, Equals, "foobar")

	// Ensure it is the same as the DB
	dbname := ts.getHostnameFromDB(c).Hostname
	c.Assert(dbname, Equals, "foobar")
}

//
// Helpers
//

func (ts *HostnameTestSuite) getHostnameFromDB(c *C) Hostname {
	host := Hostname{}
	err := ts.db.First(&host, 1).Error
	c.Assert(err, IsNil)
	return host
}
