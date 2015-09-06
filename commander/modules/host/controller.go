package host

import (
	"fmt"
	"net/http"

	"rocketship/regulog"

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
	// Endpoint at which users can be configured
	EUsers   = URLPrefix + "/users"
	EUsersID = EUsers + "/:id"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux
	log regulog.Logger
}

func NewController(db *gorm.DB, logger regulog.Logger) *Controller {

	c := Controller{
		db:  db,
		mux: web.New(),
		log: logger,
	}

	// Hostname endpoints
	c.mux.Get(EHostname, c.GetHostname)
	c.mux.Put(EHostname, c.PutHostname)
	// Domain endpoints
	c.mux.Get(EDomain, c.GetDomain)
	c.mux.Put(EDomain, c.PutDomain)
	// User endpoints
	c.mux.Get(EUsers, c.GetUsers)
	c.mux.Post(EUsers, c.CreateUser)
	c.mux.Delete(EUsersID, c.DeleteUser)
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
	c.log.Infoln("Migrating hostname table")
	c.db.AutoMigrate(&Hostname{})
	c.log.Infoln("Migrating domain table")
	c.db.AutoMigrate(&Domain{})

	c.log.Infoln("Migrating DHCP profiles table")
	c.db.AutoMigrate(&DHCPProfile{})
	c.log.Infoln("Migrating interfaces table")
	c.db.AutoMigrate(&InterfaceConfig{})

	c.log.Infoln("Migrating users table")
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
		c.EnsureHomedirs,
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
