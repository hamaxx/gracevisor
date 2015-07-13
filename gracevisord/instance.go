package main

import (
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
	PortBadge          = "{port}"
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

	cmd              *exec.Cmd
	processErr       error
	processExitState *os.ProcessState

	instanceLogger *InstanceLogger
}

func NewInstance(app *App, id uint32) (*Instance, error) {
	port, err := app.portPool.ReserveNewPort()
	if err != nil {
		return nil, err
	}

	instance := &Instance{
		id:               id,
		app:              app,
		internalHost:     app.config.InternalHost,
		internalPort:     port,
		internalHostPort: fmt.Sprintf("%s:%d", app.config.InternalHost, port),
		status:           InstanceStatusStarting,
		connWg:           &sync.WaitGroup{},
		lastChange:       time.Now(),
	}

	// prepare command arguments
	cmdPath, cmdArgs := parseCommand(parsePortBadge(app.config.Command, port))

	environment := make([]string, 0, len(app.config.Environment))
	for _, env := range app.config.Environment {
		environment = append(environment, parsePortBadge(env, port))
	}

	// start command
	gvCmd, err := NewGvCmd(
		cmdPath,
		environment,
		cmdArgs,
		app.config.Directory,
		app.config.User.Uid,
	)
	if err != nil {
		return nil, err
	}
	cmd, outPipe, errPipe, err := gvCmd.start()
	instance.cmd = cmd
	instance.processErr = err

	// init logger
	instance.instanceLogger, err = NewInstanceLogger(instance, outPipe, errPipe)
	if err != nil {
		return nil, err
	}

	// wait for process to exit and update process state
	go func() {
		if instance.cmd.Process != nil {
			state, err := instance.cmd.Process.Wait()
			instance.processErr = err
			instance.processExitState = state
		}
	}()

	return instance, nil
}

func parsePortBadge(input string, port uint32) string {
	return strings.Replace(input, PortBadge, fmt.Sprint(port), -1)
}

func parseCommand(cmd string) (string, []string) {
	command := strings.Split(cmd, " ")
	return command[0], command
}

func (i *Instance) Stop() {
	i.status = InstanceStatusStopping
	i.lastChange = time.Now()

	// wait for all http requests to finish
	go func() {
		i.connWg.Wait()
		if i.cmd.Process != nil {
			if err := i.cmd.Process.Signal(i.app.config.StopSignal); err != nil {
				log.Print("Stop signal error:", err)
				return
			}
		}
	}()

}

func (i *Instance) Kill() {
	i.status = InstanceStatusStopping
	i.lastChange = time.Now()
	if i.cmd.Process != nil {
		i.processErr = i.cmd.Process.Kill()
	}
}

// Serve registers active http request
func (i *Instance) Serve() {
	i.connWg.Add(1)
	atomic.AddInt32(&i.connCount, 1)
}

// Done finishes active http request
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
	if err := resp.Body.Close(); err != nil {
		log.Print(err)
	}

	return true
}

func (i *Instance) checkProcessStartupStatus() int {
	if i.processExitState != nil || i.processErr != nil {
		log.Print("Process exited on startup", i.processErr, i.processExitState)
		return InstanceStatusFailed
	}

	if i.app.config.StopTimeout > 0 && time.Since(i.lastChange) > time.Duration(i.app.config.StartTimeout)*time.Second {
		if i.cmd.Process != nil {
			i.processErr = i.cmd.Process.Kill()
		}
		return InstanceStatusTimedOut
	}

	if i.cmd.Process == nil {
		return InstanceStatusStarting
	}

	if i.healthCheck() {
		return InstanceStatusServing
	}
	return InstanceStatusStarting
}

func (i *Instance) checkProcessStoppingStatus() int {
	if i.processErr != nil || i.cmd.Process == nil {
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
		i.processErr = i.cmd.Process.Kill()
		return InstanceStatusKilled
	}

	return InstanceStatusStopping
}

func (i *Instance) checkProcessRunningStatus() int {
	if i.processErr != nil || i.cmd.Process == nil || i.processExitState != nil {
		log.Printf("%s:%s", i.processExitState, i.processErr)
		return InstanceStatusExited
	}

	return InstanceStatusServing
}

// UpdateStatus is called from app every second for status update
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
