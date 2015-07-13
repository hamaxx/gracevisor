package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/hamaxx/gracevisor/deps/cli"
	"github.com/hamaxx/gracevisor/deps/lumberjack"
)

var defaultConfigDir = "/etc/gracevisor/"

func configureGracevisorLogger(config *LoggerConfig) {
	writer := &lumberjack.Logger{
		Filename:   config.LogFile,
		MaxSize:    config.MaxLogSize,
		MaxAge:     config.MaxLogAge,
		MaxBackups: config.MaxLogsKept,
	}

	log.SetOutput(writer)
}

func startApp(config *Config) {
	portPool := NewPortPool(config.PortRange.From, config.PortRange.To)
	runningApps := map[string]*App{}

	appWg := sync.WaitGroup{}
	for _, appConfig := range config.Apps {
		appWg.Add(1)
		app := NewApp(appConfig, portPool)
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

func main() {
	MaybeBecomeChildProcess()

	// solution for https://github.com/golang/go/issues/6785
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	app := cli.NewApp()
	app.Name = "gracevisord"
	app.Usage = "gracevisor daemon"
	app.Email = "jure@hamsworld.net"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "conf",
			Value: defaultConfigDir,
			Usage: "path to config dir",
		},
	}
	app.Action = func(c *cli.Context) {
		config, err := ParseConfing(c.String("conf"))
		if err != nil {
			log.Fatal(err)
		}

		configureGracevisorLogger(config.Logger)
		startApp(config)
	}
	app.Run(os.Args)
}
