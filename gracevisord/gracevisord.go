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
		app := NewApp(appConfig, portPool)
		runningApps[app.config.Name] = app
		go func() {
			app.StartNewInstance()
			app.ListenAndServe()
			appWg.Done()
		}()
	}

	rpcListener, err := NewRpcServer(runningApps, config.Rpc)
	if err != nil {
		log.Fatal(err)
	}
	http.Serve(rpcListener, nil)

	appWg.Wait()

}
