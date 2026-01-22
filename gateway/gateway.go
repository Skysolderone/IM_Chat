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

// 网关服务
func main() {
	if err := config.LoadConfig(""); err != nil {
		log.Fatalf("加载配置失败: %v", err)
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
	// 目前不需要多网关机制
	model.InitSend()
	// 修改为监听所有接口，支持外部连接
	listener, err := netpoll.CreateListener("tcp4", "0.0.0.0:"+port)
	if err != nil {
		log.Fatalf("创建监听器失败: %v", err)
	}
	log.Printf("服务器启动，监听地址: 0.0.0.0:%s (IPv4)", port)
	err = eventLoop.Serve(listener)
	if err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
	model.SendClose()
}

func onPrepare(conn netpoll.Connection) context.Context {
	// 这里做限流
	remoteAddr := conn.RemoteAddr().String()
	fmt.Println("remoteAddr: ", remoteAddr)
	ctx := context.WithValue(context.Background(), "remoteAddr", remoteAddr)
	return ctx
}

func onRequest(ctx context.Context, conn netpoll.Connection) error {
	fmt.Printf("onRequest被调用，RemoteAddr: %s\n", conn.RemoteAddr().String())
	reader := conn.Reader()
	fmt.Printf("接收到数据，长度: %d\n", reader.Len())
	if reader.Len() == 0 {
		fmt.Println("数据长度为0")
		return nil
	}
	if reader.Len() < 21 {
		fmt.Printf("数据长度小于21，实际长度: %d\n", reader.Len())
		return nil
	}
	// 读取 DataLen
	header, _ := reader.Peek(model.HeaderLen)
	dataLen := binary.BigEndian.Uint32(header[17:21])
	totalLen := model.HeaderLen + int(dataLen)
	fmt.Printf("解析头部：数据长度=%d, 总长度=%d, 当前可读=%d\n", dataLen, totalLen, reader.Len())

	// 数据不完整
	if reader.Len() < totalLen {
		fmt.Printf("数据不完整：需要%d字节，实际%d字节\n", totalLen, reader.Len())
		return nil
	}
	auth := ctx.Value("auth").(*model.Auth)
	data, err := reader.Next(totalLen)
	if err != nil {
		fmt.Println("onRequest error: ", err)
		conn.Close()
		return err
	}
	msg := model.Decode(data)

	switch msg.Type {
	case model.MessageTypeAuth:
		if !auth.IsAuth {
			// 读取首包
			fmt.Printf("收到登陆请求: %+v\n", msg)
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
		fmt.Println("收到文本消息: ", msg)

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
				n, err := receiver.Conn.Writer().WriteString(string(data))
				if err != nil {
					fmt.Println("write string error: ", err)
					conn.Writer().WriteString("write string error")
					conn.Writer().Flush()
					return err
				}
				fmt.Println("write string success: ", n)
				receiver.Conn.Writer().Flush()
			} else {
				// 如果接收者不在当前网关，转发给其他网关
				fmt.Println("receiver not found in current gateway, forwarding to other gateways")
				model.SendMessage(msg)
				return nil
			}
		}
		return nil
	case model.MessageTypeImage:
		fmt.Printf("收到: %s\n", string(msg.Data))
		return nil
	case model.MessageTypeVoice:
		fmt.Printf("收到: %s\n", string(msg.Data))
		return nil
	case model.MessageTypeVideo:
		fmt.Printf("收到: %s\n", string(msg.Data))
		return nil
	}

	fmt.Printf("收到: %s\n", string(data))

	// 回复客户端
	conn.Writer().WriteString("OK\n")
	conn.Writer().Flush()
	return nil
}

func onConnect(ctx context.Context, conn netpoll.Connection) context.Context {
	auth := &model.Auth{
		IsAuth:     false,
		RemoteAddr: ctx.Value("remoteAddr").(string),
	}
	fmt.Println("onConnect: ", auth)
	return context.WithValue(ctx, "auth", auth)
}

// handleGatewayForwardedMessage 处理来自其他网关转发的消息
func handleGatewayForwardedMessage(msg model.Message, receiver *model.User) {
	fmt.Printf("处理来自其他网关的消息: %+v\n", msg)

	switch msg.Type {
	case model.MessageTypeText:
		responseData := fmt.Sprintf("%d 发送了消息: %s", msg.FromUserID, string(msg.Data))
		_, err := receiver.Conn.Writer().WriteString(responseData)
		if err != nil {
			fmt.Printf("转发消息给用户 %d 失败: %v\n", msg.ToUserID, err)
			return
		}
		receiver.Conn.Writer().Flush()
		fmt.Printf("消息已转发给用户 %d\n", msg.ToUserID)
	case model.MessageTypeImage, model.MessageTypeVoice, model.MessageTypeVideo:
		// 转发二进制消息
		_, err := receiver.Conn.Writer().WriteBinary(msg.Data)
		if err != nil {
			fmt.Printf("转发消息给用户 %d 失败: %v\n", msg.ToUserID, err)
			return
		}
		receiver.Conn.Writer().Flush()
		fmt.Printf("消息已转发给用户 %d\n", msg.ToUserID)
	}
}
