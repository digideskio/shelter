package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"shelter/config"
	"shelter/log"
	"shelter/net/http/rest"
	"shelter/net/scan"
	"shelter/scheduler"
	"syscall"
	"time"
)

// We store all listeners to make it easier later to stop all in a system SIGTERM event
var (
	restListeners []net.Listener
)

// List of possible return codes of the program. This will be useful later to build a
// command line documentation
const (
	NoError = iota
	ErrInputParameters
	ErrLoadingConfig
	ErrListeningRESTInterfaces
	ErrStartingRESTServer
)

func init() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Printf("Usage: %s <configuration file>\n", os.Args[0])
		os.Exit(ErrInputParameters)
	}

	if err := loadSettings(); err != nil {
		log.Println(err)
		os.Exit(ErrLoadingConfig)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	logPath := fmt.Sprintf("%s/%s",
		config.ShelterConfig.BasePath,
		config.ShelterConfig.LogFilename,
	)

	if err := log.SetOutput(logPath); err != nil {
		log.Println(err)
		return
	}
	defer log.Close()

	if config.ShelterConfig.RESTServer.Enabled {
		var err error
		restListeners, err = rest.Listen()
		if err != nil {
			log.Println("Error while aquiring interfaces for REST server. Details:", err)
			os.Exit(ErrListeningRESTInterfaces)
		}
	}

	manageSystemSignals()

	if config.ShelterConfig.RESTServer.Enabled {
		if err := rest.Start(restListeners); err != nil {
			log.Println("Error starting the REST server. Details:", err)
			os.Exit(ErrStartingRESTServer)
		}
	}

	if config.ShelterConfig.Scan.Enabled {
		// TODO: Scan time must be configurable
		scheduler.Register(scheduler.Job{
			Interval: 24 * time.Hour,
			Task:     scan.ScanDomains,
		})
	}

	scheduler.Start()

	select {}
}

func manageSystemSignals() {
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGHUP)

		for {
			sig := <-sigs

			if sig == syscall.SIGHUP {
				if err := loadSettings(); err != nil {
					log.Println("Error reloading confirguration file. Details:", err)
				}

			} else if sig == syscall.SIGTERM {
				for _, listener := range restListeners {
					if err := listener.Close(); err != nil {
						log.Println("Error closing listener. Details:", err)
					}
				}

				// TODO: Wait the last requests to be processed?

				os.Exit(NoError)
			}
		}
	}()
}

func loadSettings() error {
	// TODO: Possible concurrent access problem while reloading the configuration file
	return config.LoadConfig(flag.Arg(0))
}
