package modules

import (
	"net/http"

	"rocketship/commander/modules/bootbanks"
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
	ServeHTTPC(web.C, http.ResponseWriter, *http.Request)
	RoutePrefix() string // RoutePrefix returns the prefix the controller is responsible for.
	RewriteFiles() error // RewriteFiles causes the controller to rewrite config files it is responsible for.
	MigrateDB()          // MigrateDB tells the controller to make its required changes to the DB.
	SeedDB()             // SeedDB tells the controller to seed the db with any state that is essential to it.
}

func LoadAll(db *gorm.DB, log regulog.Logger) []Controller {
	return []Controller{
		crashcorder.NewController(db, log),
		host.NewController(db, log),
		radio.NewController(db, log),
		ssh.NewController(db, log),
		syslog.NewController(db, log),
		bootbank.NewController(db, log),
	}
}
