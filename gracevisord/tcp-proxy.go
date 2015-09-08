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

func (p *TcpProxy) processConn(lconn *net.TCPConn) {
	defer func() {
		lconn.Close()
		<-p.throttle
		//log.Printf("Num open connections: %d", len(p.throttle))
	}()

	instance, err := p.App.reserveInstance()
	if err != nil {
		log.Print(err)
		return
	}

	raddr, err := net.ResolveTCPAddr("tcp", instance.internalHostPort)
	if err != nil {
		log.Printf("Remote connection failed: %s", err)
		return
	}
	rconn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Printf("Remote connection failed: %s", err)
		return
	}

	p.connHandler(lconn, rconn)
	instance.Done()

	rconn.Close()
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

		if err != nil {
			log.Printf("Failed to accept connection '%s'\n", err)
			continue
		}
		go p.processConn(conn)
	}
}

func (p *TcpProxy) connHandler(lconn, rconn *net.TCPConn) {
	dl := time.Now().Add(IdleTimeout)

	rconn.SetDeadline(dl)
	lconn.SetDeadline(dl)

	doneLocal := make(chan bool)
	doneRemote := make(chan bool)

	go p.connCopy(rconn, lconn, dl, doneLocal)
	go p.connCopy(lconn, rconn, dl, doneRemote)

	for i := 0; i < 2; i++ {
		select {
		case <-doneLocal:
			rconn.CloseRead()
		case <-doneRemote:
			lconn.CloseRead()
		}
	}
}

func (p *TcpProxy) connCopy(dst, src *net.TCPConn, dl time.Time, done chan bool) {
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

		dlNew := time.Now().Add(IdleTimeout)
		if nr > 0 && dlNew.Sub(dl) > time.Second {
			dl = dlNew
			dst.SetDeadline(dl)
			src.SetDeadline(dl)
		}
	}

	bufferPool.Put(buf)
	done <- true
}
