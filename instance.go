package main

import (
	"os/exec"
	"sync"
)

const (
	InstanceStatusServing = iota
	InstanceStatusStarting
	InstanceStatusStopping
	InstanceStatusStopped
	InstanceStatusFailed
)

type Instance struct {
	hostname string
	status   int
	exec     *exec.Cmd
	connWg   *sync.WaitGroup
}

func NewInstance(app *App) *Instance {
	newHostname := "localhost:8001"

	instance := &Instance{
		hostname: newHostname,
		status:   InstanceStatusStarting,
		exec:     exec.Command(app.config.command),
		connWg:   &sync.WaitGroup{},
	}
	instance.exec.Start()

	return instance
}

func (i *Instance) Status() int {
	return i.status
}
func (i *Instance) Stop() {
	i.status = InstanceStatusStopping
	go func() {
		i.connWg.Wait()
		// TODO stop signal
	}()

}
func (i *Instance) Kill() {
	// TODO
}

func (i *Instance) Hostname() string {
	return i.hostname
}

func (i *Instance) serve() {
	i.connWg.Add(1)
}

func (i *Instance) Done() {
	i.connWg.Done()
}

func (i *Instance) UpdateStatus(app *App) int {
	// TODO
	i.status = InstanceStatusServing
	return InstanceStatusServing
}
