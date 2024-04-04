package commands

import (
	"fmt"
	"github.com/akimsavvin/tablet_telegram/internal/dto"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/constraints"
	"log"
)

type ITabletService interface {
	Create(createDTO *dto.CreateTabletDTO) error
}

type Create struct {
	chatID int64
	userID int64
	dto    *dto.CreateTabletDTO
}

func NewCreate[T constraints.Signed](chatID, userID T, dto *dto.CreateTabletDTO) Create {
	return Create{
		chatID: int64(chatID),
		userID: int64(userID),
		dto:    dto,
	}
}

type CreateHandler struct {
	svc ITabletService
}

func NewCreateHandler(svc ITabletService) *CreateHandler {
	return &CreateHandler{svc: svc}
}

func (h *CreateHandler) Handle(event Create) (tgbotapi.MessageConfig, error) {
	err := h.svc.Create(event.dto)
	if err != nil {
		log.Printf("Could not create tablet due to error: %s", err.Error())
		return tgbotapi.MessageConfig{}, err
	}

	msgText := fmt.Sprintf("Отлично! В %d:%d, тебе будут приходить сообщения с напоминанием!", event.dto.UseHour, event.dto.UseMinute)
	msgCfg := tgbotapi.NewMessage(event.chatID, msgText)
	return msgCfg, nil
}
