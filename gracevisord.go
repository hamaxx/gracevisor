package main

func main() {
	app := NewApp(&AppConfig{
		name:         "test",
		command:      "./b1",
		healthcheck:  "/HealthCheck",
		externalHost: "localhost:8080",
	},
	)

	app.StartNewInstance()
	app.ListenAndServe()
}
