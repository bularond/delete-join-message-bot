package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
	"strconv"
	"time"
)

type ProjectContext struct {
	bot        *tgbotapi.BotAPI
	captchaMap map[int64]chan struct{}
}

func getToken() string {
	return os.Getenv("TELEGRAM_BOT_TOKEN")
}

func deleteMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) error {
	msgToDelete := tgbotapi.DeleteMessageConfig{
		ChatID:    message.Chat.ID,
		MessageID: message.MessageID,
	}

	_, err := bot.Request(msgToDelete)
	return err
}

func banUser(bot *tgbotapi.BotAPI, user *tgbotapi.User, chatId int64) error {
	botRequest := tgbotapi.BanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: chatId,
			UserID: user.ID,
		},
		UntilDate:      0,
		RevokeMessages: true,
	}

	_, err := bot.Request(botRequest)
	return err
}

func getEntryMessage(user *tgbotapi.User) string {
	return fmt.Sprintf("[%s](@%s), добро пожаловать в клуб настольных игр! "+
		"Если ты не бот, нажми пожалуйста на кнопку",
		user.FirstName, user.UserName)
}

func getInlineKeyboard(user *tgbotapi.User) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Я не бот", strconv.FormatInt(user.ID, 10)),
		),
	)
}

func sendCaptcha(bot *tgbotapi.BotAPI, message *tgbotapi.Message) (tgbotapi.Message, error) {
	msgToSend := tgbotapi.NewMessage(message.Chat.ID, getEntryMessage(message.From))
	msgToSend.ReplyMarkup = getInlineKeyboard(message.From)
	msgToSend.ParseMode = tgbotapi.ModeMarkdown

	newMessage, err := bot.Send(msgToSend)
	return newMessage, err
}

func handleNewUser(pc *ProjectContext, user *tgbotapi.User, message *tgbotapi.Message) {
	newCaptchaPass := make(chan struct{})
	pc.captchaMap[user.ID] = newCaptchaPass
	go func() {
		select {
		case <-time.After(120 * time.Second):
			err := deleteMessage(pc.bot, message)
			if err != nil {
				log.Printf("Error while delete message: %v\n", err)
			}
			err = banUser(pc.bot, user, message.Chat.ID)
			if err != nil {
				log.Printf("Error while delete message: %v\n", err)
			}
		case <-newCaptchaPass:
		}
		delete(pc.captchaMap, user.ID)
	}()
}

func handleUpdate(pc *ProjectContext, update *tgbotapi.Update) error {
	if update.Message != nil && update.Message.LeftChatMember != nil {
		err := deleteMessage(pc.bot, update.Message)
		if err != nil {
			return err
		}
	} else if update.Message != nil && update.Message.NewChatMembers != nil {
		err := deleteMessage(pc.bot, update.Message)
		if err != nil {
			return err
		}

		invited := update.Message.NewChatMembers[0]
		inviting := update.Message.From
		if inviting.ID == invited.ID {
			newMessage, err := sendCaptcha(pc.bot, update.Message)
			if err != nil {
				return err
			}

			handleNewUser(pc, inviting, &newMessage)
		}
	} else if update.Message != nil {
		_, ok := pc.captchaMap[update.Message.From.ID]
		if ok {
			err := deleteMessage(pc.bot, update.Message)
			if err != nil {
				return err
			}
		}
	} else if update.CallbackQuery != nil {
		if update.CallbackQuery.Data == strconv.FormatInt(update.CallbackQuery.From.ID, 10) {
			err := deleteMessage(pc.bot, update.CallbackQuery.Message)
			if err != nil {
				return err
			}
			pc.captchaMap[update.CallbackQuery.From.ID] <- struct{}{}
		}
	}
	return nil
}

func getProjectContext() *ProjectContext {
	bot, err := tgbotapi.NewBotAPI(getToken())
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &ProjectContext{
		bot:        bot,
		captchaMap: make(map[int64]chan struct{}),
	}
}

func main() {
	pc := getProjectContext()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := pc.bot.GetUpdatesChan(u)
	for update := range updates {
		log.Printf("New update: %v", update)
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}

		err := handleUpdate(pc, &update)
		if err != nil {
			log.Printf("Error while handle message: %v\n", err)
		}
	}
}
