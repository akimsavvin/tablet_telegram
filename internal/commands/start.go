package commands

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/constraints"
)

type Start struct {
	chatID int64
}

func NewStart[T constraints.Signed](chatID T) Start {
	return Start{chatID: int64(chatID)}
}

type StartHandler struct {
}

func NewStartHandler() *StartHandler {
	return &StartHandler{}
}

func (h *StartHandler) Handle(event Start) (tgbotapi.MessageConfig, error) {
	msgText := "Привет!\nТы можешь создать таблетку с помощью /create и тебе будут приходить сообщение, с напоминанием о том, что ее нужно выпить!"
	msgCfg := tgbotapi.NewMessage(event.chatID, msgText)
	return msgCfg, nil
}
