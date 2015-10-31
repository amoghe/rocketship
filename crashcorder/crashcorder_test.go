package crashcorder

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"rocketship/radio"

	"github.com/amoghe/distillog"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

func init() {
	Suite(&TestSuite{})
}

type TestSuite struct{}

func (s *TestSuite) TestHandleCoreFile(c *C) {
	var (
		reqBody []byte
		err     error
	)

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		reqBody, err = ioutil.ReadAll(r.Body)
		c.Assert(err, IsNil)
		w.WriteHeader(200)
	}

	testServer := httptest.NewServer(http.HandlerFunc(testHandler))
	defer testServer.Close()

	testAddr := testServer.Listener.Addr()
	radioAddr, err := net.ResolveTCPAddr(testAddr.Network(), testAddr.String())

	cc := New(Config{[]string{"%e", "%p", "%s", "%t"}, "/tmp", *radioAddr}, distillog.NewNullLogger(""))
	err = cc.handleCoreFile("foo_bar_baz_quz")
	c.Assert(err, IsNil)

	var rmsg radio.MessageRequest
	err = json.Unmarshal(reqBody, &rmsg)
	c.Assert(err, IsNil)

	c.Assert(rmsg.Subject, Equals, NotificationSubject)
}
