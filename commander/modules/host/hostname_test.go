package host

import (
	"rocketship/regulog"
	"strings"

	. "gopkg.in/check.v1"

	"github.com/jinzhu/gorm"
)

type HostnameTestSuite struct {
	db         gorm.DB
	controller *Controller
}

func (ts *HostnameTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)
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
