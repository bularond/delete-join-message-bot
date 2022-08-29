package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getToken() string {
	return os.Getenv("TELEGRAM_BOT_TOKEN")
}

func main() {
	bot, err := tgbotapi.NewBotAPI(getToken())
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		log.Printf("New update: %v", update)
		if update.Message == nil {
			continue
		}

		if update.Message.NewChatMembers != nil || update.Message.LeftChatMember != nil {
			msgToDelete := tgbotapi.DeleteMessageConfig{
				ChatID:    update.Message.Chat.ID,
				MessageID: update.Message.MessageID,
			}

			_, err = bot.Request(msgToDelete)
			if err != nil {
				log.Printf("Error while deleting message: %v\n", err)
			}
		}
	}
}
