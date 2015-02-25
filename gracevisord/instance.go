package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	InstanceStatusServing = iota
	InstanceStatusStarting
	InstanceStatusStopping
	InstanceStatusStopped
	InstanceStatusKilled
	InstanceStatusExited
	InstanceStatusFailed
	InstanceStatusTimedOut
)

const (
	HealthCheckTimeout = 1
)

var (
	ErrInvalidStopSignal = errors.New("Invalid stop signal")
)

type Instance struct {
	app *App
	id  uint32

	internalHost     string
	internalPort     uint32
	internalHostPort string
	status           int
	lastChange       time.Time

	connWg    *sync.WaitGroup
	connCount int32

	exec             *exec.Cmd
	processErr       error
	processExitState *os.ProcessState

	instanceLogger *InstanceLogger
}

func NewInstance(app *App, id uint32) (*Instance, error) {
	port, err := app.portPool.ReserveNewPort()
	if err != nil {
		return nil, err
	}

	cmd, attrs := parseCommand(app.config.Command, port)

	instance := &Instance{
		id:               id,
		app:              app,
		internalHost:     app.config.InternalHost,
		internalPort:     port,
		internalHostPort: fmt.Sprintf("%s:%d", app.config.InternalHost, port),
		status:           InstanceStatusStarting,
		exec:             exec.Command(cmd, attrs...),
		connWg:           &sync.WaitGroup{},
		lastChange:       time.Now(),
	}
	instance.instanceLogger, err = NewInstanceLogger(instance)
	if err != nil {
		return nil, err
	}
	instance.processErr = instance.exec.Start()

	go func() {
		if instance.exec.Process != nil {
			state, err := instance.exec.Process.Wait()
			instance.processErr = err
			instance.processExitState = state
		}
	}()

	return instance, nil
}

func parseCommand(cmd string, port uint32) (string, []string) {
	withPort := strings.Replace(cmd, "{port}", fmt.Sprint(port), -1)
	command := strings.Split(withPort, " ")
	return command[0], command[1:]
}

func (i *Instance) Stop() {
	i.status = InstanceStatusStopping
	i.lastChange = time.Now()
	go func() {
		i.connWg.Wait()
		if i.exec.Process != nil {
			signal, ok := Signals[i.app.config.StopSignal]
			if !ok {
				log.Print(ErrInvalidStopSignal)
				return
			}
			i.exec.Process.Signal(signal)
		}
	}()

}

func (i *Instance) Kill() {
	i.status = InstanceStatusStopping
	i.lastChange = time.Now()
	if i.exec.Process != nil {
		i.processErr = i.exec.Process.Kill()
	}
}

func (i *Instance) Serve() {
	i.connWg.Add(1)
	atomic.AddInt32(&i.connCount, 1)
}

func (i *Instance) Done() {
	i.connWg.Done()
	atomic.AddInt32(&i.connCount, -1)
}

func (i *Instance) healthCheck() bool {
	if i.app.config.HealthCheck == "" {
		return true
	}

	healthCheckUrl := url.URL{
		Scheme: "http",
		Host:   i.internalHostPort,
		Path:   i.app.config.HealthCheck,
	}

	resp, err := http.Get(healthCheckUrl.String())
	if err != nil || resp.StatusCode != 200 {
		return false
	}
	resp.Body.Close()

	return true
}

func (i *Instance) checkProcessStartupStatus() int {
	if i.processExitState != nil || i.processErr != nil {
		log.Print("aa", i.processErr, i.processExitState)
		return InstanceStatusFailed
	}

	if i.app.config.StopTimeout > 0 && time.Since(i.lastChange) > time.Duration(i.app.config.StartTimeout)*time.Second {
		if i.exec.Process != nil {
			i.processErr = i.exec.Process.Kill()
		}
		return InstanceStatusTimedOut
	}

	if i.exec.Process == nil {
		return InstanceStatusStarting
	}

	if i.healthCheck() {
		return InstanceStatusServing
	}
	return InstanceStatusStarting
}

func (i *Instance) checkProcessStoppingStatus() int {
	if i.processErr != nil || i.exec.Process == nil {
		log.Print(i.processErr)
		return InstanceStatusExited
	}
	if i.processExitState != nil {
		if s, ok := i.processExitState.Sys().(int); ok && s == 9 {
			return InstanceStatusKilled
		}
		return InstanceStatusStopped
	}

	if i.app.config.StopTimeout > 0 && time.Since(i.lastChange) > time.Duration(i.app.config.StopTimeout)*time.Second {
		i.processErr = i.exec.Process.Kill()
		return InstanceStatusKilled
	}

	return InstanceStatusStopping
}

func (i *Instance) checkProcessRunningStatus() int {
	if i.processErr != nil || i.exec.Process == nil || i.processExitState != nil {
		log.Printf("%s:%s", i.processExitState, i.processErr)
		return InstanceStatusExited
	}

	return InstanceStatusServing
}

func (i *Instance) UpdateStatus() int {
	if i.status == InstanceStatusStarting {
		status := i.checkProcessStartupStatus()
		if status != i.status {
			i.status = status
			i.lastChange = time.Now()
		}
	} else if i.status == InstanceStatusStopping {
		status := i.checkProcessStoppingStatus()
		if status != i.status {
			i.status = status
			i.lastChange = time.Now()
		}
	} else if i.status == InstanceStatusServing {
		status := i.checkProcessRunningStatus()
		if status != i.status {
			i.status = status
			i.lastChange = time.Now()
		}
	}
	return i.status
}
