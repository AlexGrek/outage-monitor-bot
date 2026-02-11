# Quick Start Guide

Get the Telegram Monitor Bot running in **under 5 minutes**! 🚀

## Prerequisites

- Go 1.24+
- Node.js 18+
- A Telegram bot token (get from [@BotFather](https://t.me/BotFather))

## One-Command Setup

```bash
# 1. Clone and enter the directory
cd tg-monitor-bot

# 2. Start development (auto-creates .env)
make dev
```

That's it! On first run, the script will:
- ✅ Auto-create `.env` from `.env.example`
- ✅ Auto-generate a secure API key
- ✅ Prompt you to set your TELEGRAM_TOKEN

### Optional Configuration

After the first run creates `.env`, the API key is **auto-generated** for you!

**To enable Telegram notifications** (optional):

```bash
# Edit .env and set your bot token from @BotFather
TELEGRAM_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
```

**Web-only mode**: Leave `TELEGRAM_TOKEN` empty to run without Telegram integration. You can still:
- ✅ Use the web dashboard
- ✅ Manage sources via REST API
- ✅ Monitor infrastructure
- ❌ No Telegram notifications

### Optional: Generate New API Key

```bash
make api-key  # Generates and updates .env automatically
```

## Start Development Environment

Run **one command** to start both backend and frontend:

```bash
make dev
```

This will:
- ✅ Build the Go backend
- ✅ Set ICMP capabilities (may ask for sudo password)
- ✅ Start backend API on http://localhost:8080
- ✅ Start frontend dashboard on http://localhost:5173
- ✅ Use the same API_KEY from .env for both

### What You'll See

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓ Development environment ready!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Backend API:    http://localhost:8080
  Frontend:       http://localhost:5173
  API Key:        a1b2c3d4...wxyz

  Press Ctrl+C to stop both servers
```

## Access the Dashboard

1. Open your browser to **http://localhost:5173**
2. Enter your API key (the one from `.env`)
3. Done! You can now:
   - ✅ View real-time bot health
   - ✅ Monitor active sources
   - ✅ Edit configuration live
   - ✅ Trigger bot restarts
   - ✅ View auto-restart status

## Add Your First Monitoring Source

In Telegram, send these commands to your bot:

```
# Start the bot
/start

# Add a source to monitor (example: ping google.com every 30 seconds)
/add_source Google ping 8.8.8.8 30s <your_chat_id>

# List all sources
/list
```

Get your chat ID:
1. Send any message to your bot
2. Visit: `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
3. Look for `"chat":{"id":123456789}` in the JSON response

## Alternative: Run Components Separately

### Backend Only

```bash
make run
```

### Frontend Only

```bash
cd frontend
npm install
npm run dev
```

## Stop Everything

Press **Ctrl+C** in the terminal running `make dev`. Both services will shut down gracefully.

## Common Issues

### "TELEGRAM_TOKEN environment variable is required"
- Make sure you copied `.env.example` to `.env`
- Add your bot token from @BotFather

### "API_KEY environment variable is required"
- Generate a key: `openssl rand -hex 32`
- Add it to `.env` as `API_KEY=<generated-key>`

### "Ping checks always fail"
- The script will try to set ICMP capabilities automatically
- If it fails, manually run: `sudo make setcap`

### "Port 8080 already in use"
- Change `API_PORT=8080` to another port in `.env`
- Restart with `make dev`

### "Cannot connect to backend"
- Make sure backend is running: `lsof -i :8080`
- Check logs: `tail -f logs/backend.log`

## What's Next?

- **Read the architecture**: Check `CLAUDE.md` for system design
- **Add sources**: Monitor your infrastructure via Telegram commands
- **Configure alerts**: Each source can notify multiple chat IDs
- **Customize settings**: All config editable via dashboard or API

## Production Deployment

For production, see:
- Docker deployment: `make docker-build && make docker-run-detached`
- Manual deployment: `make build-prod` and configure systemd
- Documentation: See `README.md` and `CLAUDE.md`

---

**Need help?** Check `CLAUDE.md` for detailed architecture and troubleshooting.
