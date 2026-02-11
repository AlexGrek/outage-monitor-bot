package bot

import (
	"context"
	"log"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"tg-monitor-bot/internal/config"
	"tg-monitor-bot/internal/monitor"
	"tg-monitor-bot/internal/storage"
)

type Bot struct {
	bot     *bot.Bot
	config  *config.Config
	storage *storage.BoltDB
	monitor *monitor.Monitor
	logger  *log.Logger
}

// New creates a new Bot instance
func New(cfg *config.Config, db *storage.BoltDB, mon *monitor.Monitor) (*Bot, error) {
	b := &Bot{
		config:  cfg,
		storage: db,
		monitor: mon,
		logger:  log.New(log.Writer(), "[BOT] ", log.LstdFlags),
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(b.loggingMiddleware(b.authMiddleware(b.defaultHandler))),
	}

	tgBot, err := bot.New(cfg.TelegramToken, opts...)
	if err != nil {
		return nil, err
	}

	b.bot = tgBot

	// Register command handlers
	b.registerHandlers()

	return b, nil
}

// Start starts the bot
func (b *Bot) Start(ctx context.Context) {
	b.bot.Start(ctx)
}

// SetMonitor sets the monitor reference (used during initialization)
func (b *Bot) SetMonitor(mon *monitor.Monitor) {
	b.monitor = mon
}

// registerHandlers registers all command handlers
func (b *Bot) registerHandlers() {
	// Basic commands
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, b.handleStart)

	// Source management
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/add_source", bot.MatchTypePrefix, b.handleAddSource)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/remove_source", bot.MatchTypePrefix, b.handleRemoveSource)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/list_sources", bot.MatchTypeExact, b.handleListSources)

	// Status and history
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/status", bot.MatchTypePrefix, b.handleStatus)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/history", bot.MatchTypePrefix, b.handleHistory)

	// Control
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/check", bot.MatchTypePrefix, b.handleCheck)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/pause", bot.MatchTypePrefix, b.handlePause)
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/resume", bot.MatchTypePrefix, b.handleResume)
}

// loggingMiddleware logs all incoming updates
func (b *Bot) loggingMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
		start := time.Now()

		var userInfo string
		if update.Message != nil && update.Message.From != nil {
			userInfo = update.Message.From.Username
			if userInfo == "" {
				userInfo = update.Message.From.FirstName
			}
		}

		b.logger.Printf("Received update from user: %s", userInfo)

		next(ctx, tgBot, update)

		b.logger.Printf("Processed update in %v", time.Since(start))
	}
}

// authMiddleware checks if user is authorized
func (b *Bot) authMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.From == nil {
			return
		}

		userID := update.Message.From.ID

		// Check if user is in allowed list (if configured)
		if len(b.config.AllowedUsers) > 0 {
			allowed := false
			for _, allowedID := range b.config.AllowedUsers {
				if userID == allowedID {
					allowed = true
					break
				}
			}

			if !allowed {
				b.logger.Printf("Unauthorized access attempt from user ID: %d", userID)
				_, _ = tgBot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "❌ Unauthorized. You are not allowed to use this bot.",
				})
				return
			}
		}

		next(ctx, tgBot, update)
	}
}

// defaultHandler handles unknown commands
func (b *Bot) defaultHandler(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	_, _ = tgBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "❓ Unknown command. Use /start to see available commands.",
	})
}

// OnStatusChange is called by the Monitor when a source's status changes
func (b *Bot) OnStatusChange(source *storage.Source, change *storage.StatusChange) {
	ctx := context.Background()

	// Get all chats for this source
	chatIDs, err := b.storage.GetSourceChats(source.ID)
	if err != nil {
		b.logger.Printf("Failed to get chats for source %s: %v", source.Name, err)
		return
	}

	// Format notification message
	message := b.formatStatusChangeMessage(source, change)

	// Send to all configured chats
	for _, chatID := range chatIDs {
		_, err := b.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   message,
		})
		if err != nil {
			b.logger.Printf("Failed to send notification to chat %d: %v", chatID, err)
		} else {
			b.logger.Printf("Sent status change notification for %s to chat %d", source.Name, chatID)
		}
	}
}
