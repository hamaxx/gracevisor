package main

import (
	"flag"
	"log"
	"net/http"
	"sync"

	"github.com/hamaxx/gracevisor/deps/lumberjack"
)

var defaultConfigDir = "/etc/gracevisor/"
var configPath = flag.String("conf", defaultConfigDir, "path to config dir")

func configureGracevisorLogger(config *LoggerConfig) {
	writer := &lumberjack.Logger{
		Filename:   config.LogFile,
		MaxSize:    config.MaxLogSize,
		MaxAge:     config.MaxLogAge,
		MaxBackups: config.MaxLogsKept,
	}

	log.SetOutput(writer)
}

func main() {
	flag.Parse()

	config, err := ParseConfing(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	configureGracevisorLogger(config.Logger)

	portPool := NewPortPool(config.PortRange.From, config.PortRange.To)
	runningApps := map[string]*App{}

	appWg := sync.WaitGroup{}
	for _, appConfig := range config.Apps {
		appWg.Add(1)
		app := NewApp(appConfig, config.Logger, portPool)
		runningApps[app.config.Name] = app
		go func() {
			if err := app.StartNewInstance(); err != nil {
				log.Print("Start new instance error:", err)
				return
			}
			if err := app.ListenAndServe(); err != nil {
				log.Print("App listen and serve error:", err)
			}
			appWg.Done()
		}()
	}

	rpcListener, err := NewRpcServer(runningApps, config.Rpc)
	if err != nil {
		log.Fatal(err)
	}
	if err := http.Serve(rpcListener, nil); err != nil {
		log.Print("Rpc server error:", err)
	}

	appWg.Wait()

}
