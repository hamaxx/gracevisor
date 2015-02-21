package main

import (
	"flag"
	"log"
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

	appWg := sync.WaitGroup{}

	portPool := NewPortPool(config.Port.From, config.Port.To)

	for _, appConfig := range config.Apps {
		appWg.Add(1)
		go func(appConfig *AppConfig) {
			app := NewApp(appConfig, portPool)
			app.ListenAndServe()
			appWg.Done()
		}(appConfig)
	}

	appWg.Wait()

}
