package main

import (
	"log"
	"net/http"
	"os"

	"github.com/harshyadavone/anonymous_chat/queue"
	"github.com/harshyadavone/anonymous_chat/store"

	"github.com/harshyadavone/tgx"
	"github.com/harshyadavone/tgx/models"
	"github.com/harshyadavone/tgx/pkg/logger"
)

var userStore = &store.UserStore{
	Users: make(map[int64]*store.User),
}

var waitingQueue = queue.NewQueue()

func main() {
	token := os.Getenv("BOT_TOKEN")
	webhookURL := os.Getenv("WEBHOOK_URL")

	logger := logger.NewDefaultLogger(logger.DEBUG)

	bot := tgx.NewBot(token, webhookURL, logger)

	logger.Info("Starting the bot...")

	bot.OnError(func(ctx *tgx.Context, err error) {
		payload := &tgx.SendMessageRequest{
			ChatId: ctx.ChatID,
			Text:   MessageErrSomethingWentWrong,
		}
		ctx.ReplyWithOpts(payload)
	})

	bot.OnCommand("start", func(ctx *tgx.Context) error {

		req := &tgx.SendMessageRequest{
			ChatId: ctx.ChatID,
			Text:   "👋 Welcome! Chat anonymously with random people here. Type /connect to start or /help for commands!",
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: inlineKeyboardButton,
			},
		}

		return bot.SendMessageWithOpts(req)
	})

	bot.OnCommand("help", func(ctx *tgx.Context) error {
		return ctx.Reply(MessageQuickGuide)
	})

	bot.OnCommand("connect", func(ctx *tgx.Context) error {
		return HandleConnect(bot, ctx.ChatID)
	})

	bot.OnCommand("status", func(ctx *tgx.Context) error {
		return HandleStatus(bot, ctx.ChatID)
	})

	bot.OnCommand("next", func(ctx *tgx.Context) error {
		return ctx.Reply(MessageFeatureNotImplemented)
	})

	bot.OnCommand("stop", func(ctx *tgx.Context) error {
		return HandleStop(bot, ctx.ChatID)
	})

	bot.SetMyCommands(Commands)

	bot.OnCallback("connect", func(ctx *tgx.CallbackContext) error {
		if err := HandleConnect(bot, ctx.GetChatID()); err != nil {
			return err
		}
		return ctx.AnswerCallback(&tgx.CallbackAnswerOptions{})
	})

	bot.OnCallback("status", func(ctx *tgx.CallbackContext) error {
		if err := HandleStatus(bot, ctx.GetChatID()); err != nil {
			return err
		}
		return ctx.AnswerCallback(&tgx.CallbackAnswerOptions{})
	})

	bot.OnMessage("Text", func(ctx *tgx.Context) error {

		partner, errMsg := CheckAndGetPartner(ctx.ChatID, userStore)
		if errMsg != "" {
			return ctx.Reply(errMsg)
		}

		return bot.SendMessage(partner.ChatId, ctx.Text)
	})

	bot.OnMessage("Animation", func(ctx *tgx.Context) error {

		partner, errMsg := CheckAndGetPartner(ctx.ChatID, userStore)
		if errMsg != "" {
			return ctx.Reply(errMsg)
		}

		req := &tgx.SendAnimationRequest{
			Animation: ctx.Animation.FileId,
			BaseMediaRequest: tgx.BaseMediaRequest{
				ChatId: partner.ChatId,
			},
		}
		return bot.SendAnimation(req)
	})

	bot.OnMessage("Photo", func(ctx *tgx.Context) error {

		partner, errMsg := CheckAndGetPartner(ctx.ChatID, userStore)
		if errMsg != "" {
			return ctx.Reply(errMsg)
		}

		req := &tgx.SendPhotoRequest{
			Photo: ctx.Photo[0].FileId,
			BaseMediaRequest: tgx.BaseMediaRequest{
				ChatId: partner.ChatId,
			},
		}
		return bot.SendPhoto(req)
	})

	bot.OnMessage("Voice", func(ctx *tgx.Context) error {

		partner, errMsg := CheckAndGetPartner(ctx.ChatID, userStore)
		if errMsg != "" {
			return ctx.Reply(errMsg)
		}

		req := &tgx.SendVoiceRequest{
			Voice: ctx.Voice.FileId,
			BaseMediaRequest: tgx.BaseMediaRequest{
				ChatId: partner.ChatId,
			},
		}
		return bot.SendVoice(req)
	})

	bot.OnMessage("Document", func(ctx *tgx.Context) error {

		partner, errMsg := CheckAndGetPartner(ctx.ChatID, userStore)
		if errMsg != "" {
			return ctx.Reply(errMsg)
		}

		req := &tgx.SendDocumentRequest{
			Document: ctx.Document.FileId,
			BaseMediaRequest: tgx.BaseMediaRequest{
				ChatId: partner.ChatId,
			},
		}

		return bot.SendDocument(req)
	})

	bot.OnMessage("Video", func(ctx *tgx.Context) error {

		partner, errMsg := CheckAndGetPartner(ctx.ChatID, userStore)
		if errMsg != "" {
			return ctx.Reply(errMsg)
		}

		req := &tgx.SendVideoRequest{
			Video: ctx.Video.FileId,
			BaseMediaRequest: tgx.BaseMediaRequest{
				ChatId: partner.ChatId,
			},
		}

		return bot.SendVideo(req)
	})

	bot.OnMessage("Sticker", func(ctx *tgx.Context) error {

		partner, errMsg := CheckAndGetPartner(ctx.ChatID, userStore)
		if errMsg != "" {
			return ctx.Reply(errMsg)
		}

		req := &tgx.SendStickerRequest{
			Sticker: ctx.Sticker.FileId,
			BaseMediaRequest: tgx.BaseMediaRequest{
				ChatId: partner.ChatId,
			},
		}

		return bot.SendSticker(req)
	})

	if err := bot.SetWebhook(); err != nil {
		log.Fatal("Failed to set webhook:", err)
	}

	logger.Info("Starting server on :8080")
	http.HandleFunc("/webhook", bot.HandleWebhook)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server error:", err)
	}
}

func HandleConnect(b *tgx.Bot, chatId int64) error {
	user, exists := userStore.GetUser(chatId)

	if !exists {
		user = &store.User{ChatId: chatId, IsConnecting: true}
		userStore.AddUser(user)
	}

	if user.IsConnected {
		return b.SendMessage(chatId, MessageAlreadyConnected)
	}

	waitingQueue.RemoveNode(chatId)

	partnerChatId, err := waitingQueue.Dequeue()

	if err == nil {
		partner, exists := userStore.GetUser(partnerChatId)

		if exists {
			user.IsConnected = true
			user.IsConnecting = false
			user.Partner = partner.ChatId

			partner.IsConnected = true
			partner.IsConnecting = false
			partner.Partner = user.ChatId

			if err := b.SendMessage(chatId, MessageConnected); err != nil {
				return err
			}
			return b.SendMessage(partner.ChatId, MessageConnected)
		}

	}

	waitingQueue.Enqueue(chatId)
	user.IsConnecting = true
	return b.SendMessage(chatId, MessageLookingForPartner)
}

func HandleStop(b *tgx.Bot, chatId int64) error {
	user, exists := userStore.GetUser(chatId)

	if !exists || !user.IsConnected {
		return b.SendMessage(chatId, MessageConnectWithSomeoneFirst)
	}

	if user.IsConnected {
		partner, partnerExists := userStore.GetUser(user.Partner)
		if partnerExists {
			partner.IsConnected = false
			partner.Partner = 0

			if err := b.SendMessage(partner.ChatId, MessagePartnerLeftChat); err != nil {
				return err
			}
		}

		user.IsConnected = false
		user.IsConnecting = false
		user.Partner = 0

		return b.SendMessage(chatId, MessageChatEnded)
	}
	return nil
}

func HandleStatus(b *tgx.Bot, chatId int64) error {
	user, exists := userStore.GetUser(chatId)
	if !exists {
		return b.SendMessage(chatId, MessageNotConnectedStatus)
	}

	if user.IsConnected {
		return b.SendMessage(chatId, MessageCurrentlyChatting)
	}

	if user.IsConnecting {
		return b.SendMessage(chatId, MessageInWaitingList)
	}

	return b.SendMessage(chatId, MessageNotConnectedStatus)
}

func CheckAndGetPartner(chatId int64, userStore *store.UserStore) (*store.User, string) {
	user, exists := userStore.GetUser(chatId)
	if !exists || !user.IsConnected {
		return nil, MessageNotConnected
	}

	partner, exists := userStore.GetUser(user.Partner)
	if !exists {
		return nil, MessagePartnerNotAvailable
	}
	return partner, ""
}
