package ssh

import (
	"strings"
	"testing"

	"rocketship/regulog"

	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"
)

type SshConfigTestSuite struct {
	db         *gorm.DB
	controller *Controller
}

// Register the test suite with gocheck.
func init() {
	Suite(&SshConfigTestSuite{})
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

func (ts *SshConfigTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)
	ts.db = &db

	ts.controller = NewController(ts.db, regulog.NewNull(""))
	ts.controller.MigrateDB()
	ts.controller.SeedDB()
}

func (ts *SshConfigTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

func (ts *SshConfigTestSuite) TestSyslogConfFile(c *C) {

	// these are lines that are populated by us in the template
	expectedLines := []string{
		"PasswordAuthentication yes",
		"PubkeyAuthentication no",
	}

	contents, err := ts.controller.sshConfigFileContents()
	c.Assert(err, IsNil)

	for _, line := range expectedLines {
		if strings.Contains(string(contents), line) != true {
			c.Error("Did not find expected line [", line, "] in generated file")
		}
	}
}
