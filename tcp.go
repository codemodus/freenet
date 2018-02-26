package main

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"syscall"

	"github.com/codemodus/freenet/coms"
)

const (
	sockOptOrigDst = 80
)

func listen(cs *coms.Coms, port int, secure bool) {
	a := &net.TCPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: port,
	}

	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		cs.Errorf("cannot setup listener: %s", err)
		return
	}
	defer qtClose(l)
	cs.Infof("listener opened (:%d)", port)

	go func() {
		<-cs.Done()
		qtClose(l)
	}()

	spawn(cs, l)

	cs.Infof("listener closed (:%d)", port)
}

func spawn(cs *coms.Coms, l *net.TCPListener) {
	for {
		c, err := l.AcceptTCP()
		if err != nil {
			select {
			case <-cs.Done():
				return
			default:
				cs.Errorf("cannot accept connection: %s", err)
			}
		}
		cs.Infof("accepted connection on %s", c.RemoteAddr().String())

		done := make(chan struct{})
		go func() {
			select {
			case <-cs.Done():
			case <-done:
			}

			qtClose(c)
		}()

		if err := c.SetLinger(10); err != nil {
			cs.Errorf("cannot set conn linger: %s", err)
			close(done)
			continue
		}

		cs.Conc(func() {
			intercept(cs, c)
			close(done)
		})
	}
}

var (
	gup, gdn = 0, 0
	cup, cdn = 0, 0
)

func intercept(cs *coms.Coms, conn *net.TCPConn) {
	f, err := conn.File()
	if err != nil {
		cs.Errorf("cannot get conn file: %s", err)
		return
	}
	defer qtClose(f)

	fd := int(f.Fd())
	/*c, err := net.FileConn(f)
	if err != nil {
		cs.Errorf("cannot get file conn: %s", err)
		return
	}
	defer qtClose(c)*/

	/*	cDone := make(chan struct{})
		go func() {
			select {
			case <-cs.Done():
			case <-cDone:
			}
			qtClose(c)
		}()
		defer func() { close(cDone) }()*/

	a, err := syscall.GetsockoptIPv6Mreq(fd, syscall.SOL_IP, sockOptOrigDst)
	if err != nil {
		cs.Errorf("cannot determine source address: %s", err)
		return
	}
	as := multiaddrString(a.Multiaddr)

	cl, err := net.Dial("tcp", as)
	if err != nil {
		cs.Errorf("cannot dial destination (%s): %s", as, err)
	}
	defer qtClose(cl)
	cs.Infof("connected to remote at %s\n", as)

	done := make(chan struct{})
	go func() {
		select {
		case <-cs.Done():
		case <-done:
		}
		qtClose(cl)
	}()
	defer func() { close(done) }()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		cup++
		fmt.Println("cu", cup)
		_, _ = io.Copy(cl, conn)
		qtClose(cl)
		wg.Done()
		cdn++
		fmt.Println("cd", cdn)
	}()

	go func() {
		gup++
		fmt.Println("gu", gup)
		_, _ = io.Copy(conn, cl)
		qtClose(conn)
		wg.Done()
		gdn++
		fmt.Println("gd", gdn)
	}()

	wg.Wait()
}

func qtClose(c io.Closer) {
	_ = c.Close()
}

func multiaddrString(multiaddr [16]byte) string {
	ip := multiaddr[4:8]
	ipStr := net.IPv4(ip[0], ip[1], ip[2], ip[3]).String()

	port := multiaddr[2:4]
	portUint := int64((uint32(port[0]) << 8) + uint32(port[1]))
	portStr := strconv.FormatInt(portUint, 10)

	return (ipStr + ":" + portStr)
}
