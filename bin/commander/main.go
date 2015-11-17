package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rocketship/commander"

	"github.com/alecthomas/kingpin"
	"github.com/amoghe/distillog"
	"github.com/facebookgo/httpdown"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ListenPort = kingpin.Flag("port", "Listen port.").Default("8888").Uint64()
	ListenAddr = kingpin.Flag("interface", "Listen interface").Default("127.0.0.1").String()
	DbType     = kingpin.Flag("db-type", "DB type to connect").Default("sqlite3").String()
	DbDSN      = kingpin.Flag("db-dsn", "DB DSN to connect").Default("/tmp/commander").String()
	LogType    = kingpin.Flag("log-to", "Log output").Default("stdout").Enum("syslog", "stdout", "stderr")
)

func main() {
	var (
		svr    httpdown.Server
		db     gorm.DB
		err    error
		logger distillog.Logger

		sigChan  = make(chan os.Signal)
		dbOpened = false
	)

	kingpin.Version("0.0.1")
	kingpin.Parse()

	die := func(err error) {
		logger.Errorln("Exiting due to:", err.Error())
		os.Exit(2)
	}

	setupLogger := func() {
		switch *LogType {
		case "syslog":
			logger = distillog.NewSyslogLogger("commander")
		case "stdout":
			logger = distillog.NewStdoutLogger("commander")
		case "stderr":
			logger = distillog.NewStderrLogger("commander")
		default:
			die(fmt.Errorf("Unknown log output specified type"))
		}
	}

	reconnectDB := func() {
		if dbOpened {
			db.Close() // close the conn before connecting
		}

		logger.Infoln("Connecting to", *DbType, "using DSN", *DbDSN)
		db, err = gorm.Open(*DbType, *DbDSN)
		if err != nil {
			die(err)
		}
		dbOpened = true
	}

	startCommander := func() {
		logger.Infoln("Initializing commander server database")
		reconnectDB()

		// Start an http server with this radio app
		logger.Infoln("Starting commander server on port", *ListenPort)
		var err error
		svr, err = httpdown.HTTP{
			StopTimeout: 5 * time.Second,
			KillTimeout: 5 * time.Second,
		}.ListenAndServe(&http.Server{
			Addr:    fmt.Sprintf("%s:%d", *ListenAddr, *ListenPort),
			Handler: commander.New(&db, logger),
		})
		if err != nil {
			die(fmt.Errorf("Failed to start http server: %s", err))
		}
	}

	restartCommander := func() {
		if svr != nil {
			svr.Stop()
		}
		startCommander()
	}

	mainLoop := func() {
		logger.Infoln("Initializing signal handler")
		signal.Notify(sigChan, syscall.SIGINT)

		logger.Infoln("Starting main signal handler loop")
		for sig := range sigChan {
			switch sig {
			case syscall.SIGHUP:
				logger.Infoln("Received SIGHUP - reloading")
				restartCommander()
			case syscall.SIGINT, syscall.SIGTERM:
				logger.Infoln("Received sig:", sig, "- terminating")
				svr.Stop()
				return
			}
		}

	}

	setupLogger()

	startCommander()

	mainLoop()

	logger.Infoln("Commander server exited")

}
