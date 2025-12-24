package model

type Message struct {
	FromUserID int64       `json:"from_user_id"`
	ToUserID   int64       `json:"to_user_id"`
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
)

func (m MessageType) Int() int {
	return int(m)
}


