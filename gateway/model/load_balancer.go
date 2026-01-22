package model

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/netpoll"
)

var (
	// 当前网关的连接数
	currentConnCount int
	connCountMutex   sync.RWMutex
	// 网关列表（用于负载均衡）
	gatewayListForLB []string
	// 每个网关的最后连接时间
	gatewayLastConnTime map[string]time.Time
	gatewayTimeMutex    sync.RWMutex
)

// InitLoadBalancer 初始化负载均衡器
func InitLoadBalancer() {
	// 获取网关列表用于负载均衡
	gatewayListStr := os.Getenv("GATEWAY_LIST")
	if gatewayListStr != "" {
		gatewayListForLB = strings.Split(gatewayListStr, ",")
		for i, gw := range gatewayListForLB {
			gatewayListForLB[i] = strings.TrimSpace(gw)
		}
	}

	// 初始化网关最后连接时间
	gatewayLastConnTime = make(map[string]time.Time)
	for _, gw := range gatewayListForLB {
		gatewayLastConnTime[gw] = time.Time{} // 初始化为零值，表示从未连接
	}
}

// IncrementConnCount 增加连接数
func IncrementConnCount() {
	connCountMutex.Lock()
	currentConnCount++
	connCountMutex.Unlock()
}

// DecrementConnCount 减少连接数
func DecrementConnCount() {
	connCountMutex.Lock()
	if currentConnCount > 0 {
		currentConnCount--
	}
	connCountMutex.Unlock()
}

// GetConnCount 获取当前连接数
func GetConnCount() int {
	connCountMutex.RLock()
	defer connCountMutex.RUnlock()
	return currentConnCount
}

// ShouldForwardConnection 判断是否应该转发连接（总是返回true，因为我们要根据时间选择）
func ShouldForwardConnection() bool {
	// 如果有其他网关可用，就转发
	return len(gatewayListForLB) > 0
}

// SelectBestGateway 选择最久没有连接的网关
func SelectBestGateway() string {
	gatewayTimeMutex.Lock()
	defer gatewayTimeMutex.Unlock()

	if len(gatewayListForLB) == 0 {
		return ""
	}

	// 找到最久没有连接的网关
	var selectedGateway string
	var oldestTime time.Time
	first := true

	for _, gw := range gatewayListForLB {
		lastTime := gatewayLastConnTime[gw]
		if first || lastTime.Before(oldestTime) || lastTime.IsZero() {
			oldestTime = lastTime
			selectedGateway = gw
			first = false
		}
	}

	// 更新选中网关的最后连接时间
	if selectedGateway != "" {
		gatewayLastConnTime[selectedGateway] = time.Now()
		if Logger != nil {
			Logger.Printf("选择网关: %s (最久未连接: %v)", selectedGateway, oldestTime)
		}
	}

	return selectedGateway
}

// ForwardConnectionToGateway 将连接转发到其他网关
func ForwardConnectionToGateway(ctx context.Context, clientConn netpoll.Connection, targetGateway string) error {
	if Logger != nil {
		Logger.Printf("转发连接到网关: %s, 当前连接数: %d", targetGateway, GetConnCount())
	}

	// 连接到目标网关
	backendConn, err := netpoll.NewDialer().DialConnection("tcp", targetGateway, time.Second*10)
	if err != nil {
		if Logger != nil {
			Logger.Printf("连接目标网关 %s 失败: %v", targetGateway, err)
		}
		return err
	}
	defer backendConn.Close()

	// 双向转发数据
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 客户端 -> 后端网关
	go func() {
		reader := clientConn.Reader()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if reader.Len() > 0 {
					data, err := reader.Next(reader.Len())
					if err != nil {
						cancel()
						return
					}
					_, err = backendConn.Writer().WriteBinary(data)
					if err != nil {
						cancel()
						return
					}
					backendConn.Writer().Flush()
				}
				time.Sleep(time.Millisecond * 10)
			}
		}
	}()

	// 后端网关 -> 客户端
	backendReader := backendConn.Reader()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if backendReader.Len() > 0 {
				data, err := backendReader.Next(backendReader.Len())
				if err != nil {
					cancel()
					return err
				}
				_, err = clientConn.Writer().WriteBinary(data)
				if err != nil {
					cancel()
					return err
				}
				clientConn.Writer().Flush()
			}
			time.Sleep(time.Millisecond * 10)
		}
	}
}
