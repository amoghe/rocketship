package syslog

import (
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

type SyslogTestSuite struct{}

// Register the test suite with gocheck.
func init() {
	Suite(&SyslogTestSuite{})
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

func (ts *SyslogTestSuite) TestSyslogConfFile(c *C) {
	ctrl := Controller{}

	// these are lines that are populated by us in the template
	expectedLines := []string{
		"$FileOwner 101",
		"$FileGroup 103",
		"$PrivDropToUser 101",
		"$PrivDropToGroup 103",
	}

	contents, err := ctrl.syslogConfFileContents()
	c.Assert(err, IsNil)

	for _, line := range expectedLines {
		c.Assert(strings.Contains(string(contents), line), Equals, true)
	}
}
