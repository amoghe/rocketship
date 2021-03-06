package host

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Register the test suites we wish to run
func init() {
	Suite(&HostnameTestSuite{})
	Suite(&DomainTestSuite{})
	Suite(&InterfacesTestSuite{})
	Suite(&UsersTestSuite{})
	Suite(&SudoersTestSuite{})
	Suite(&ResolversTestSuite{})
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}
