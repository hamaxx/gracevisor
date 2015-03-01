package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"
	"text/tabwriter"
	"time"
)

var (
	ErrNoActiveInstances  = errors.New("No active instances")
	ErrInstanceNotRunning = errors.New("Instance is not running")
)

type App struct {
	config       *AppConfig
	loggerConfig *LoggerConfig

	instances          []*Instance
	activeInstance     *Instance
	activeInstanceLock sync.Mutex

	rp       *httputil.ReverseProxy
	portPool *PortPool

	externalHostPort string

	instanceId uint32

	appLogger *AppLogger
}

func NewApp(config *AppConfig, loggerConfig *LoggerConfig, portPool *PortPool) *App {
	app := &App{
		config:           config,
		loggerConfig:     loggerConfig,
		instances:        make([]*Instance, 0, 10),
		portPool:         portPool,
		externalHostPort: fmt.Sprintf("%s:%d", config.ExternalHost, config.ExternalPort),
	}

	app.appLogger = NewAppLogger(app)
	app.rp = &httputil.ReverseProxy{Director: func(req *http.Request) {}}

	app.startInstanceUpdater()

	return app
}

func (a *App) startInstanceUpdater() {
	ticker := time.NewTicker(time.Second)

	restartCount := 0

	go func() {
		// TODO refactor this. Instances should trigger status changes.
		for {
			lastStatus := -1

			for _, instance := range a.instances {
				status := instance.UpdateStatus()
				lastStatus = status

				if instance == a.activeInstance {
					if status != InstanceStatusServing {
						a.activeInstance = nil
					}
				} else {
					if status == InstanceStatusServing {
						restartCount = 0
						a.activeInstanceLock.Lock()
						currentActive := a.activeInstance
						a.activeInstance = instance
						a.activeInstanceLock.Unlock()

						if currentActive != nil {
							currentActive.Stop()
						}
					}
				}
			}

			if lastStatus == InstanceStatusExited || lastStatus == InstanceStatusFailed || lastStatus == InstanceStatusTimedOut {
				if restartCount < a.config.MaxRetries {
					restartCount++
					err := a.StartNewInstance()
					if err != nil {
						log.Print(err)
					}
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

func (a *App) StopInstances(instanceId int, kill bool) error {
	stopped := false
	for _, instance := range a.instances {
		if instanceId > 0 && int(instance.id) != instanceId {
			continue
		}
		if instance.status == InstanceStatusServing || instance.status == InstanceStatusStarting {
			stopped = true
			if kill {
				instance.Kill()
			} else {
				instance.Stop()
			}
		}
	}
	if !stopped {
		return ErrInstanceNotRunning
	}
	return nil
}

func (a *App) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	instance, err := a.reserveInstance()
	defer func() {
		if instance != nil {
			instance.Done()
		}
	}()
	if err != nil {
		if err == ErrNoActiveInstances {
			rw.WriteHeader(503)
			if err := req.Body.Close(); err != nil {
				log.Print(err)
			}
		} else {
			log.Print(err)
		}
		return
	}

	req.URL.Scheme = "http"
	req.URL.Host = instance.internalHostPort

	host, _, _ := net.SplitHostPort(req.RemoteAddr) //TODO parse real real ip, add fwd for
	req.Header.Add("X-Real-IP", host)

	a.rp.ServeHTTP(rw, req)
}

func (a *App) ListenAndServe() error {
	return http.ListenAndServe(a.externalHostPort, a)
}

func (a *App) Report(displayN int) string {
	writer := &bytes.Buffer{}
	tabWriter := tabwriter.NewWriter(writer, 2, 2, 1, ' ', 0)
	writeColumn := func(s string, f ...interface{}) {
		if _, err := tabWriter.Write([]byte(fmt.Sprintf(s, f...))); err != nil {
			log.Print(err)
		}
		if _, err := tabWriter.Write([]byte("\t")); err != nil {
			log.Print(err)
		}
	}

	if _, err := tabWriter.Write([]byte(fmt.Sprintf("[%s/%s]\n", a.config.Name, a.externalHostPort))); err != nil {
		log.Print(err)
	}

	from := 0
	if len(a.instances) > displayN {
		from = len(a.instances) - displayN
	}

	for _, instance := range a.instances[from:len(a.instances)] {
		if instance == a.activeInstance {
			writeColumn(" *")
		} else {
			writeColumn("")
		}
		writeColumn("%d/%s", instance.id, instance.internalHostPort)
		switch instance.status {
		case InstanceStatusServing:
			writeColumn("serving")
		case InstanceStatusStarting:
			writeColumn("starting")
		case InstanceStatusStopping:
			writeColumn("stopping")
		case InstanceStatusStopped:
			writeColumn("stopped")
		case InstanceStatusKilled:
			writeColumn("killed")
		case InstanceStatusFailed:
			writeColumn("failed")
		case InstanceStatusExited:
			writeColumn("exited")
		case InstanceStatusTimedOut:
			writeColumn("timed out")
		}
		writeColumn("%s", time.Since(instance.lastChange)/time.Second*time.Second)
		if instance.processErr != nil {
			writeColumn("%q", instance.processErr)
		}
		if _, err := tabWriter.Write([]byte("\n")); err != nil {
			log.Print(err)
		}
	}
	if err := tabWriter.Flush(); err != nil {
		log.Print(err)
	}
	return writer.String()
}
