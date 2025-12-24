package model

import (
	"encoding/binary"
	"io"
	"sync"
)

type Message struct {
	FromUserID uint64      `json:"from_user_id"`
	ToUserID   uint64      `json:"to_user_id"`
	Type       MessageType `json:"type"`
	Data       []byte      `json:"data"`
}

type MessageType int

const (
	MessageTypeAuth  MessageType = 0
	MessageTypeText  MessageType = 1
	MessageTypeImage MessageType = 2
	MessageTypeVoice MessageType = 3
	MessageTypeVideo MessageType = 4
	MessageTypeFile  MessageType = 5
	MessageTypePing  MessageType = 6
	MessageTypePong  MessageType = 7
)

func (m MessageType) Int() int {
	return int(m)
}

var HeaderLen = 21 // 8+8+1+4 = 21字节
func Decode(data []byte) Message {
	if len(data) < HeaderLen {
		return Message{}
	}
	fromUserID := binary.BigEndian.Uint64(data[:8])
	toUserID := binary.BigEndian.Uint64(data[8:16])
	msgType := MessageType(data[16])
	dataLen := binary.BigEndian.Uint32(data[17:21])
	if int(dataLen) > len(data)-21 {
		return Message{}
	}
	return Message{
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		Type:       msgType,
		Data:       data[21 : 21+dataLen],
	}
}

func Encode(msg Message) []byte {
	datalen := len(msg.Data)
	buf := make([]byte, HeaderLen+datalen)
	binary.BigEndian.PutUint64(buf[:8], msg.FromUserID)
	binary.BigEndian.PutUint64(buf[8:16], msg.ToUserID)
	buf[16] = uint8(msg.Type)
	binary.BigEndian.PutUint32(buf[17:21], uint32(datalen))
	copy(buf[21:], msg.Data)
	return buf
}

// 零拷贝编码器，使用缓冲区池重用内存
var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 1024) // 初始大小1KB
	},
}

// EncodeZeroCopy 零拷贝编码到Writer，避免内存分配和拷贝
func EncodeZeroCopy(writer io.Writer, msg Message) error {
	// 创建头部缓冲区（固定21字节）
	header := make([]byte, 21)
	binary.BigEndian.PutUint64(header[:8], msg.FromUserID)
	binary.BigEndian.PutUint64(header[8:16], msg.ToUserID)
	header[16] = uint8(msg.Type)
	binary.BigEndian.PutUint32(header[17:21], uint32(len(msg.Data)))

	// 直接写入头部，无需拷贝
	if _, err := writer.Write(header); err != nil {
		return err
	}

	// 直接写入数据，无需拷贝
	if len(msg.Data) > 0 {
		_, err := writer.Write(msg.Data)
		return err
	}

	return nil
}

// EncodeWithPool 使用缓冲区池的编码方法，减少内存分配
func EncodeWithPool(msg Message) []byte {
	datalen := len(msg.Data)
	totalLen := HeaderLen + datalen

	// 从池中获取缓冲区
	poolBuf := bufferPool.Get().([]byte)
	defer bufferPool.Put(poolBuf)

	// 如果池中的缓冲区太小，扩展它
	var buf []byte
	if len(poolBuf) < totalLen {
		buf = make([]byte, totalLen)
	} else {
		buf = poolBuf[:totalLen]
	}

	// 编码头部
	binary.BigEndian.PutUint64(buf[:8], msg.FromUserID)
	binary.BigEndian.PutUint64(buf[8:16], msg.ToUserID)
	buf[16] = uint8(msg.Type)
	binary.BigEndian.PutUint32(buf[17:21], uint32(datalen))

	// 拷贝数据（这里仍有拷贝，但重用了缓冲区）
	copy(buf[21:], msg.Data)

	// 创建新的切片返回（必须拷贝，因为缓冲区会被归还池中）
	result := make([]byte, totalLen)
	copy(result, buf)
	return result
}

// MessageWriter 零拷贝消息写入器，支持批量写入
type MessageWriter struct {
	writer io.Writer
}

// NewMessageWriter 创建新的消息写入器
func NewMessageWriter(writer io.Writer) *MessageWriter {
	return &MessageWriter{writer: writer}
}

// WriteMessage 零拷贝写入单个消息
func (mw *MessageWriter) WriteMessage(msg Message) error {
	return EncodeZeroCopy(mw.writer, msg)
}

// WriteMessages 零拷贝批量写入消息
func (mw *MessageWriter) WriteMessages(msgs []Message) error {
	for _, msg := range msgs {
		if err := mw.WriteMessage(msg); err != nil {
			return err
		}
	}
	return nil
}
