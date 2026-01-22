package model

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/netpoll"
)

var (
	gateWayMap      map[string]netpoll.Connection
	gateWayMapMutex sync.RWMutex
	// 用户路由表：用户ID -> 网关ID
	userRouteMap    map[uint64]string
	userRouteMutex  sync.RWMutex
	currentGatewayID string
)

// InitSend 初始化网关连接，从环境变量读取网关列表
func InitSend() {
	gateWayMap = make(map[string]netpoll.Connection)
	userRouteMap = make(map[uint64]string)
	
	// 从环境变量获取当前网关ID
	currentGatewayID = os.Getenv("GATEWAY_ID")
	if currentGatewayID == "" {
		currentGatewayID = "gateway1"
	}
	
	// 从环境变量获取网关列表
	gatewayListStr := os.Getenv("GATEWAY_LIST")
	var gateWayList []string
	
	if gatewayListStr != "" {
		gateWayList = strings.Split(gatewayListStr, ",")
		// 清理空格
		for i, gw := range gateWayList {
			gateWayList[i] = strings.TrimSpace(gw)
		}
	} else {
		// 默认值，用于本地开发
		gateWayList = []string{}
	}
	
	fmt.Printf("当前网关ID: %s\n", currentGatewayID)
	fmt.Printf("需要连接的网关列表: %v\n", gateWayList)
	
	// 初始化跟其他gateway的连接
	for _, gateway := range gateWayList {
		// 使用容器名称或IP:端口格式
		conn, err := netpoll.NewDialer().DialConnection("tcp", gateway, time.Second*10)
		if err != nil {
			fmt.Printf("连接网关 %s 失败: %v，将稍后重试\n", gateway, err)
			// 启动后台重连机制
			go reconnectGateway(gateway)
		} else {
			gateWayMapMutex.Lock()
			gateWayMap[gateway] = conn
			gateWayMapMutex.Unlock()
			fmt.Printf("成功连接到网关: %s\n", gateway)
		}
	}
	
	fmt.Printf("网关连接初始化完成，已连接: %d 个网关\n", len(gateWayMap))
}

// reconnectGateway 后台重连失败的网关
func reconnectGateway(gateway string) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	
	for range ticker.C {
		gateWayMapMutex.RLock()
		_, exists := gateWayMap[gateway]
		gateWayMapMutex.RUnlock()
		
		if exists {
			return // 已经连接成功，退出重连
		}
		
		conn, err := netpoll.NewDialer().DialConnection("tcp", gateway, time.Second*10)
		if err != nil {
			fmt.Printf("重连网关 %s 失败: %v\n", gateway, err)
			continue
		}
		
		gateWayMapMutex.Lock()
		gateWayMap[gateway] = conn
		gateWayMapMutex.Unlock()
		fmt.Printf("重连网关 %s 成功\n", gateway)
		return
	}
}

// SetUserRoute 设置用户路由（用户ID -> 网关ID）
func SetUserRoute(userID uint64, gatewayID string) {
	userRouteMutex.Lock()
	defer userRouteMutex.Unlock()
	userRouteMap[userID] = gatewayID
}

// GetUserRoute 获取用户所在网关
func GetUserRoute(userID uint64) (string, bool) {
	userRouteMutex.RLock()
	defer userRouteMutex.RUnlock()
	gatewayID, ok := userRouteMap[userID]
	return gatewayID, ok
}

// GetCurrentGatewayID 获取当前网关ID
func GetCurrentGatewayID() string {
	return currentGatewayID
}

// SendMessage 发送消息到其他网关
func SendMessage(msg Message) {
	fmt.Printf("SendMessage: %+v\n", msg)
	
	// 获取目标用户所在的网关
	targetGatewayID, ok := GetUserRoute(msg.ToUserID)
	if !ok {
		// 如果找不到路由，广播到所有网关
		fmt.Printf("用户 %d 路由不存在，广播到所有网关\n", msg.ToUserID)
		broadcastMessage(msg)
		return
	}
	
	// 如果目标用户在当前网关，不需要转发
	if targetGatewayID == currentGatewayID {
		fmt.Printf("用户 %d 在当前网关，无需转发\n", msg.ToUserID)
		return
	}
	
	// 找到目标网关的连接
	gateWayMapMutex.RLock()
	var targetConn netpoll.Connection
	for gatewayAddr, conn := range gateWayMap {
		// 如果地址包含目标网关ID（如 gateway2:8085 包含 gateway2）
		if strings.HasPrefix(gatewayAddr, targetGatewayID) || strings.Contains(gatewayAddr, targetGatewayID+":") {
			targetConn = conn
			break
		}
	}
	gateWayMapMutex.RUnlock()
	
	if targetConn == nil {
		fmt.Printf("未找到网关 %s 的连接，尝试广播\n", targetGatewayID)
		// 尝试广播
		broadcastMessage(msg)
		return
	}
	
	data := Encode(msg)
	_, err := targetConn.Writer().WriteBinary(data)
	if err != nil {
		fmt.Printf("发送消息到网关 %s 失败: %v\n", targetGatewayID, err)
		return
	}
	targetConn.Writer().Flush()
	fmt.Printf("消息已发送到网关 %s\n", targetGatewayID)
}

// broadcastMessage 广播消息到所有网关
func broadcastMessage(msg Message) {
	data := Encode(msg)
	gateWayMapMutex.RLock()
	defer gateWayMapMutex.RUnlock()
	
	for gatewayAddr, conn := range gateWayMap {
		_, err := conn.Writer().WriteBinary(data)
		if err != nil {
			fmt.Printf("广播消息到网关 %s 失败: %v\n", gatewayAddr, err)
			continue
		}
		conn.Writer().Flush()
		fmt.Printf("消息已广播到网关 %s\n", gatewayAddr)
	}
}

// SendClose 关闭所有网关连接
func SendClose() {
	gateWayMapMutex.Lock()
	defer gateWayMapMutex.Unlock()
	
	for gatewayAddr, conn := range gateWayMap {
		conn.Close()
		fmt.Printf("已关闭网关连接: %s\n", gatewayAddr)
	}
	gateWayMap = make(map[string]netpoll.Connection)
}
