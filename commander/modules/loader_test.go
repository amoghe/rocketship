package modules

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/amoghe/distillog"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"
)

func init() {
	Suite(&ModulesTestSuite{})
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

type ModulesTestSuite struct {
	db  gorm.DB
	log distillog.Logger
}

func (ts *ModulesTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)

	db.SetLogger(log.New(ioutil.Discard, "", 0))
	ts.db = db

	ts.log = distillog.NewNullLogger("test")
}

func (ts *ModulesTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

func (ts *ModulesTestSuite) TestLoadAll(c *C) {
	c.Assert(LoadAll(&ts.db, ts.log), Not(HasLen), 0)
}
