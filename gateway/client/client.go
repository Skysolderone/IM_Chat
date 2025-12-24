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

	"github.com/cloudwego/hertz/pkg/network/netpoll"
)

func main() {
	conn, err := netpoll.NewDialer().DialConnection("tcp", "127.0.0.1:8085", time.Second*10, nil)
	// conn, err := netpoll.NewDialer().DialConnection("tcp", "52.201.237.21:8085", time.Second*10, nil)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	var msg model.Message
	msg.FromUserID = uint64(math.Round(rand.Float64() * 1000000))
	msg.FromUserID = 1
	msg.Type = model.MessageTypeAuth
	data := model.Encode(msg)
	fmt.Printf("发送认证消息，数据长度: %d, 内容: %+v\n", len(data), msg)
	// n, err := conn.Write(data)
	n, err := conn.WriteBinary(data)
	fmt.Printf("发送认证消息，数据长度: %d, 内容: %+v, 发送长度: %d\n", len(data), msg, n)
	if err != nil {
		fmt.Println("Failed to send message: ", err)
		return
	}
	conn.Flush()
	fmt.Println("认证消息发送成功")

	// 启动 goroutine 接收服务器消息
	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Println("Failed to read from server: ", err)
				return
			}
			fmt.Println(string(buf[:n]))
		}
	}()

	// 主循环：接收终端输入并发送
	fmt.Println("请输入消息（直接按回车发送）:")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		if input == "" {
			continue
		}
		msg.FromUserID = 1
		msg.ToUserID = 2
		msg.Type = model.MessageTypeText
		msg.Data = []byte(input)
		data := model.Encode(msg)
		fmt.Printf("发送文本消息，长度: %d, 内容: %s\n", len(data), input)
		// _, err = conn.Write(data)
		_, err = conn.WriteBinary(data)
		if err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}
		conn.Flush()
		fmt.Println("消息发送成功")
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read from stdin: %v", err)
	}
}
