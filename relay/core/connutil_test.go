package core

import (
	"bufio"
	"net"
	"testing"
)

func TestBufferedConnImplementsNetConn(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	bc := &BufferedConn{Conn: c1, Reader: bufio.NewReader(c1)}
	var rw interface{} = bc
	if _, ok := rw.(net.Conn); !ok {
		t.Fatal("BufferedConn should satisfy net.Conn when passed as interface")
	}
}
