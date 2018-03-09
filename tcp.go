package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

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
			bond(cs, done, c)

			select {
			case <-done:
			default:
				close(done)
			}
		})
	}
}

func bond(cs *coms.Coms, done chan struct{}, conn *net.TCPConn) {
	f, err := conn.File()
	if err != nil {
		cs.Errorf("cannot get conn file: %s", err)
		return
	}
	defer qtClose(f)

	a, err := address(f)
	if err != nil {
		cs.Errorf("cannot determine source address: %s", err)
		return
	}

	cl, err := net.DialTimeout("tcp", a, time.Second*16)
	if err != nil {
		cs.Errorf("cannot dial destination (%s): %s", a, err)
		close(done)
		return
	}
	defer qtClose(cl)
	cs.Infof("connected to remote at %s", a)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		if _, err := reqCopy(cl, conn); err != nil {
			fmt.Println("copy err:", err)
		}
	}()

	go func() {
		defer wg.Done()

		secure := splitPort(a) == "443"

		_ = cl.SetReadDeadline(time.Now().Add(time.Second * 16))
		if _, err := intercept(conn, cl, secure); err != nil {
			fmt.Println("intercept", err)
		}
	}()

	wg.Wait()
}

func reqCopy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
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

func splitPort(s string) string {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == ':' {
			return s[i+1:]
		}
	}

	return "443"
}
