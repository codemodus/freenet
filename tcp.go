package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/codemodus/freenet/coms"
)

const (
	sockOptOrigDst = 80
)

func listenTCP(cs *coms.Coms, port int, secure bool) {
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
		c, err := acceptTCP(l)
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

		cs.Conc(func() {
			if err := bond(cs, done, c); err != nil {
				cs.Errorf("bond broken: %s", err)
			}

			select {
			case <-done:
			default:
				close(done)
			}
		})
	}
}

func bond(cs *coms.Coms, done chan struct{}, conn *net.TCPConn) error {
	f, err := conn.File()
	if err != nil {
		return fmt.Errorf("cannot get conn file: %s", err)
	}
	defer qtClose(f)

	a, err := address(f)
	if err != nil {
		return fmt.Errorf("cannot determine source address: %s", err)
	}

	cl, err := dial(a)
	if err != nil {
		return fmt.Errorf("cannot dial destination (%s): %s", a, err)
	}
	defer qtClose(cl)
	cs.Infof("connected to remote at %s", a)

	ec := make(chan error)
	defer close(ec)

	go func() {
		_, _ = io.Copy(cl, conn)
		ec <- nil
	}()

	_, err = intercept(conn, cl, isSecure(a))
	if err != nil {
		<-ec
		return err
	}

	return <-ec
}

func acceptTCP(l *net.TCPListener) (*net.TCPConn, error) {
	c, err := l.AcceptTCP()
	if err != nil {
		return nil, err
	}

	if err := c.SetLinger(10); err != nil {
		return nil, err
	}

	return c, nil
}

func dial(address string) (net.Conn, error) {
	cl, err := net.DialTimeout("tcp", address, time.Second*16)
	if err != nil {
		return nil, err
	}

	if err = cl.SetReadDeadline(time.Now().Add(time.Second * 16)); err != nil {
		return nil, err
	}

	return cl, nil
}

func qtClose(c io.Closer) {
	_ = c.Close()
}

func address(f *os.File) (string, error) {
	fd := int(f.Fd())

	a, err := syscall.GetsockoptIPv6Mreq(fd, syscall.SOL_IP, sockOptOrigDst)
	if err != nil {
		return "", err
	}

	return multiaddrToString(a.Multiaddr), nil
}

func multiaddrToString(multiaddr [16]byte) string {
	ip := multiaddr[4:8]
	ipStr := net.IPv4(ip[0], ip[1], ip[2], ip[3]).String()

	port := multiaddr[2:4]
	portUint := int64((uint32(port[0]) << 8) + uint32(port[1]))
	portStr := strconv.FormatInt(portUint, 10)

	return (ipStr + ":" + portStr)
}

func isSecure(s string) bool {
	return splitPort(s) == "443"
}

func splitPort(s string) string {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == ':' {
			return s[i+1:]
		}
	}

	return "0"
}
