package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kingpin"

	"rocketship/crashcorder"
)

const (
	DefaultCfgFile = "/etc/crashcorder/crashcorder.conf"
)

var (
	cfgFile = kingpin.Flag("conf", "Config file path.").Required().ExistingFile()
)

func main() {
	var (
		cc *crashcorder.Crashcorder

		logger  = log.New(os.Stderr, "", log.LstdFlags)
		sigChan = make(chan os.Signal)
	)

	kingpin.Version("0.0.1")
	kingpin.Parse()

	die := func(err error) {
		logger.Fatalln("Exiting due to:", err.Error())
	}

	parseConfig := func() (cfg crashcorder.Config) {
		// read config file
		logger.Println("Reading config file from", *cfgFile)
		fileBytes, err := ioutil.ReadFile(*cfgFile)
		if err != nil {
			die(err)
		}

		// parse info cfg struct
		err = json.Unmarshal(fileBytes, &cfg)
		if err != nil {
			die(err)
		}

		return
	}

	startCrashcorder := func() {
		logger.Println("Starting crashcorder")
		cc = crashcorder.New(parseConfig(), logger)
		if err := cc.Start(); err != nil {
			die(err)
		}
	}

	restartCrashcorder := func() {
		if cc != nil {
			logger.Println("Stopping crashcorder")
			cc.Stop(true)
		}
		startCrashcorder()
	}

	mainLoop := func() {
		logger.Println("Initializing signal handler")
		signal.Notify(sigChan, syscall.SIGINT)

		logger.Println("Starting main signal handler loop")
		for sig := range sigChan {
			switch sig {
			case syscall.SIGHUP:
				logger.Println("Received SIGHUP - reloading")
				restartCrashcorder()
			case syscall.SIGINT, syscall.SIGTERM:
				logger.Println("Received", sig, "- terminating")
				cc.Stop(true)
				return
			}
		}

	}

	// start the crashcorder
	startCrashcorder()

	// run main loop that coordinates crashcorder state by handling signals
	mainLoop()

	logger.Println("Crashcorder process exited")
}
