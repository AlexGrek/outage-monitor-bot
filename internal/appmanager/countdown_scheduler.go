package appmanager

import (
	"context"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"tg-monitor-bot/internal/bot"
	"tg-monitor-bot/internal/notifier"
	"tg-monitor-bot/internal/storage"
)

// countdownScheduler fires daily "days until a date" notifications.
//
// Once a minute it scans all enabled countdowns. For each, it computes the
// current wall-clock time in the countdown's timezone; when that time has
// reached the configured notify time and a notification has not already been
// sent today, it dispatches the rendered message to the countdown's sinks.
//
// Using a "send if past notify time and not yet sent today" rule (rather than
// requiring an exact minute match) makes the scheduler robust to restarts and
// brief downtime: a missed minute still results in a single send for the day.
type countdownScheduler struct {
	storage         *storage.BoltDB
	bot             *bot.Bot // may be nil in web-only mode (Telegram disabled)
	webhookNotifier *notifier.WebhookNotifier
	logger          *log.Logger
}

func newCountdownScheduler(db *storage.BoltDB, b *bot.Bot, wn *notifier.WebhookNotifier) *countdownScheduler {
	return &countdownScheduler{
		storage:         db,
		bot:             b,
		webhookNotifier: wn,
		logger:          log.New(log.Writer(), "[COUNTDOWN] ", log.LstdFlags),
	}
}

// run blocks until ctx is cancelled, ticking once a minute.
func (s *countdownScheduler) run(ctx context.Context) {
	s.logger.Println("Countdown scheduler started")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	s.tick() // immediate check on start

	for {
		select {
		case <-ctx.Done():
			s.logger.Println("Countdown scheduler stopping")
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

// tick processes all enabled countdowns once.
func (s *countdownScheduler) tick() {
	countdowns, err := s.storage.ListCountdowns()
	if err != nil {
		s.logger.Printf("Failed to list countdowns: %v", err)
		return
	}

	now := time.Now()
	for _, c := range countdowns {
		if !c.Enabled {
			continue
		}
		loc, err := time.LoadLocation(c.Timezone)
		if err != nil {
			s.logger.Printf("Countdown %q: invalid timezone %q: %v", c.Name, c.Timezone, err)
			continue
		}
		if !countdownDue(c, loc, now) {
			continue
		}

		daysLeft, err := computeDaysLeft(c.TargetDate, loc, now)
		if err != nil {
			s.logger.Printf("Countdown %q: invalid target date %q: %v", c.Name, c.TargetDate, err)
			continue
		}

		// Mark as sent for today before dispatching so an overlapping/next tick
		// cannot send a duplicate, even if dispatch is slow.
		todayStr := now.In(loc).Format("2006-01-02")
		if err := s.storage.UpdateCountdownLastSent(c.ID, todayStr); err != nil {
			s.logger.Printf("Countdown %q: failed to record last sent date: %v", c.Name, err)
		}

		go s.dispatch(c, daysLeft)

		// Auto-delete if target date is reached or passed
		if daysLeft <= 0 {
			if err := s.storage.DeleteCountdown(c.ID); err != nil {
				s.logger.Printf("Countdown %q: failed to auto-delete: %v", c.Name, err)
			} else {
				s.logger.Printf("Countdown %q: auto-deleted because it reached %d days left", c.Name, daysLeft)
			}
		}
	}
}

// dispatch renders the message and sends it to every sink of the countdown.
// It does not touch LastSentDate, so it is also safe to call for "test now".
func (s *countdownScheduler) dispatch(c *storage.Countdown, daysLeft int) {
	message := renderCountdownMessage(c.MessageTemplate, daysLeft)

	// Telegram sinks
	if len(c.ChatIDs) > 0 {
		if s.bot != nil {
			ctx := context.Background()
			for _, chatID := range c.ChatIDs {
				if err := s.bot.SendText(ctx, chatID, message); err != nil {
					s.logger.Printf("Countdown %q: failed to send to chat %d: %v", c.Name, chatID, err)
				} else {
					s.logger.Printf("Countdown %q: sent to chat %d (%d days left)", c.Name, chatID, daysLeft)
				}
			}
		} else {
			s.logger.Printf("Countdown %q: Telegram bot unavailable (web-only mode), skipping %d chat(s)", c.Name, len(c.ChatIDs))
		}
	}

	// Webhook sinks
	if len(c.WebhookIDs) > 0 && s.webhookNotifier != nil {
		payload := notifier.CountdownPayload{
			Type:        "countdown",
			CountdownID: c.ID,
			Name:        c.Name,
			TargetDate:  c.TargetDate,
			DaysLeft:    daysLeft,
			Message:     message,
			Timestamp:   time.Now().Format(time.RFC3339),
		}
		for _, whID := range c.WebhookIDs {
			wh, err := s.storage.GetWebhook(whID)
			if err != nil {
				s.logger.Printf("Countdown %q: webhook %s not found: %v", c.Name, whID, err)
				continue
			}
			s.webhookNotifier.SendCountdown(wh, payload)
		}
	}
}

// renderCountdownMessage substitutes every "{}" in the template with the day count.
func renderCountdownMessage(template string, daysLeft int) string {
	return strings.ReplaceAll(template, "{}", strconv.Itoa(daysLeft))
}

// countdownDue reports whether a notification should be sent for c given the
// current time. It is true when "now" (in the countdown's timezone) has reached
// the configured notify time for today and no notification has been sent today.
func countdownDue(c *storage.Countdown, loc *time.Location, now time.Time) bool {
	nowLoc := now.In(loc)
	if c.LastSentDate == nowLoc.Format("2006-01-02") {
		return false
	}
	nt, err := time.Parse("15:04", c.NotifyTime)
	if err != nil {
		return false
	}
	scheduled := time.Date(nowLoc.Year(), nowLoc.Month(), nowLoc.Day(), nt.Hour(), nt.Minute(), 0, 0, loc)
	return !nowLoc.Before(scheduled)
}

// computeDaysLeft returns the whole number of calendar days from today (in loc)
// to the target date. It can be negative when the target date is in the past.
func computeDaysLeft(targetDate string, loc *time.Location, now time.Time) (int, error) {
	target, err := time.ParseInLocation("2006-01-02", targetDate, loc)
	if err != nil {
		return 0, err
	}
	y, m, d := now.In(loc).Date()
	todayMidnight := time.Date(y, m, d, 0, 0, 0, 0, loc)
	// math.Round absorbs the 23/25-hour days that occur across DST transitions.
	return int(math.Round(target.Sub(todayMidnight).Hours() / 24)), nil
}
