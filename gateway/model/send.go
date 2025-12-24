package model

import (
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/netpoll"
)

var gateWayMap map[string]netpoll.Connection

var gateWayList = []string{"52.201.237.21:8085"}

func InitSend() {
	gateWayMap = make(map[string]netpoll.Connection)
	// 初始化跟gateway的连接
	fmt.Println("InitSend: ", gateWayList)
	for _, gateway := range gateWayList {
		conn, err := netpoll.NewDialer().DialConnection("tcp", gateway, time.Second*10)
		if err != nil {
			fmt.Printf("Failed to connect to gateway: %v", err)
		}
		gateWayMap[gateway] = conn
	}
	fmt.Println("InitSend success: ", gateWayMap)
}

func SendMessage(msg Message) {
	fmt.Println("SendMessage: ", msg)
	// 需要先从全局路由表拿到该用户在那个网关 这里使用redis存储
	// 目前测试 写死为52.201.237.21:8095
	gateway := "52.201.237.21:8085"
	conn, ok := gateWayMap[gateway]
	if !ok {
		fmt.Printf("Failed to get gateway: %s", gateway)
	}
	data, err := sonic.Marshal(msg)
	if err != nil {
		fmt.Printf("Failed to marshal message: %v", err)
	}
	conn.Write(data)
	conn.Writer().Flush()
}

func SendClose() {
	for _, conn := range gateWayMap {
		conn.Close()
	}
}
