package main

import (
	"fmt"
	"os/exec"
	"sync"
	"time"
)

const (
	InstanceStatusServing = iota
	InstanceStatusStarting
	InstanceStatusStopping
	InstanceStatusStopped
	InstanceStatusFailed
)

type Instance struct {
	app *App

	internalHost string
	internalPort uint32
	status       int
	exec         *exec.Cmd
	connWg       *sync.WaitGroup
	lastChange   time.Time
}

func NewInstance(app *App) (*Instance, error) {
	port, err := app.portPool.ReserveNewPort()
	if err != nil {
		return nil, err
	}

	instance := &Instance{
		internalHost: app.config.internalHost,
		internalPort: port,
		app:          app,
		status:       InstanceStatusStarting,
		exec:         exec.Command(app.config.command),
		connWg:       &sync.WaitGroup{},
		lastChange:   time.Now(),
	}
	instance.exec.Start()

	return instance, nil
}

func (i *Instance) Status() int {
	return i.status
}
func (i *Instance) Stop() {
	i.status = InstanceStatusStopping
	i.lastChange = time.Now()
	go func() {
		i.connWg.Wait()
		// TODO stop signal
	}()

}
func (i *Instance) Kill() {
	i.lastChange = time.Now()
	// TODO
}

func (i *Instance) Hostname() string {
	hostPort := fmt.Sprintf("%s:%d", i.internalHost, i.internalPort)
	return hostPort
}

func (i *Instance) Serve() {
	i.connWg.Add(1)
}

func (i *Instance) Done() {
	i.connWg.Done()
}

func (i *Instance) UpdateStatus() int {
	// TODO
	if i.status == InstanceStatusStarting {
		i.status = InstanceStatusServing
		i.lastChange = time.Now()
	}
	return i.status
}
