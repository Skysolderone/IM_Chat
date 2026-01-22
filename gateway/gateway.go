package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"time"

	"wsim/gateway/model"
	"wsim/pkg/config"

	"github.com/cloudwego/netpoll"
)

// 网关服务集群
func main() {
	// 获取网关ID，用于日志文件命名
	gatewayID := os.Getenv("GATEWAY_ID")
	if gatewayID == "" {
		gatewayID = "gateway1"
	}

	// 初始化日志系统
	if err := model.InitLogger(gatewayID); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer model.CloseLogger()

	if err := config.LoadConfig(""); err != nil {
		model.Logger.Fatalf("加载配置失败: %v", err)
	}

	eventLoop, _ := netpoll.NewEventLoop(
		onRequest,
		netpoll.WithOnPrepare(onPrepare),
		netpoll.WithOnConnect(onConnect),
		netpoll.WithReadTimeout(time.Second*30),
	)
	port := os.Getenv("PORT")
	if port == "" {
		port = config.GetServerPort()
	}
	// connManager := netpoll.NewConnectionManager()
	model.NewUsers()
	model.InitSend()
	model.InitLoadBalancer()
	// 修改为监听所有接口，支持外部连接
	listener, err := netpoll.CreateListener("tcp4", "0.0.0.0:"+port)
	if err != nil {
		model.Logger.Fatalf("创建监听器失败: %v", err)
	}
	model.Logger.Printf("服务器启动，监听地址: 0.0.0.0:%s (IPv4), 网关ID: %s", port, gatewayID)
	err = eventLoop.Serve(listener)
	if err != nil {
		model.Logger.Fatalf("服务器启动失败: %v", err)
	}
	model.SendClose()
}

func onPrepare(conn netpoll.Connection) context.Context {
	// 这里做负载均衡：选择最久没有连接的网关
	remoteAddr := conn.RemoteAddr().String()
	model.Logger.Printf("新连接: %s, 当前连接数: %d", remoteAddr, model.GetConnCount())
	
	// 检查是否有其他网关可用，如果有就转发到最久没有连接的网关
	if model.ShouldForwardConnection() {
		targetGateway := model.SelectBestGateway()
		if targetGateway != "" {
			model.Logger.Printf("转发连接到最久未连接的网关: %s", targetGateway)
			// 异步转发连接
			go func() {
				ctx := context.WithValue(context.Background(), "remoteAddr", remoteAddr)
				if err := model.ForwardConnectionToGateway(ctx, conn, targetGateway); err != nil {
					model.Logger.Printf("转发连接失败: %v", err)
					conn.Close()
				}
			}()
			// 返回一个特殊的 context，标记为转发连接
			ctx := context.WithValue(context.Background(), "forwarded", true)
			return ctx
		}
	}
	
	// 没有其他网关可用，自己处理连接
	model.IncrementConnCount()
	ctx := context.WithValue(context.Background(), "remoteAddr", remoteAddr)
	return ctx
}

func onRequest(ctx context.Context, conn netpoll.Connection) error {
	// 检查是否是转发连接，如果是则不需要处理
	if forwarded, ok := ctx.Value("forwarded").(bool); ok && forwarded {
		return nil
	}
	
	reader := conn.Reader()
	if reader.Len() == 0 {
		return nil
	}
	if reader.Len() < 21 {
		model.Logger.Printf("数据长度不足，实际长度: %d", reader.Len())
		return nil
	}
	// 读取 DataLen
	header, _ := reader.Peek(model.HeaderLen)
	dataLen := binary.BigEndian.Uint32(header[17:21])
	totalLen := model.HeaderLen + int(dataLen)

	// 数据不完整
	if reader.Len() < totalLen {
		return nil
	}
	auth := ctx.Value("auth").(*model.Auth)
	data, err := reader.Next(totalLen)
	if err != nil {
		model.Logger.Printf("读取数据错误: %v", err)
		conn.Close()
		return err
	}
	msg := model.Decode(data)

	switch msg.Type {
	case model.MessageTypeAuth:
		if !auth.IsAuth {
			// 读取首包
			model.Logger.Printf("收到登陆请求: FromUserID=%d", msg.FromUserID)
			data := fmt.Sprintf("%d 登陆成功", msg.FromUserID)

			// 回复客户端
			conn.Writer().WriteString(data)
			conn.Writer().Flush()
			auth.IsAuth = true
			auth.UserID = msg.FromUserID
			ctx = context.WithValue(ctx, "auth", auth)

		}
		// 保存该用户登陆状态
		// 首先判断用户是否存在
		if model.Users[msg.FromUserID] == nil {
			// 不存在需要创建用户
			model.Users[msg.FromUserID] = &model.User{
				UserID: msg.FromUserID,
				Conn:   conn,
				IsAuth: true,
			}
			// 设置用户路由：用户ID -> 当前网关ID
			model.SetUserRoute(msg.FromUserID, model.GetCurrentGatewayID())
			conn.Writer().WriteString("用户登陆成功")
			conn.Writer().Flush()
			return nil
		}
		// 首先判断是不是已经登陆
		if model.Users[msg.FromUserID].IsAuth {
			conn.Writer().WriteString("用户已登陆")
			conn.Writer().Flush()
			return nil
		}

		model.Users[msg.FromUserID] = &model.User{
			UserID: msg.FromUserID,
			Conn:   conn,
			IsAuth: true,
		}
		// 设置用户路由
		model.SetUserRoute(msg.FromUserID, model.GetCurrentGatewayID())

	case model.MessageTypeText:
		model.Logger.Printf("收到文本消息: FromUserID=%d, ToUserID=%d, DataLen=%d", msg.FromUserID, msg.ToUserID, len(msg.Data))

		// 如果消息是发给其他用户的，则需要转发给其他用户
		if msg.ToUserID != 0 {
			// 检查是否是来自其他网关的消息（发送者不在当前网关）
			fromUserInCurrentGateway := model.Users[msg.FromUserID] != nil && model.Users[msg.FromUserID].IsAuth

			if !fromUserInCurrentGateway {
				// 这是来自其他网关的消息，检查目标用户是否在当前网关
				if receiver, ok := model.Users[msg.ToUserID]; ok && receiver.IsAuth {
					// 目标用户在当前网关，转发给用户
					handleGatewayForwardedMessage(msg, receiver)
					return nil
				}
				// 目标用户也不在当前网关，忽略（可能在其他网关）
				return nil
			}

			// 这是本地用户发送的消息
			data := fmt.Sprintf("%d 发送了消息: %s", msg.FromUserID, string(msg.Data))
			if receiver, ok := model.Users[msg.ToUserID]; ok && receiver.IsAuth {
				// 用户在当前网关，直接转发
				_, err := receiver.Conn.Writer().WriteString(string(data))
				if err != nil {
					model.Logger.Printf("转发消息失败: %v", err)
					conn.Writer().WriteString("write string error")
					conn.Writer().Flush()
					return err
				}
				receiver.Conn.Writer().Flush()
				model.Logger.Printf("消息已转发给用户 %d", msg.ToUserID)
			} else {
				// 如果接收者不在当前网关，转发给其他网关
				model.Logger.Printf("用户 %d 不在当前网关，转发给其他网关", msg.ToUserID)
				model.SendMessage(msg)
				return nil
			}
		}
		return nil
	case model.MessageTypeImage:
		model.Logger.Printf("收到图片消息: FromUserID=%d, ToUserID=%d, DataLen=%d", msg.FromUserID, msg.ToUserID, len(msg.Data))
		return nil
	case model.MessageTypeVoice:
		model.Logger.Printf("收到语音消息: FromUserID=%d, ToUserID=%d, DataLen=%d", msg.FromUserID, msg.ToUserID, len(msg.Data))
		return nil
	case model.MessageTypeVideo:
		model.Logger.Printf("收到视频消息: FromUserID=%d, ToUserID=%d, DataLen=%d", msg.FromUserID, msg.ToUserID, len(msg.Data))
		return nil
	}

	model.Logger.Printf("收到未知类型消息: Type=%d", msg.Type)

	// 回复客户端
	conn.Writer().WriteString("OK\n")
	conn.Writer().Flush()
	return nil
}

func onConnect(ctx context.Context, conn netpoll.Connection) context.Context {
	// 检查是否是转发连接
	if forwarded, ok := ctx.Value("forwarded").(bool); ok && forwarded {
		// 这是转发连接，不需要处理
		return ctx
	}
	
	// 正常连接处理
	remoteAddr, _ := ctx.Value("remoteAddr").(string)
	auth := &model.Auth{
		IsAuth:     false,
		RemoteAddr: remoteAddr,
	}
	model.Logger.Printf("连接建立: RemoteAddr=%s", auth.RemoteAddr)
	return context.WithValue(ctx, "auth", auth)
}

// handleGatewayForwardedMessage 处理来自其他网关转发的消息
func handleGatewayForwardedMessage(msg model.Message, receiver *model.User) {
	model.Logger.Printf("处理来自其他网关的消息: FromUserID=%d, ToUserID=%d, Type=%d", msg.FromUserID, msg.ToUserID, msg.Type)

	switch msg.Type {
	case model.MessageTypeText:
		responseData := fmt.Sprintf("%d 发送了消息: %s", msg.FromUserID, string(msg.Data))
		_, err := receiver.Conn.Writer().WriteString(responseData)
		if err != nil {
			model.Logger.Printf("转发消息给用户 %d 失败: %v", msg.ToUserID, err)
			return
		}
		receiver.Conn.Writer().Flush()
		model.Logger.Printf("消息已转发给用户 %d", msg.ToUserID)
	case model.MessageTypeImage, model.MessageTypeVoice, model.MessageTypeVideo:
		// 转发二进制消息
		_, err := receiver.Conn.Writer().WriteBinary(msg.Data)
		if err != nil {
			model.Logger.Printf("转发消息给用户 %d 失败: %v", msg.ToUserID, err)
			return
		}
		receiver.Conn.Writer().Flush()
		model.Logger.Printf("消息已转发给用户 %d", msg.ToUserID)
	}
}
