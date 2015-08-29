package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rocketship/commander"

	"github.com/alecthomas/kingpin"
	"github.com/facebookgo/httpdown"
	"github.com/jinzhu/gorm"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ListenPort = kingpin.Flag("port", "Listen port.").Default("8888").Uint64()
	DbType     = kingpin.Flag("db-type", "DB type to connect").Default("sqlite3").String()
	DbDSN      = kingpin.Flag("db-dsn", "DB DSN to connect").Default("/tmp/commander").String()
)

func main() {
	var (
		svr httpdown.Server
		db  gorm.DB
		err error

		logger   = log.New(os.Stderr, "", log.LstdFlags)
		sigChan  = make(chan os.Signal)
		dbOpened = false
	)

	kingpin.Version("0.0.1")
	kingpin.Parse()

	die := func(err error) {
		logger.Fatalln("Exiting due to:", err.Error())
	}

	reconnectDB := func() {
		if dbOpened {
			db.Close() // close the conn before connecting
		}

		logger.Println("Connecting to", *DbType, "using DSN", *DbDSN)
		db, err = gorm.Open(*DbType, *DbDSN)
		if err != nil {
			die(err)
		}
		dbOpened = true
	}

	startCommander := func() {
		logger.Println("Initializing commander server database")
		reconnectDB()

		// Start an http server with this radio app
		logger.Println("Starting commander server on port", *ListenPort)
		var err error
		svr, err = httpdown.HTTP{
			StopTimeout: 5 * time.Second,
			KillTimeout: 5 * time.Second,
		}.ListenAndServe(&http.Server{
			Addr:    fmt.Sprintf("127.0.0.1:%d", *ListenPort),
			Handler: commander.New(&db),
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
		logger.Println("Initializing signal handler")
		signal.Notify(sigChan, syscall.SIGINT)

		logger.Println("Starting main signal handler loop")
		for sig := range sigChan {
			switch sig {
			case syscall.SIGHUP:
				logger.Println("Received SIGHUP - reloading")
				restartCommander()
			case syscall.SIGINT, syscall.SIGTERM:
				logger.Println("Received sig:", sig, "- terminating")
				svr.Stop()
				return
			}
		}

	}

	startCommander()

	mainLoop()

	logger.Println("Commander server exited")

}
