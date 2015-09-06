package main

import (
	"os"
	"rocketship/commander"
	"rocketship/regulog"

	"github.com/alecthomas/kingpin"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DbType = kingpin.Flag("db-type", "DB type to connect").Default("sqlite3").String()
	DbDSN  = kingpin.Flag("db-dsn", "DB DSN to connect").Default("/tmp/commander").String()
)

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()

	logger := regulog.New("preflight")

	die := func(err error) {
		logger.Errorln("Exiting due to:", err.Error())
		os.Exit(1)
	}

	logger.Infoln("Connecting to", *DbType, "using DSN", *DbDSN)
	db, err := gorm.Open(*DbType, *DbDSN)
	if err != nil {
		die(err)
	}

	cmdr := commander.New(&db, logger)

	logger.Infoln("Migrating database")
	cmdr.MigrateDB()

	logger.Infoln("Seeding database")
	cmdr.SeedDB()

	logger.Infoln("Regenerating config files")
	cmdr.RewriteFiles()

	logger.Infoln("Preflight finished")
}
