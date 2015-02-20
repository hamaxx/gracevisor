package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	InstanceStatusServing = iota
	InstanceStatusStarting
	InstanceStatusStopping
	InstanceStatusStopped
	InstanceStatusExited
	InstanceStatusFailed
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

	exec         *exec.Cmd
	processErr   error
	processState *os.ProcessState
}

func NewInstance(app *App, id uint32) (*Instance, error) {
	port, err := app.portPool.ReserveNewPort()
	if err != nil {
		return nil, err
	}

	cmd, attrs := parseCommand(app.config.command, port)

	instance := &Instance{
		id:               id,
		app:              app,
		internalHost:     app.config.internalHost,
		internalPort:     port,
		internalHostPort: fmt.Sprintf("%s:%d", app.config.internalHost, port),
		status:           InstanceStatusStarting,
		exec:             exec.Command(cmd, attrs...),
		connWg:           &sync.WaitGroup{},
		lastChange:       time.Now(),
	}
	instance.processErr = instance.exec.Start()

	go func() {
		if instance.exec.Process != nil {
			state, err := instance.exec.Process.Wait()
			instance.processErr = err
			instance.processState = state
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
			i.exec.Process.Signal(syscall.SIGTERM) // TODO: configurable signal, kill on timeout
		}
	}()

}
func (i *Instance) Kill() {
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

func (i *Instance) healthCheck() int {
	if i.app.config.healthcheck == "" {
		return InstanceStatusServing
	}
	return InstanceStatusServing // TODO
}

func (i *Instance) checkProcessStartupStatus() int {
	if i.processErr != nil {
		log.Print(i.processErr)
		return InstanceStatusFailed
	}
	if i.exec.Process == nil {
		return InstanceStatusStarting
	}

	return i.healthCheck()
}

func (i *Instance) checkProcessStoppingStatus() int {
	if i.processErr != nil || i.exec.Process == nil {
		log.Print(i.processErr)
		return InstanceStatusExited
	}
	if i.processState != nil && i.processState.Exited() {
		return InstanceStatusStopped
	}

	return InstanceStatusStopping
}

func (i *Instance) checkProcessRunningStatus() int {
	if i.processErr != nil || i.exec.Process == nil {
		log.Print(i.processErr)
		return InstanceStatusExited
	}
	if i.processState != nil && i.processState.Exited() {
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
