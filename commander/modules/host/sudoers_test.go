package host

import (
	"strings"

	"github.com/amoghe/distillog"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"
)

type SudoersTestSuite struct {
	controller *Controller
}

func (ts *SudoersTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)

	ts.controller = NewController(&db, distillog.NewNullLogger(""))
}

func (ts *SudoersTestSuite) TestSudoersFileContents(c *C) {
	contents, err := ts.controller.sudoersFileContents()
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(contents), "admin ALL=(ALL:ALL) ALL"), Equals, true)
}
