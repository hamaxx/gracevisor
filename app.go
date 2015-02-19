package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

var ErrNoActiveInstances = errors.New("No active instances")

type AppConfig struct {
	name        string
	command     string
	healthcheck string
	stopSignal  int
	timeout     int

	internalHost string
	externalHost string
	externalPort uint32
}

type App struct {
	config *AppConfig

	instances          []*Instance
	activeInstance     *Instance
	activeInstanceLock sync.Mutex

	requestInstance     map[*http.Request]*Instance
	requestInstanceLock sync.Mutex

	rp       *httputil.ReverseProxy
	portPool *PortPool
}

func NewApp(config *AppConfig, portPool *PortPool) *App {
	app := &App{
		config: config,

		instances:       make([]*Instance, 0, 3),
		requestInstance: make(map[*http.Request]*Instance, 100),

		portPool: portPool,
	}

	app.rp = &httputil.ReverseProxy{Director: app.reverseProxyDirector}

	app.startInstanceUpdater()

	return app
}

func (a *App) startInstanceUpdater() {
	ticker := time.NewTicker(time.Second)

	go func() {
		for {
			for _, instance := range a.instances {
				status := instance.UpdateStatus()
				if status == InstanceStatusServing && instance != a.activeInstance {
					a.activeInstanceLock.Lock()
					currentActive := a.activeInstance
					a.activeInstance = instance
					a.activeInstanceLock.Unlock()

					if currentActive != nil {
						currentActive.Stop()
					}
				}
			}
			a.Report()
			<-ticker.C
		}
	}()
}

func (a *App) reserveInstance(req *http.Request) (*Instance, error) {
	a.activeInstanceLock.Lock()
	defer a.activeInstanceLock.Unlock()

	if a.activeInstance == nil {
		return nil, ErrNoActiveInstances
	}

	a.activeInstance.Serve()

	a.requestInstanceLock.Lock()
	a.requestInstance[req] = a.activeInstance
	a.requestInstanceLock.Unlock()

	return a.activeInstance, nil
}

func (a *App) releaseInstance(req *http.Request) {
	a.requestInstanceLock.Lock()
	defer a.requestInstanceLock.Unlock()

	instance, ok := a.requestInstance[req]
	if ok {
		instance.Done()
		delete(a.requestInstance, req)
	}
}

func (a *App) reverseProxyDirector(req *http.Request) {
	instance, err := a.reserveInstance(req)
	if err != nil {
		panic(err)
	}

	req.URL.Scheme = "http"
	req.URL.Host = instance.Hostname()

	host, _, _ := net.SplitHostPort(req.RemoteAddr) //TODO parse real real ip, add fwd for
	req.Header.Add("X-Real-IP", host)
}

func (a *App) StartNewInstance() error {
	for _, instance := range a.instances {
		if instance.Status() == InstanceStatusStarting {
			instance.Stop()
		}
	}

	newInstance, err := NewInstance(a)
	if err != nil {
		return err
	}

	a.instances = append(a.instances, newInstance)
	return nil
}

func (a *App) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			if err == ErrNoActiveInstances {
				log.Print(err)
				rw.WriteHeader(503)
				req.Body.Close()
			} else {
				log.Print(err)
			}
		}

		a.releaseInstance(req)
	}()

	a.rp.ServeHTTP(rw, req)
}

func (a *App) Hostname() string {
	return fmt.Sprintf("%s:%d", a.config.externalHost, a.config.externalPort)
}

func (a *App) ListenAndServe() {
	http.ListenAndServe(a.Hostname(), a)
}

func (a *App) Report() {
	fmt.Printf("[%s/%s]\n", a.config.name, a.Hostname())
	for _, instance := range a.instances {
		if instance == a.activeInstance {
			fmt.Print(" * ")
		} else {
			fmt.Print("   ")
		}
		fmt.Printf("%s ", instance.Hostname())
		switch instance.Status() {
		case InstanceStatusServing:
			fmt.Print("serving ")
		case InstanceStatusStarting:
			fmt.Print("starting")
		case InstanceStatusStopping:
			fmt.Print("stopping")
		case InstanceStatusStopped:
			fmt.Print("stopped ")
		case InstanceStatusFailed:
			fmt.Print("failed  ")
		}
		fmt.Printf(" %s", instance.lastChange.String())
		fmt.Println()
	}
}
