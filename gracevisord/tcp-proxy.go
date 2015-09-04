package main

import (
	"io"
	"log"
	"sync"
	"time"

	"net"
)

var MaxConnections = 4096
var IdleTimeout = time.Second * 30

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 0xffff)
	},
}

type TcpProxy struct {
	App *App

	throttle chan struct{}
}

func NewTcpProxy(a *App) *TcpProxy {
	p := &TcpProxy{
		App:      a,
		throttle: make(chan struct{}, MaxConnections),
	}
	return p
}

func (p *TcpProxy) processConn(conn *net.TCPConn) {
	defer func() { <-p.throttle }()

	instance, err := p.App.reserveInstance()
	if err != nil {
		log.Print(err)
		conn.Close()
		return
	}

	raddr, err := net.ResolveTCPAddr("tcp", instance.internalHostPort)

	p.start(conn, raddr)
	instance.Done()
}

func (p *TcpProxy) ServeTcp() error {
	laddr, err := net.ResolveTCPAddr("tcp", p.App.externalHostPort)
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}

	for {
		p.throttle <- struct{}{}
		conn, err := listener.AcceptTCP()
		conn.SetDeadline(time.Now().Add(IdleTimeout))

		if err != nil {
			log.Printf("Failed to accept connection '%s'\n", err)
			continue
		}
		go p.processConn(conn)
	}
}

func (p *TcpProxy) start(lconn *net.TCPConn, raddr *net.TCPAddr) {
	defer lconn.Close()

	rconn, err := net.DialTCP("tcp", nil, raddr)
	rconn.SetDeadline(time.Now().Add(IdleTimeout))
	defer rconn.Close()

	if err != nil {
		log.Printf("Remote connection failed: %s", err)
		return
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go p.connCopy(1, rconn, lconn, wg)
	go p.connCopy(2, lconn, rconn, wg)

	wg.Wait()
}

func (p *TcpProxy) connCopy(t int, dst, src *net.TCPConn, wg *sync.WaitGroup) {
	buf := bufferPool.Get().([]byte)

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if ew != nil {
				log.Print(ew)
				break
			}
			if nr != nw {
				log.Print(io.ErrShortWrite)
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			log.Print(er)
			break
		}

		if nr > 0 {
			dst.SetDeadline(time.Now().Add(IdleTimeout))
			src.SetDeadline(time.Now().Add(IdleTimeout))
		}
	}
	src.CloseRead()
	dst.CloseWrite()

	bufferPool.Put(buf[:])
	wg.Done()
}
