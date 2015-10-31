package host

import (
	"io/ioutil"
	"log"

	"github.com/amoghe/distillog"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"
)

type ResolversTestSuite struct {
	db         gorm.DB
	controller *Controller
}

func (ts *ResolversTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)

	// Comment this to enable db logs during tests
	db.SetLogger(log.New(ioutil.Discard, "", 0))
	ts.db = db

	ts.controller = NewController(&ts.db, distillog.NewNullLogger("test"))
	ts.controller.MigrateDB()
	ts.controller.SeedDB()
}

func (ts *ResolversTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

//
// tests
//

func (ts *ResolversTestSuite) TestSeeds(c *C) {
	r := ResolversConfig{}
	c.Assert(ts.db.First(&r, 1).Error, IsNil)
	c.Assert(r.DNSServerIP1, Not(HasLen), 0)
	c.Assert(r.DNSServerIP2, Not(HasLen), 0)
}
