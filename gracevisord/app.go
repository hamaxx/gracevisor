package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"
	"time"
)

var ErrNoActiveInstances = errors.New("No active instances")

type App struct {
	config *AppConfig

	instances          []*Instance
	activeInstance     *Instance
	activeInstanceLock sync.Mutex

	rp       *httputil.ReverseProxy
	portPool *PortPool

	externalHostPort string

	instanceId uint32
}

func NewApp(config *AppConfig, portPool *PortPool) *App {
	app := &App{
		config:           config,
		instances:        make([]*Instance, 0, 3),
		portPool:         portPool,
		externalHostPort: fmt.Sprintf("%s:%d", config.ExternalHost, config.ExternalPort),
	}

	app.rp = &httputil.ReverseProxy{Director: func(req *http.Request) {}}

	app.startInstanceUpdater()

	return app
}

func (a *App) startInstanceUpdater() {
	ticker := time.NewTicker(time.Second)

	go func() {
		for {
			starting := false
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
				} else if status == InstanceStatusExited && instance == a.activeInstance {
					a.activeInstance = nil
				} else if status == InstanceStatusStarting {
					starting = true
				}
			}

			if a.activeInstance == nil && !starting {
				// TODO retry count
				err := a.StartNewInstance()
				if err != nil {
					log.Print(err)
				}
			}

			<-ticker.C
		}
	}()
}

func (a *App) reserveInstance() (*Instance, error) {
	a.activeInstanceLock.Lock()

	instance := a.activeInstance
	if instance == nil {
		return nil, ErrNoActiveInstances
	}
	instance.Serve()

	a.activeInstanceLock.Unlock()

	return instance, nil
}

func (a *App) StartNewInstance() error {
	newInstance, err := NewInstance(a, atomic.AddUint32(&a.instanceId, 1))
	if err != nil {
		return err
	}

	a.instances = append(a.instances, newInstance)
	return nil
}

func (a *App) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	instance, err := a.reserveInstance()

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

		if instance != nil {
			instance.Done()
		}
	}()

	if err != nil {
		panic(err)
	}

	req.URL.Scheme = "http"
	req.URL.Host = instance.internalHostPort

	host, _, _ := net.SplitHostPort(req.RemoteAddr) //TODO parse real real ip, add fwd for
	req.Header.Add("X-Real-IP", host)

	a.rp.ServeHTTP(rw, req)
}

func (a *App) ListenAndServe() {
	http.ListenAndServe(a.externalHostPort, a)
}

func (a *App) Report() string {
	report := ""

	displayN := 3

	l := len(a.instances)
	from := 0
	if l > displayN {
		from = l - displayN
	}

	report += fmt.Sprintf("[%s/%s]\n", a.config.Name, a.externalHostPort)
	for _, instance := range a.instances[from:l] {
		if instance == a.activeInstance {
			report += fmt.Sprint(" * ")
		} else {
			report += fmt.Sprint("   ")
		}
		report += fmt.Sprintf("%d/%s ", instance.id, instance.internalHostPort)
		switch instance.status {
		case InstanceStatusServing:
			report += fmt.Sprint("serving ")
		case InstanceStatusStarting:
			report += fmt.Sprint("starting")
		case InstanceStatusStopping:
			report += fmt.Sprint("stopping")
		case InstanceStatusStopped:
			report += fmt.Sprint("stopped ")
		case InstanceStatusFailed:
			report += fmt.Sprint("failed  ")
		case InstanceStatusExited:
			report += fmt.Sprint("exited  ")
		}
		report += fmt.Sprintf(" %s", time.Since(instance.lastChange)/time.Second*time.Second)
		if instance.processErr != nil {
			report += fmt.Sprintf(" %q", instance.processErr)
		}
		report += fmt.Sprintln()
	}
	return report
}
