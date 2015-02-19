package main

import "time"

func main() {
	app := NewApp(&AppConfig{
		name:         "test",
		command:      "../demoapp/demoapp --port={port}",
		healthcheck:  "/HealthCheck",
		externalHost: "localhost",
		externalPort: 8080,
		internalHost: "localhost",
	},
		NewPortPool(10000, 11000),
	)

	go func() {
		app.StartNewInstance()
		time.Sleep(time.Second * time.Duration(2))
		app.StartNewInstance()
		time.Sleep(time.Second * time.Duration(10))
		app.StartNewInstance()
	}()

	app.ListenAndServe()
}
