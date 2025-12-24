package test

import (
	"testing"

	"wsim/gateway/model"
)

func TestSend(t *testing.T) {
	model.InitSend()
	model.SendMessage(model.Message{
		FromUserID: 1,
		ToUserID:   2,
		Type:       model.MessageTypeText,
		Data:       []byte("Hello, World!"),
	})
	model.SendClose()
}
