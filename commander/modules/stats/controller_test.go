package stats

import (
	"testing"

	"github.com/amoghe/distillog"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"
	"gopkg.in/yaml.v2"
)

type StatsConfigTestSuite struct {
	db         *gorm.DB
	controller *Controller
}

// Register the test suite with gocheck.
func init() {
	Suite(&StatsConfigTestSuite{})
}

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

func (ts *StatsConfigTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)
	ts.db = &db

	ts.controller = NewController(ts.db, distillog.NewNullLogger(""))
	ts.controller.MigrateDB()
	ts.controller.SeedDB()
}

func (ts *StatsConfigTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

//
// Tests
//

func (ts *StatsConfigTestSuite) TestPrometheusConfigHasValidYAML(c *C) {
	conf, _ := ts.controller.prometheusFileContents()
	c.Assert(yaml.Unmarshal(conf, &map[interface{}]interface{}{}), IsNil)
}
