package main

import (
	"context"
	"log"

	"github.com/cloudwego/netpoll"
)

func main() {
	eventLoop, err := netpoll.NewEventLoop(
		onRequest, // 注意：这里是 OnRequest（数据可读时触发），不是 OnConnect
		netpoll.WithOnPrepare(onPrepare),
		netpoll.WithOnConnect(onConnect),
	)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := netpoll.CreateListener("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	if err := eventLoop.Serve(listener); err != nil {
		log.Fatal(err)
	}
}

// 1. 最先触发 - 预处理
func onPrepare(conn netpoll.Connection) context.Context {
	log.Printf("[Prepare] 新连接准备: %s", conn.RemoteAddr())
	return context.Background()
}

// 2. 连接建立完成
func onConnect(ctx context.Context, conn netpoll.Connection) context.Context {
	log.Printf("[Connect] 连接建立: %s", conn.RemoteAddr())

	// 设置关闭回调
	if err := conn.AddCloseCallback(onClose); err != nil {
		log.Printf("[CloseCallback] 注册失败: %v", err)
	}

	// 发送欢迎语（写/Flush 失败通常说明连接已不可用，直接忽略即可）
	w := conn.Writer()
	if _, err := w.WriteString("欢迎连接\n"); err == nil {
		_ = w.Flush()
	}

	return ctx
}

// 3. 收到数据时触发
func onRequest(ctx context.Context, conn netpoll.Connection) error {
	// ✅ 关键：必须把 input buffer 读空（或主动 Close），否则会陷入死循环
	for {
		r := conn.Reader()
		n := r.Len()
		if n <= 0 {
			return nil
		}
		data, err := r.Next(n) // 读取并消费
		if err != nil {
			return nil
		}
		log.Printf("收到: %s", string(data))
		_ = r.Release()
	}
}

// 4. 连接关闭时触发
func onClose(conn netpoll.Connection) error {
	log.Printf("[Close] 连接关闭: %s", conn.RemoteAddr())
	return nil
}
