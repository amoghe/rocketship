package main

import (
	"fmt"
	"os"
	"rocketship/commander"

	"github.com/alecthomas/kingpin"
	"github.com/amoghe/distillog"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DbType   = kingpin.Flag("db-type", "DB type to connect").Default("sqlite3").String()
	DbDSN    = kingpin.Flag("db-dsn", "DB DSN to connect").Default("/tmp/commander").String()
	SeedOnly = kingpin.Flag("seed-only", "Only migrate+seed the database, do not rewrite files").Default("false").Bool()
	LogTo    = kingpin.Flag("log-to", "Log output").Default("stdout").Enum("syslog", "stdout", "stderr")
)

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()

	logger := distillog.NewStderrLogger("preflight")

	die := func(err error) {
		logger.Errorln("Exiting due to:", err.Error())
		os.Exit(1)
	}

	switch *LogTo {
	case "syslog":
		logger = distillog.NewSyslogLogger("commander")
	case "stdout":
		logger = distillog.NewStdoutLogger("commander")
	case "stderr":
		logger = distillog.NewStderrLogger("commander")
	default:
		die(fmt.Errorf("Unknown log output specified type"))
	}

	logger.Infoln("Connecting to", *DbType, "using DSN", *DbDSN)
	db, err := gorm.Open(*DbType, *DbDSN)
	if err != nil {
		die(err)
	}

	cmdr := commander.New(&db, logger)

	logger.Infoln("<1> Migrating database")
	cmdr.MigrateDB()

	logger.Infoln("<2> Seeding database")
	cmdr.SeedDB()

	if *SeedOnly == true {
		logger.Infoln("Exiting early due to seed-only")
		return
	}

	logger.Infoln("<3> Regenerating config files")
	cmdr.RewriteFiles()

	logger.Infoln("Preflight finished")
}
