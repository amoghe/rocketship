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

func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

func (c *Controller) MigrateDB() {

	SeedHostname(c.db)
	SeedDomain(c.db)
	SeedInterface(c.db)
	SeedUsers(c.db)
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
