package host

import (
	"fmt"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

const (
	// Prefix under which all the endpoints reside
	URLPrefix = "/host"
	// Endpoint at which the hostname can be configured
	EHostname = URLPrefix + "/hostname"
	// Endpoint at which domain can be configured
	EDomain = URLPrefix + "/domain"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux

	commitChan      chan func() error
	commitErrorChan chan error
}

func NewController(db *gorm.DB) *Controller {

	c := Controller{
		db:  db,
		mux: web.New(),
	}

	c.mux.Get(EHostname, c.GetHostname)
	c.mux.Put(EHostname, c.PutHostname)

	c.mux.Get(EDomain, c.GetDomain)
	c.mux.Put(EDomain, c.PutDomain)

	return &c
}

// ServeHTTP satisfies the http.Handler interface (net/http as well as goji)
func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

// RoutePrefix returns the prefix under which this router handles endpoints
func (c *Controller) RoutePrefix() string {
	return URLPrefix
}

func (c *Controller) MigrateDB() {
	c.db.AutoMigrate(&Hostname{})
	c.db.AutoMigrate(&Domain{})

	c.db.AutoMigrate(&DHCPProfile{})
	c.db.AutoMigrate(&InterfaceConfig{})

	c.db.AutoMigrate(&User{})
}

func (c *Controller) SeedDB() {
	c.seedHostname()
	c.seedDomain()
	c.seedInterface()
	c.seedUsers()
}

func (c *Controller) RewriteFiles() error {
	for _, f := range []func() error{
		c.RewriteHostnameFile,
		c.RewritePasswdFile,
		c.RewriteShadowFile,
		c.RewriteGroupsFile,
		c.RewriteInterfacesFile,
		c.RewriteDhclientConfFile,
	} {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

//
// AfterCommit
//

func (c *Controller) AfterCommit() error {
	return nil
}

//
// Helpers
//

func (c *Controller) jsonError(err error, w http.ResponseWriter) {
	// TODO: switch on err type
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
}
