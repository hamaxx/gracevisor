package main

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"sort"
)

var ErrInvalidApp = errors.New("Invalid app")

type AppNameSort []*App

func (v AppNameSort) Len() int {
	return len(v)
}
func (v AppNameSort) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}
func (v AppNameSort) Less(i, j int) bool {
	return v[i].config.Name < v[j].config.Name
}

type Rpc struct {
	runningApps map[string]*App
}

func (r *Rpc) Restart(appName string, res *string) error {
	app, ok := r.runningApps[appName]
	if !ok {
		return ErrInvalidApp
	}
	return app.StartNewInstance()
}

func (r *Rpc) Start(appName string, res *string) error {
	app, ok := r.runningApps[appName]
	if !ok {
		return ErrInvalidApp
	}
	return app.StartNewInstance()
}

func (r *Rpc) Stop(appName string, res *string) error {
	app, ok := r.runningApps[appName]
	if !ok {
		return ErrInvalidApp
	}
	return app.StopInstances(-1, false)
}

func (r *Rpc) Kill(appName string, res *string) error {
	app, ok := r.runningApps[appName]
	if !ok {
		return ErrInvalidApp
	}
	return app.StopInstances(-1, true)
}

func (r *Rpc) Status(appName string, res *string) error {
	if appName != "" {
		app, ok := r.runningApps[appName]
		if !ok {
			return ErrInvalidApp
		}
		*res = app.Report(10)
	} else {
		sortedApps := []*App{}
		for _, app := range r.runningApps {
			sortedApps = append(sortedApps, app)
		}
		sort.Sort(AppNameSort(sortedApps))
		for _, app := range sortedApps {
			*res += app.Report(3)
		}
	}
	return nil
}

func NewRpcServer(runningApps map[string]*App, config *RpcConfig) (net.Listener, error) {

	r := &Rpc{
		runningApps: runningApps,
	}

	if err := rpc.Register(r); err != nil {
		return nil, err
	}
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port))
	if e != nil {
		return nil, e
	}
	return l, nil
}
