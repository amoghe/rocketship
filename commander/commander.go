package commander

import (
	"net/http"

	"rocketship/commander/modules/crashcorder"
	"rocketship/commander/modules/host"
	"rocketship/commander/modules/radio"
	"rocketship/commander/modules/ssh"
	"rocketship/commander/modules/syslog"
	"rocketship/regulog"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

type Controller interface {
	RoutePrefix() string // RoutePrefix returns the prefix the controller is responsible for.
	RewriteFiles() error // RewriteFiles causes the controller to rewrite config files it is responsible for.
	MigrateDB()          // MigrateDB tells the controller to make its required changes to the DB.
	SeedDB()             // SeedDB tells the controller to seed the db with any state that is essential to it.
}

type Commander struct {
	controllers []Controller
	mux         *web.Mux
	db          *gorm.DB
	log         regulog.Logger
}

func New(db *gorm.DB, log regulog.Logger) *Commander {
	c := Commander{
		db:  db,
		mux: web.New(),
		log: log,
	}

	// crashcorder controller
	cc := crashcorder.NewController(db, log)
	c.mux.Handle(cc.RoutePrefix()+"/*", cc)
	c.controllers = append(c.controllers, cc)

	// Host controller
	hc := host.NewController(db, log)
	c.mux.Handle(hc.RoutePrefix()+"/*", hc)
	c.controllers = append(c.controllers, hc)

	// Radio controller
	rc := radio.NewController(db, log)
	c.mux.Handle(rc.RoutePrefix()+"/*", rc)
	c.controllers = append(c.controllers, rc)

	// SSH controller
	sc := ssh.NewController(db, log)
	c.mux.Handle(sc.RoutePrefix()+"/*", sc)
	c.controllers = append(c.controllers, sc)

	// Syslog controller
	ssc := syslog.NewController(db, log)
	c.mux.Handle(ssc.RoutePrefix()+"/*", ssc)
	c.controllers = append(c.controllers, ssc)

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
