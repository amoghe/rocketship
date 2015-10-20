package commander

import (
	"net/http"

	"rocketship/commander/modules"
	"rocketship/regulog"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

type Commander struct {
	controllers []modules.Controller
	mux         *web.Mux
	db          *gorm.DB
	log         regulog.Logger
}

func New(db *gorm.DB, log regulog.Logger) *Commander {
	c := Commander{
		controllers: modules.LoadAll(db, log),
		db:          db,
		mux:         web.New(),
		log:         log,
	}

	routes := map[string]modules.Controller{}
	for _, ctrl := range c.controllers {
		if c, there := routes[ctrl.RoutePrefix()]; there {
			log.Warningf("Route %s is already serviced by %T. Skipping...", ctrl.RoutePrefix(), c)
			continue
		} else {
			routes[ctrl.RoutePrefix()] = ctrl
		}
		c.mux.Handle(ctrl.RoutePrefix()+"/*", ctrl)
	}

	return &c
}

// ServeHTTP makes Commander adhere to the http.Handler interface so it can act as a http application.
func (c *Commander) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

func (c *Commander) MigrateDB() error {
	c.log.Infoln("Migrating database")
	for _, ctrl := range c.controllers {
		ctrl.MigrateDB()
	}
	return nil
}

func (c *Commander) SeedDB() error {
	c.log.Infoln("Seeding database")
	for _, ctrl := range c.controllers {
		ctrl.SeedDB()
	}
	return nil
}

func (c *Commander) RewriteFiles() error {
	c.log.Infoln("Rewriting all configuration files")
	for _, ctrl := range c.controllers {
		if err := ctrl.RewriteFiles(); err != nil {
			c.log.Warningf("Error: %s. Continuing...", err)
		}
	}
	return nil
}
