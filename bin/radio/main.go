package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rocketship/radio"

	"github.com/alecthomas/kingpin"
	"github.com/facebookgo/httpdown"
)

const (
	DefaultCfgFile = "/etc/radio/radio.conf"
)

var (
	cfgFile = kingpin.Flag("conf", "Config file path.").Required().ExistingFile()
)

func main() {
	var (
		radioserver httpdown.Server

		logger  = log.New(os.Stderr, "", log.LstdFlags)
		sigChan = make(chan os.Signal)
	)

	kingpin.Version("0.0.1")
	kingpin.Parse()

	die := func(err error) {
		logger.Fatalln("Exiting due to:", err.Error())
	}

	parseConfig := func() (cfg radio.Config) {
		// read config file
		logger.Println("Reading config file from", *cfgFile)
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
		logger.Println("Starting radio on ", addr)
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
			logger.Println("Stopping radio server")
			radioserver.Stop()
		}
		startRadio()
	}

	mainLoop := func() {
		logger.Println("Initializing signal handler")
		signal.Notify(sigChan, syscall.SIGINT)

		logger.Println("Starting main signal handler loop")
		for sig := range sigChan {
			switch sig {
			case syscall.SIGHUP:
				logger.Println("Received SIGHUP - reloading")
				restartRadio()
			case syscall.SIGINT, syscall.SIGTERM:
				logger.Println("Received", sig, "- terminating")
				radioserver.Stop()
				return
			}
		}

	}

	startRadio()

	mainLoop()

	logger.Println("Radio server exited")
}
