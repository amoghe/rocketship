package main

import (
	"log"
	"os"
	"rocketship/commander"

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

	logger := log.New(os.Stderr, "", log.LstdFlags)

	die := func(err error) {
		logger.Fatalln("Exiting due to:", err.Error())
	}

	logger.Println("Connecting to", *DbType, "using DSN", *DbDSN)
	db, err := gorm.Open(*DbType, *DbDSN)
	if err != nil {
		die(err)
	}

	cmdr := commander.New(&db)

	logger.Println("Migrating database")
	cmdr.MigrateDB()

	logger.Println("Seeding database")
	cmdr.SeedDB()

	logger.Println("Preflight finished")
}
