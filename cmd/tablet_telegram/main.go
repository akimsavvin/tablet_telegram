package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/akimsavvin/tablet_telegram/internal/commands"
	"github.com/akimsavvin/tablet_telegram/internal/dto"
	"github.com/akimsavvin/tablet_telegram/internal/models"
	"github.com/akimsavvin/tablet_telegram/internal/services"
	"github.com/akimsavvin/tablet_telegram/pkg/cache"
	"github.com/akimsavvin/tablet_telegram/pkg/cqrs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/redis/go-redis/v9"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UserState int

const (
	StateDefault      UserState = 0
	StateCreatingName UserState = 1
	StateCreatingTime UserState = 2
)

func main() {
	// Get environment variables
	telegramApiToken := os.Getenv("TELEGRAM_API_TOKEN")

	// Create a bot
	bot, err := tgbotapi.NewBotAPI(telegramApiToken)
	if err != nil {
		log.Fatalf("Could not create bot due to error: %s", err.Error())
	} else {
		log.Println("Created new bot")
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		runBot(bot)
	}()

	go func() {
		defer wg.Done()
		runConsumer(bot)
	}()

	wg.Wait()
}

func runBot(bot *tgbotapi.BotAPI) {
	// Get environment variables
	backendURL := os.Getenv("BACKEND_URL")

	// Services
	redisClient := cache.NewClient()
	tabletService := services.NewTabletService(backendURL)

	// Creating handlers
	startCommandHandler := commands.NewStartHandler()
	createCommandHandler := commands.NewCreateHandler(tabletService)

	// Registering handlers
	cqrs.Register[commands.Start, tgbotapi.MessageConfig](startCommandHandler)
	cqrs.Register[commands.Create, tgbotapi.MessageConfig](createCommandHandler)

	for update := range bot.GetUpdatesChan(tgbotapi.NewUpdate(0)) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		msg := update.Message
		userID := msg.From.ID

		userState := StateDefault

		var (
			userStateKey     = fmt.Sprintf("user:%d.state", userID)
			userStateNameKey = fmt.Sprintf("user:%d.state.name", userID)
		)

		if cmd := redisClient.Get(ctx, userStateKey); cmd.Err() != nil {
			if errors.Is(cmd.Err(), redis.Nil) {
				userState = StateDefault
			} else {
				fmt.Printf("Could not get user state due to error: %s\n", cmd.Err().Error())
				continue
			}
		} else {
			intVal, _ := cmd.Int()
			userState = UserState(intVal)
		}

		if !msg.IsCommand() {
			switch userState {
			case StateCreatingName:
				msgText := "Круто!\nТеперь укажи время в которое ты будешь ее пить в формате '14:30'"
				if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, msgText)); err != nil {
					fmt.Println(err.Error())
				} else {
					cmd := redisClient.Set(ctx, userStateNameKey, msg.Text, 0)
					if cmd.Err() != nil {
						log.Printf("Could not set user state name: %s\n", cmd.Err().Error())
						continue
					}

					cmd = redisClient.Set(ctx, userStateKey, int(StateCreatingTime), 0)
					if cmd.Err() != nil {
						log.Printf("Could not set user state: %s\n", cmd.Err().Error())
						continue
					}
				}

				continue
			case StateCreatingTime:
				time := msg.Text

				hoursMinutes := strings.Split(time, ":")
				hours, err := strconv.ParseInt(hoursMinutes[0], 10, 64)
				if err != nil || hours > 24 || hours < 0 {
					msgText := "Введи корректное время в формате '14:30'"
					if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, msgText)); err != nil {
						fmt.Println(err.Error())
					}

					continue
				}

				minutes, err := strconv.ParseInt(hoursMinutes[1], 10, 64)
				if err != nil || minutes > 60 || minutes < 0 {
					msgText := "Введи корректное время в формате '14:30'"
					if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, msgText)); err != nil {
						fmt.Println(err.Error())
					}

					continue
				}

				userStateName, err := redisClient.Get(ctx, userStateNameKey).Result()
				if err != nil {
					log.Printf("Could not get user state name due to error: %s\n", err.Error())
					continue
				}

				createDTO := &dto.CreateTabletDTO{
					UserTelegramID: userID,
					Name:           userStateName,
					UseHour:        int(hours),
					UseMinute:      int(minutes),
				}

				createCmd := commands.NewCreate(msg.Chat.ID, msg.From.ID, createDTO)

				msgCfg, err := cqrs.Handle[tgbotapi.MessageConfig](createCmd)
				if err != nil {
					if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, err.Error())); err != nil {
						fmt.Println(err.Error())
					}

					continue
				}

				if _, err := bot.Send(msgCfg); err != nil {
					fmt.Println(err.Error())
				}

				continue
			}
		}

		switch msg.Command() {
		case "start":
			command := commands.NewStart(msg.Chat.ID)
			msgCfg, _ := cqrs.Handle[tgbotapi.MessageConfig](command)

			if _, err := bot.Send(msgCfg); err != nil {
				fmt.Println(err.Error())
			}
		case "create":
			if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Какое будет название у таблетки?")); err != nil {
				fmt.Println(err.Error())
			} else {
				cmd := redisClient.Set(ctx, userStateKey, int(StateCreatingName), 0)
				if cmd.Err() != nil {
					log.Printf("Could not set user state: %s\v", cmd.Err().Error())
					continue
				}
			}
		}
	}
}

func runConsumer(bot *tgbotapi.BotAPI) {
	// Get environment variables
	kafkaURL := os.Getenv("KAFKA_URL")

	// Setup kafka
	kafkaCfg := sarama.NewConfig()
	consumer, err := sarama.NewConsumer([]string{kafkaURL}, kafkaCfg)
	if err != nil {
		log.Fatalf("Could not create kafka consumer due to error: %s", err.Error())
	}
	defer consumer.Close()

	partConsumer, err := consumer.ConsumePartition("TabletsSchedule", 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatalf("Failed to consume partition: %s", err.Error())
	}

	for {
		select {
		case msg, ok := <-partConsumer.Messages():
			if !ok {
				log.Fatalf("Consumer closed")
			}

			// Десериализация входящего сообщения из JSON
			tablet := new(models.Tablet)
			err := json.Unmarshal(msg.Value, tablet)

			if err != nil {
				log.Printf("Could not unmarshal JSON: %s\v", err.Error())
				continue
			}

			log.Printf("Received a tablet message: %v\v", tablet)

			tgMsgText := fmt.Sprintf("Хай, сучка! Пора выпить таблеточку '%s'", tablet.Name)
			tgMsgCfg := tgbotapi.NewMessage(tablet.UserTelegramID, tgMsgText)
			if _, err := bot.Send(tgMsgCfg); err != nil {
				fmt.Println(err.Error())
			}
		}
	}
}
