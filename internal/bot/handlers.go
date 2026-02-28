package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"tg-monitor-bot/internal/storage"
)

// handleStart handles the /start command
func (b *Bot) handleStart(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	welcomeMsg := `🤖 *Outage Monitoring Bot*

I monitor your infrastructure and alert you when things go down!

*Source Management:*
/add\_source - Add a new monitoring source
/remove\_source <name> - Remove a source
/list\_sources - List all sources

*Status & History:*
/status [name] - View current status
/history <name> [limit] - View status change history

*Control:*
/check <name> - Manual check now
/pause <name> - Pause monitoring
/resume <name> - Resume monitoring

*Examples:*
` + "`/add_source Home_Power ping 192.168.1.1 10s 123456789`" + `
` + "`/status Home_Power`" + `
` + "`/history Home_Power 10`"

	_, err := tgBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      welcomeMsg,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		b.logger.Printf("Failed to send start message: %v", err)
	}
}

// handleAddSource handles the /add_source command
// Format: /add_source <name> <type> <target> <interval> <chat_ids>
// Example: /add_source Home_Power ping 192.168.1.1 10s 123456789,987654321
func (b *Bot) handleAddSource(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	args := strings.Fields(update.Message.Text)
	if len(args) < 5 {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			"❌ Usage: /add_source <name> <type> <target> <interval> <chat_ids>\n"+
				"Example: /add_source Home_Power ping 192.168.1.1 10s "+strconv.FormatInt(update.Message.Chat.ID, 10))
		return
	}

	name := args[1]
	sourceType := args[2]
	target := args[3]
	intervalStr := args[4]

	// Parse check interval
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Invalid interval '%s'. Use format like: 10s, 1m, 5m", intervalStr))
		return
	}

	// Validate type
	if sourceType != "ping" && sourceType != "http" {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			"❌ Type must be 'ping' or 'http'")
		return
	}

	// Parse chat IDs (optional, defaults to current chat)
	var chatIDs []int64
	if len(args) >= 6 {
		chatIDsStr := strings.Split(args[5], ",")
		for _, idStr := range chatIDsStr {
			id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
			if err == nil {
				chatIDs = append(chatIDs, id)
			}
		}
	} else {
		chatIDs = []int64{update.Message.Chat.ID}
	}

	// Perform initial check to determine starting status
	source := &storage.Source{
		Name:          name,
		Type:          sourceType,
		Target:        target,
		CheckInterval: interval,
		Enabled:       true,
		CreatedAt:     time.Now(),
	}

	// Do initial check
	initialStatus := b.monitor.CheckSource(source)
	source.CurrentStatus = initialStatus
	source.LastCheckTime = time.Now()
	source.LastChangeTime = time.Now()

	// Save source to database
	if err := b.storage.SaveSource(source); err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Failed to save source: %v", err))
		return
	}

	// Add chat associations
	for _, chatID := range chatIDs {
		if err := b.storage.AddSourceChat(source.ID, chatID); err != nil {
			b.logger.Printf("Failed to add chat %d to source: %v", chatID, err)
		}
	}

	// Start monitoring
	monitorCtx := context.Background() // Use background context for long-running monitor
	if err := b.monitor.AddSource(monitorCtx, source); err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Failed to start monitoring: %v", err))
		return
	}

	statusEmoji := "🔴"
	statusText := "OFFLINE"
	if initialStatus == 1 {
		statusEmoji = "🟢"
		statusText = "ONLINE"
	}

	b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
		fmt.Sprintf("✅ Source added and monitoring started!\n\n"+
			"Name: %s\n"+
			"Type: %s\n"+
			"Target: %s\n"+
			"Interval: %v\n"+
			"Initial status: %s %s\n"+
			"Notifying %d chat(s)",
			name, sourceType, target, interval, statusEmoji, statusText, len(chatIDs)))
}

// handleRemoveSource handles the /remove_source command
func (b *Bot) handleRemoveSource(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			"❌ Usage: /remove_source <name>")
		return
	}

	name := strings.Join(args[1:], " ")

	// Find source by name
	source, err := b.storage.GetSourceByName(name)
	if err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Source not found: %s", name))
		return
	}

	// Stop monitoring
	if err := b.monitor.RemoveSource(source.ID); err != nil {
		b.logger.Printf("Failed to stop monitoring: %v", err)
	}

	// Remove chat associations
	if err := b.storage.RemoveAllSourceChats(source.ID); err != nil {
		b.logger.Printf("Failed to remove chat associations: %v", err)
	}

	// Delete source
	if err := b.storage.DeleteSource(source.ID); err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Failed to delete source: %v", err))
		return
	}

	b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
		fmt.Sprintf("✅ Source '%s' removed and monitoring stopped", name))
}

// handleListSources handles the /list_sources command
func (b *Bot) handleListSources(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	sources, err := b.storage.GetAllSources()
	if err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Failed to get sources: %v", err))
		return
	}

	if len(sources) == 0 {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			"📋 No sources configured.\n\nUse /add_source to add one!")
		return
	}

	var message strings.Builder
	message.WriteString("📋 *Monitoring Sources*\n\n")

	for i, source := range sources {
		statusEmoji := "🔴"
		statusText := "OFFLINE"
		if source.CurrentStatus == 1 {
			statusEmoji = "🟢"
			statusText = "ONLINE"
		}

		enabledText := ""
		if !source.Enabled {
			enabledText = " (PAUSED)"
		}

		timeSinceCheck := time.Since(source.LastCheckTime)
		timeSinceChange := time.Since(source.LastChangeTime)

		message.WriteString(fmt.Sprintf("%d. *%s* %s %s%s\n", i+1, source.Name, statusEmoji, statusText, enabledText))
		message.WriteString(fmt.Sprintf("   Type: %s (%s)\n", source.Type, source.Target))
		message.WriteString(fmt.Sprintf("   Check: every %v (last %v ago)\n", source.CheckInterval, formatDuration(timeSinceCheck)))

		if source.CurrentStatus == 1 {
			message.WriteString(fmt.Sprintf("   Uptime: %v\n", formatDuration(timeSinceChange)))
		} else {
			message.WriteString(fmt.Sprintf("   Downtime: %v\n", formatDuration(timeSinceChange)))
		}

		message.WriteString("\n")
	}

	_, err = tgBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      message.String(),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		b.logger.Printf("Failed to send list: %v", err)
	}
}

// handleStatus handles the /status command
func (b *Bot) handleStatus(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	args := strings.Fields(update.Message.Text)

	// If specific source requested
	if len(args) >= 2 {
		name := strings.Join(args[1:], " ")
		source, err := b.storage.GetSourceByName(name)
		if err != nil {
			b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
				fmt.Sprintf("❌ Source not found: %s", name))
			return
		}

		b.showSourceStatus(ctx, tgBot, update.Message.Chat.ID, source)
		return
	}

	// Show summary of all sources
	sources, err := b.storage.GetAllSources()
	if err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Failed to get sources: %v", err))
		return
	}

	if len(sources) == 0 {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			"📊 No sources to monitor")
		return
	}

	online := 0
	for _, source := range sources {
		if source.CurrentStatus == 1 {
			online++
		}
	}

	message := fmt.Sprintf("📊 *Overall Status*\n\n"+
		"Total sources: %d\n"+
		"🟢 Online: %d\n"+
		"🔴 Offline: %d\n\n"+
		"Use `/status <name>` for details",
		len(sources), online, len(sources)-online)

	_, err = tgBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		b.logger.Printf("Failed to send status: %v", err)
	}
}

// showSourceStatus shows detailed status for a specific source
func (b *Bot) showSourceStatus(ctx context.Context, tgBot *bot.Bot, chatID int64, source *storage.Source) {
	statusEmoji := "🔴"
	statusText := "OFFLINE"
	if source.CurrentStatus == 1 {
		statusEmoji = "🟢"
		statusText = "ONLINE"
	}

	timeSinceCheck := time.Since(source.LastCheckTime)
	timeSinceChange := time.Since(source.LastChangeTime)

	var durationText string
	if source.CurrentStatus == 1 {
		durationText = fmt.Sprintf("Uptime: %v", formatDuration(timeSinceChange))
	} else {
		durationText = fmt.Sprintf("Downtime: %v", formatDuration(timeSinceChange))
	}

	message := fmt.Sprintf("%s *%s*: %s\n\n"+
		"Target: %s (%s)\n"+
		"Check interval: %v\n"+
		"Last check: %v ago\n"+
		"%s\n"+
		"Status: %s",
		statusEmoji, source.Name, statusText,
		source.Target, source.Type,
		source.CheckInterval,
		formatDuration(timeSinceCheck),
		durationText,
		func() string {
			if source.Enabled {
				return "Enabled"
			}
			return "⏸ Paused"
		}())

	_, err := tgBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		b.logger.Printf("Failed to send status: %v", err)
	}
}

// handleHistory handles the /history command
func (b *Bot) handleHistory(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			"❌ Usage: /history <name> [limit]\n"+
				"Example: /history Home_Power 10")
		return
	}

	limit := 10
	name := args[1]

	// Parse limit if provided
	if len(args) >= 3 {
		if l, err := strconv.Atoi(args[2]); err == nil && l > 0 {
			limit = l
		}
	}

	// Find source
	source, err := b.storage.GetSourceByName(name)
	if err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Source not found: %s", name))
		return
	}

	// Get status changes
	changes, err := b.storage.GetStatusChanges(source.ID, limit)
	if err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Failed to get history: %v", err))
		return
	}

	if len(changes) == 0 {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("📜 No status changes recorded for '%s'", name))
		return
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("📜 *Status History: %s*\n\n", name))

	for i, change := range changes {
		timeAgo := time.Since(change.Timestamp)
		duration := time.Duration(change.DurationMs) * time.Millisecond

		oldEmoji := "🔴"
		newEmoji := "🟢"
		if change.OldStatus == 1 {
			oldEmoji = "🟢"
		}
		if change.NewStatus == 0 {
			newEmoji = "🔴"
		}

		message.WriteString(fmt.Sprintf("%d. %s → %s (%v ago)\n",
			i+1, oldEmoji, newEmoji, formatDuration(timeAgo)))

		if change.OldStatus == 1 {
			message.WriteString(fmt.Sprintf("   Uptime was: %v\n", formatDuration(duration)))
		} else {
			message.WriteString(fmt.Sprintf("   Downtime was: %v\n", formatDuration(duration)))
		}

		message.WriteString("\n")
	}

	_, err = tgBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      message.String(),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		b.logger.Printf("Failed to send history: %v", err)
	}
}

// handleCheck handles the /check command (manual check)
func (b *Bot) handleCheck(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			"❌ Usage: /check <name>")
		return
	}

	name := strings.Join(args[1:], " ")

	source, err := b.storage.GetSourceByName(name)
	if err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Source not found: %s", name))
		return
	}

	b.sendMessage(ctx, tgBot, update.Message.Chat.ID, "🔍 Checking...")

	status := b.monitor.CheckSource(source)

	statusEmoji := "🔴"
	statusText := "OFFLINE"
	if status == 1 {
		statusEmoji = "🟢"
		statusText = "ONLINE"
	}

	b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
		fmt.Sprintf("%s *%s* is %s\n\nType: %s\nTarget: %s",
			statusEmoji, name, statusText, source.Type, source.Target))
}

// handlePause handles the /pause command
func (b *Bot) handlePause(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			"❌ Usage: /pause <name>")
		return
	}

	name := strings.Join(args[1:], " ")

	source, err := b.storage.GetSourceByName(name)
	if err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Source not found: %s", name))
		return
	}

	if err := b.monitor.PauseSource(source.ID); err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Failed to pause: %v", err))
		return
	}

	b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
		fmt.Sprintf("⏸ Monitoring paused for: *%s*\n\nNotifications will not be sent until resumed.", name))
}

// handleResume handles the /resume command
func (b *Bot) handleResume(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			"❌ Usage: /resume <name>")
		return
	}

	name := strings.Join(args[1:], " ")

	source, err := b.storage.GetSourceByName(name)
	if err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Source not found: %s", name))
		return
	}

	if err := b.monitor.ResumeSource(source.ID); err != nil {
		b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
			fmt.Sprintf("❌ Failed to resume: %v", err))
		return
	}

	b.sendMessage(ctx, tgBot, update.Message.Chat.ID,
		fmt.Sprintf("▶️ Monitoring resumed for: *%s*", name))
}

// formatStatusChangeMessage formats a notification message for a status change
func (b *Bot) formatStatusChangeMessage(source *storage.Source, change *storage.StatusChange) string {
	duration := time.Duration(change.DurationMs) * time.Millisecond

	checkType := source.Type
	if source.Target != "" {
		checkType = fmt.Sprintf("%s (%s)", source.Type, source.Target)
	}

	if change.NewStatus == 1 {
		// Restored (OFFLINE → ONLINE)
		return fmt.Sprintf("🟢 <b>RESTORED</b>\n"+
			"%s is now <b>ONLINE</b>\n\n"+
			"Downtime: %v\n"+
			"Check type: %s\n"+
			"Time: %s",
			source.Name,
			formatDuration(duration),
			checkType,
			change.Timestamp.Format("2006-01-02 15:04:05"))
	}

	// Outage (ONLINE → OFFLINE)
	return fmt.Sprintf("🔴 <b>OUTAGE DETECTED</b>\n"+
		"%s is now <b>OFFLINE</b>\n\n"+
		"Was online for: %v\n"+
		"Check type: %s\n"+
		"Time: %s",
		source.Name,
		formatDuration(duration),
		checkType,
		change.Timestamp.Format("2006-01-02 15:04:05"))
}

// Helper function to send a message
func (b *Bot) sendMessage(ctx context.Context, tgBot *bot.Bot, chatID int64, text string) {
	_, err := tgBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		b.logger.Printf("Failed to send message: %v", err)
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		if seconds == 0 {
			return fmt.Sprintf("%d minutes", minutes)
		}
		return fmt.Sprintf("%d minutes %d seconds", minutes, seconds)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%d hours", hours)
		}
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours == 0 {
		return fmt.Sprintf("%d days", days)
	}
	return fmt.Sprintf("%d days %d hours", days, hours)
}
