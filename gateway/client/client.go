package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"time"

	"wsim/gateway/model"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/network/netpoll"
)

func main() {
	// conn, err := netpoll.NewDialer().DialConnection("tcp", "127.0.0.1:8085", time.Second*10, nil)
	conn, err := netpoll.NewDialer().DialConnection("tcp", "52.201.237.21:8085", time.Second*10, nil)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	var msg model.Message
	msg.FromUserID = int64(math.Round(rand.Float64() * 1000000))
	msg.FromUserID = 2
	msg.Type = model.MessageTypeAuth
	data, err := sonic.Marshal(msg)
	if err != nil {
		log.Fatalf("Failed to marshal message: %v", err)
	}
	conn.Write(data)

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("Failed to read from server: %v", err)
	}
	fmt.Println(string(buf[:n]))

	// 启动 goroutine 接收服务器消息
	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				log.Fatalf("Failed to read from server: %v", err)
			}
			fmt.Println(string(buf[:n]))
		}
	}()

	// 主循环：接收终端输入并发送
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		if input == "" {
			continue
		}
		msg.FromUserID = 2
		msg.ToUserID = 1
		msg.Type = model.MessageTypeText
		msg.Data = []byte(input)
		data, err := sonic.Marshal(msg)
		if err != nil {
			log.Fatalf("Failed to marshal message: %v", err)
		}
		_, err = conn.Write(data)
		if err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read from stdin: %v", err)
	}
}
