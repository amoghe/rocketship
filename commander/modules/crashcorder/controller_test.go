package crashcorder

import (
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

type CrashcorderTestSuite struct{}

// Register the test suite with gocheck.
func init() {
	Suite(&CrashcorderTestSuite{})
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

func (ts *CrashcorderTestSuite) TestSyslogConfFile(c *C) {
	ctrl := Controller{}

	contents, err := ctrl.crashcorderConfigFileContents()
	c.Assert(err, IsNil)

	for _, line := range []string{
		"\"CorePatternTokens\": [",
		"\"CoresDirectory\": \"/cores\"",
	} {
		c.Assert(strings.Contains(string(contents), line), Equals, true)
	}
}
