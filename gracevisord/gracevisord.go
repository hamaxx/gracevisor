package main

import (
	"flag"
	"log"
	"net/http"
	"sync"
)

var defaultConfigDir = "/etc/gracevisor/"
var configPath = flag.String("conf", defaultConfigDir, "path to config dir")

func main() {
	flag.Parse()

	config, err := ParseConfing(*configPath)

	if err != nil {
		log.Print(err)
		return
	}

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
