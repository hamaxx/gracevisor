package main

import (
	"errors"
	"sync"
)

var ErrNoAvailablePorts = errors.New("No available ports")

type PortPool struct {
	portStart uint32
	portEnd   uint32

	current uint32
	mu      sync.Mutex

	usedPorts map[uint32]struct{}
}

func NewPortPool(start, end uint32) *PortPool {
	return &PortPool{
		portStart: start,
		portEnd:   end,
		current:   start - 1,
		usedPorts: make(map[uint32]struct{}, end-start),
	}
}

func (p *PortPool) ReserveNewPort() (uint32, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := uint32(0); i < p.portEnd-p.portStart; i++ {
		p.current = (p.current + 1) % (p.portEnd - p.portStart)
		port := p.portStart + p.current

		_, used := p.usedPorts[port]
		if !used {
			p.usedPorts[port] = struct{}{}
			return port, nil
		}
	}
	return 0, ErrNoAvailablePorts
}

func (p *PortPool) ReleasePort(port uint32) {
	p.mu.Lock()
	delete(p.usedPorts, port)
	p.mu.Unlock()
}
