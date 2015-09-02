package commander

import (
	"net/http"

	"rocketship/commander/modules/crashcorder"
	"rocketship/commander/modules/host"
	"rocketship/commander/modules/radio"
	"rocketship/commander/modules/ssh"
	"rocketship/commander/modules/syslog"

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
}

func New(db *gorm.DB) *Commander {
	c := Commander{
		db:  db,
		mux: web.New(),
	}

	// crashcorder controller
	cc := crashcorder.NewController(db)
	c.mux.Handle(cc.RoutePrefix()+"/*", cc)
	c.controllers = append(c.controllers, cc)

	// Host controller
	hc := host.NewController(db)
	c.mux.Handle(hc.RoutePrefix()+"/*", hc)
	c.controllers = append(c.controllers, hc)

	// Radio controller
	rc := radio.NewController(db)
	c.mux.Handle(rc.RoutePrefix()+"/*", rc)
	c.controllers = append(c.controllers, rc)

	// SSH controller
	sc := ssh.NewController(db)
	c.mux.Handle(sc.RoutePrefix()+"/*", sc)
	c.controllers = append(c.controllers, sc)

	// Syslog controller
	ssc := syslog.NewController(db)
	c.mux.Handle(ssc.RoutePrefix()+"/*", ssc)
	c.controllers = append(c.controllers, ssc)

	return &c
}

// ServeHTTP makes Commander adhere to the http.Handler interface so it can act as a http application.
func (c *Commander) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

func (c *Commander) MigrateDB() error {
	for _, c := range c.controllers {
		c.MigrateDB()
	}
	return nil
}

func (c *Commander) SeedDB() error {
	for _, c := range c.controllers {
		c.SeedDB()
	}
	return nil
}

func (c *Commander) RewriteFiles() error {
	for _, c := range c.controllers {
		c.RewriteFiles()
	}
	return nil
}
