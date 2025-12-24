package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/network/netpoll"
)

func TestSocket(t *testing.T) {
	conn, err := netpoll.NewDialer().DialConnection("tcp", "52.201.237.21:8085", time.Second*10, nil)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte("Hello, World!"))

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read from server: %v", err)
	}
	fmt.Println(string(buf[:n]))
}
