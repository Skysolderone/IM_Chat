package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"wsim/gateway/model"

	"github.com/cloudwego/netpoll"
)

func main() {
	eventLoop, _ := netpoll.NewEventLoop(
		onRequest,
		netpoll.WithOnPrepare(onPrepare),
		netpoll.WithOnConnect(onConnect),
	)
	// connManager := netpoll.NewConnectionManager()
	model.NewUsers()
	// 目前不需要多网关机制
	// model.InitSend()
	// 修改为监听所有接口，支持外部连接
	listener, err := netpoll.CreateListener("tcp4", "0.0.0.0:8085")
	if err != nil {
		log.Fatalf("创建监听器失败: %v", err)
	}
	log.Printf("服务器启动，监听地址: 0.0.0.0:8085 (IPv4)")
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
	reader := conn.Reader()
	if reader.Len() == 0 {
		return nil
	}
	if reader.Len() < 21 {
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

	case model.MessageTypeText:
		fmt.Println("收到文本消息: ", msg)
		data := fmt.Sprintf("%d 发送了消息: %s", msg.FromUserID, string(msg.Data))

		// 如果消息是发给其他用户的，则需要转发给其他用户
		if msg.ToUserID != 0 {
			if receiver, ok := model.Users[msg.ToUserID]; ok {
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
				// 如果接收者不存在，则需要转发给gateway
				fmt.Println("receiver not found, forwarding to gateway")
				model.SendMessage(msg)
				// fmt.Println("receiver not found")
				// conn.Writer().WriteString("receiver not found")
				// conn.Writer().Flush()
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
	fmt.Println("onConnect")
	auth := &model.Auth{
		IsAuth:     false,
		RemoteAddr: ctx.Value("remoteAddr").(string),
	}
	return context.WithValue(ctx, "auth", auth)
}
