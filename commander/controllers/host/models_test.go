package host

import (
	"github.com/jinzhu/gorm"

	. "gopkg.in/check.v1"
)

//
// Test Suite
//

type ModelsTestSuite struct {
	db         gorm.DB
	controller *Controller
}

func (ts *ModelsTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)
	ts.db = db

	ts.controller = NewController(&ts.db)
	ts.controller.MigrateDB()
}

func (ts *ModelsTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

//
// Tests
//

func (ts *ModelsTestSuite) TestSaveInterfaceConfigShouldFailWithInvalidIPs(c *C) {
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

func (ts *ModelsTestSuite) TestMinimumHostnameLength(c *C) {
	err := ts.db.Create(&Hostname{Hostname: ""}).Error
	c.Assert(err, Not(IsNil))
}

func (ts *ModelsTestSuite) TestInvalidCharsInHostname(c *C) {
	err := ts.db.Create(&Hostname{Hostname: "asdf.asdf"}).Error
	c.Assert(err, Not(IsNil))

	err = ts.db.Create(&Hostname{Hostname: "asdf/asdf"}).Error
	c.Assert(err, Not(IsNil))

	err = ts.db.Create(&Hostname{Hostname: "asdf asdf"}).Error
	c.Assert(err, Not(IsNil))
}

func (ts *ModelsTestSuite) TestFailDHCPProfileDeleteIfInterfaceUsesIt(c *C) {
	err := ts.db.Delete(&DHCPProfile{ID: 1}).Error
	c.Assert(err, Not(IsNil))
}
