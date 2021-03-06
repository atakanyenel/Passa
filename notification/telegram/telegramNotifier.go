//Package telegram provides functions for Telegram communication.
package telegram

import (
	"errors"

	"github.com/op/go-logging"

	"os"
	"strconv"

	"gopkg.in/telegram-bot-api.v4"
)

//InitializeClient creates a new telegram bot
type telegramClient struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

var log = logging.MustGetLogger("passa")

//InitializeClient starts the telegram client
func InitializeClient() (*telegramClient, error) {
	telegramToken := os.Getenv("telegramToken")
	if telegramToken == "" {
		return nil, errors.New("No token variable")
	}
	chatID, err := strconv.ParseInt(os.Getenv("chatID"), 10, 64)
	if err != nil {
		return nil, err
	}
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return nil, err
	}

	log.Debug("Authorized on account %s", bot.Self.UserName)
	return &telegramClient{bot: bot, chatID: chatID}, nil
}

//Notify notifies the user with the preffered channel
func (tc telegramClient) Notify(message string) {
	msg := tgbotapi.NewMessage(tc.chatID, message)
	tc.bot.Send(msg)
}
