package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rocketship/radio"

	"github.com/alecthomas/kingpin"
	"github.com/amoghe/distillog"
	"github.com/facebookgo/httpdown"
)

const (
	DefaultCfgFile = "/etc/radio/radio.conf"
)

var (
	cfgFile = kingpin.Flag("conf", "Config file path.").Required().ExistingFile()
	logTo   = kingpin.Flag("log-to", "Log output").Default("stdout").Enum("syslog", "stdout", "stderr")
)

func main() {
	var (
		radioserver httpdown.Server
		logger      distillog.Logger

		sigChan = make(chan os.Signal)
	)

	kingpin.Version("0.0.1")
	kingpin.Parse()

	die := func(err error) {
		logger.Errorln("Exiting due to:", err.Error())
		os.Exit(2)
	}

	setupLogger := func() {
		switch *logTo {
		case "syslog":
			logger = distillog.NewSyslogLogger("radio")
		case "stdout":
			logger = distillog.NewStdoutLogger("radio")
		case "stderr":
			logger = distillog.NewStderrLogger("radio")
		default:
			die(fmt.Errorf("Unknown log output specified type"))
		}
	}

	parseConfig := func() (cfg radio.Config) {
		// read config file
		logger.Infoln("Reading config file from", *cfgFile)
		fileBytes, err := ioutil.ReadFile(*cfgFile)
		if err != nil {
			die(fmt.Errorf("Failed to read file: %s", err))
		}

		// parse info cfg struct
		err = json.Unmarshal(fileBytes, &cfg)
		if err != nil {
			die(fmt.Errorf("Failed to parse config: %s", err))
		}

		return
	}

	startRadio := func() {
		conf := parseConfig()
		addr := fmt.Sprintf("127.0.0.1:%d", conf.ProcessConfig.ListenPort)

		// Start an http server with this radio app
		logger.Infoln("Starting radio on ", addr)
		var err error
		radioserver, err = httpdown.HTTP{
			StopTimeout: 5 * time.Second,
			KillTimeout: 5 * time.Second,
		}.ListenAndServe(&http.Server{Addr: addr, Handler: radio.New(conf)})
		if err != nil {
			die(fmt.Errorf("Failed to start http server: %s", err))
		}
	}

	restartRadio := func() {
		if radioserver != nil {
			logger.Infoln("Stopping radio server")
			radioserver.Stop()
		}
		startRadio()
	}

	mainLoop := func() {
		logger.Infoln("Initializing signal handler")
		signal.Notify(sigChan, syscall.SIGINT)

		logger.Infoln("Starting main signal handler loop")
		for sig := range sigChan {
			switch sig {
			case syscall.SIGHUP:
				logger.Infoln("Received SIGHUP - reloading")
				restartRadio()
			case syscall.SIGINT, syscall.SIGTERM:
				logger.Infoln("Received", sig, "- terminating")
				radioserver.Stop()
				return
			}
		}

	}

	setupLogger()

	startRadio()

	mainLoop()

	logger.Infoln("Radio server exited")
}
