package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kingpin"
	"github.com/amoghe/distillog"

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

		logger  = distillog.NewStdoutLogger("crashcorder")
		sigChan = make(chan os.Signal)
	)

	kingpin.Version("0.0.1")
	kingpin.Parse()

	die := func(err error) {
		logger.Errorln("Exiting due to:", err.Error())
		os.Exit(2)
	}

	parseConfig := func() (cfg crashcorder.Config) {
		// read config file
		logger.Infoln("Reading config file from", *cfgFile)
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
		logger.Infoln("Starting crashcorder")
		cc = crashcorder.New(parseConfig(), logger)
		if err := cc.Start(); err != nil {
			die(err)
		}
	}

	restartCrashcorder := func() {
		if cc != nil {
			logger.Infoln("Stopping crashcorder")
			cc.Stop(true)
		}
		startCrashcorder()
	}

	mainLoop := func() {
		logger.Infoln("Initializing signal handler")
		signal.Notify(sigChan, syscall.SIGINT)

		logger.Infoln("Starting main signal handler loop")
		for sig := range sigChan {
			switch sig {
			case syscall.SIGHUP:
				logger.Infoln("Received SIGHUP - reloading")
				restartCrashcorder()
			case syscall.SIGINT, syscall.SIGTERM:
				logger.Infoln("Received", sig, "- terminating")
				cc.Stop(true)
				return
			}
		}

	}

	// start the crashcorder
	startCrashcorder()

	// run main loop that coordinates crashcorder state by handling signals
	mainLoop()

	logger.Infoln("Crashcorder process exited")
}
