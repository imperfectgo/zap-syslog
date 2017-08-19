// Copyright (c) 2017 Timon Wong
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package zapsyslog

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

const (
	testMessage = "<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - " +
		"\xef\xbb\xbf'su root' failed for lonvick on /dev/pts/8"
)

var (
	crashy = false
)

func runPktSyslog(c net.PacketConn, done chan<- string) {
	var buf [4096]byte
	var rcvd string
	ct := 0
	for {
		var n int
		var err error

		c.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, _, err = c.ReadFrom(buf[:])
		rcvd += string(buf[:n])
		if err != nil {
			if oe, ok := err.(*net.OpError); ok {
				if ct < 3 && oe.Temporary() {
					ct++
					continue
				}
			}
			break
		}
	}
	c.Close()
	done <- rcvd
}

func runStreamSyslog(l net.Listener, done chan<- string, wg *sync.WaitGroup) {
	for {
		var c net.Conn
		var err error
		if c, err = l.Accept(); err != nil {
			return
		}
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			c.SetReadDeadline(time.Now().Add(5 * time.Second))
			b := bufio.NewReader(c)
			for ct := 1; !crashy || ct&7 != 0; ct++ {
				s, err := b.ReadString('\n')
				if err != nil {
					break
				}
				done <- s
			}
			c.Close()
		}(c)
	}
}

func startServer(n, la string, done chan<- string) (addr string, sock io.Closer, wg *sync.WaitGroup) {
	if n == "udp" || n == "tcp" {
		la = "127.0.0.1:0"
	} else {
		// unix and unixgram: choose an address if none given
		if la == "" {
			// use ioutil.TempFile to get a name that is unique
			f, err := ioutil.TempFile("", "syslogtest")
			if err != nil {
				log.Fatal("TempFile: ", err)
			}
			f.Close()
			la = f.Name()
		}
		os.Remove(la)
	}

	wg = new(sync.WaitGroup)
	if n == "udp" || n == "unixgram" {
		l, e := net.ListenPacket(n, la)
		if e != nil {
			log.Fatalf("startServer failed: %v", e)
		}
		addr = l.LocalAddr().String()
		sock = l
		wg.Add(1)
		go func() {
			defer wg.Done()
			runPktSyslog(l, done)
		}()
	} else {
		l, e := net.Listen(n, la)
		if e != nil {
			log.Fatalf("startServer failed: %v", e)
		}
		addr = l.Addr().String()
		sock = l
		wg.Add(1)
		go func() {
			defer wg.Done()
			runStreamSyslog(l, done, wg)
		}()
	}
	return
}

func TestWrite(t *testing.T) {
	t.Parallel()
	testMessages := []string{
		testMessage,
		"<165>1 2003-10-11T22:14:15.003Z localhost evntslog - ID47 - \xef\xbb\xbfAn application event log entry...",
	}

	for _, msg := range testMessages {
		done := make(chan string)
		addr, sock, srvWG := startServer("udp", "", done)
		defer srvWG.Wait()
		defer sock.Close()
		l, err := NewConnSyncer("udp", addr)
		if err != nil {
			t.Fatalf("syslog.Dial() failed: %v", err)
		}
		_, err = io.WriteString(l, msg)
		if err != nil {
			t.Fatalf("WriteString() failed: %v", err)
		}
		rcvd := <-done
		if rcvd != msg {
			t.Errorf("message didn't match: expected=%q, actual=%q", msg, rcvd)
		}
	}
}

func TestConcurrentWrite(t *testing.T) {
	addr, sock, srvWG := startServer("udp", "", make(chan string, 1))
	defer srvWG.Wait()
	defer sock.Close()
	s, err := NewConnSyncer("udp", addr)
	if err != nil {
		t.Fatalf("zapsyslog.NewConnSyncer() failed: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := io.WriteString(s, testMessage)
			if err != nil {
				t.Errorf("WriteString() failed: %v", err)
				return
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentReconnect(t *testing.T) {
	crashy = true
	defer func() { crashy = false }()

	const N = 10
	const M = 100
	done := make(chan string, N*M)
	addr, sock, srvWG := startServer("tcp", "", done)

	// count all the messages arriving
	count := make(chan int)
	go func() {
		ct := 0
		for range done {
			ct++
			// we are looking for 500 out of 1000 events
			// here because lots of log messages are lost
			// in buffers (kernel and/or bufio)
			if ct > N*M/2 {
				break
			}
		}
		count <- ct
	}()

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			s, err := NewConnSyncer("tcp", addr)
			if err != nil {
				t.Errorf("zapsyslog.NewConnSyncer() failed: %v", err)
				return
			}
			for i := 0; i < M; i++ {
				_, err := io.WriteString(s, testMessage)
				if err != nil {
					t.Errorf("WriteString() failed: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	sock.Close()
	srvWG.Wait()
	close(done)

	select {
	case <-count:
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout in concurrent reconnect")
	}
}
