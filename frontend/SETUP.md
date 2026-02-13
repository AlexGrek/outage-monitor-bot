# Frontend Setup Complete ✅

## What Was Built

A modern, responsive dashboard for the Outage Monitor Bot with:

- **Real-time monitoring** - Auto-refreshes every 5 seconds
- **Health status** - Visual indicators (🟢 Healthy, 🟡 Degraded, 🔴 Unhealthy)
- **Configuration management** - Edit all settings via UI
- **Auto-restart monitoring** - View backoff delays and retry attempts
- **API key authentication** - Secure access with localStorage persistence
- **Responsive design** - Mobile, tablet, and desktop layouts

## Tech Stack

- React 19 + TypeScript
- Vite 8
- Tailwind CSS 3
- Untitled UI components
- React Aria Components

## Quick Start

```bash
# Install dependencies (already done)
npm install

# Start dev server
npm run dev

# Open http://localhost:5173
# Enter your API key from backend .env (API_KEY variable)
```

## API Proxy

All `/api/*` requests are proxied to `http://localhost:8080` in dev mode.

Example:
- Frontend: `fetch('/api/health')`
- Proxied to: `http://localhost:8080/health`

## File Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── dashboard/
│   │   │   ├── HealthBadge.tsx      # Status indicator
│   │   │   ├── StatusCard.tsx       # Metric cards
│   │   │   ├── ConfigPanel.tsx      # Config editor
│   │   │   ├── AutoRestartInfo.tsx  # Restart status
│   │   │   └── ApiKeyModal.tsx      # Auth modal
│   │   ├── base/                    # Untitled UI components
│   │   ├── application/
│   │   └── foundations/
│   ├── lib/
│   │   └── api.ts                   # API client
│   ├── types/
│   │   └── index.ts                 # TypeScript types
│   ├── App.tsx                      # Main dashboard
│   ├── main.tsx                     # Entry point
│   └── index.css                    # Tailwind + styles
├── tailwind.config.js               # Tailwind configuration
├── vite.config.ts                   # Vite config with proxy
└── tsconfig.app.json                # TypeScript config
```

## Key Features

### 1. Health Badge
- Healthy (green) - All systems operational
- Degraded (yellow) - Bot not running
- Unhealthy (red) - Bot running with errors

### 2. Status Cards
- System Uptime
- Bot Status (Running/Stopped)
- Active Sources count
- API Status

### 3. Configuration Panel
- View all config settings
- Inline editing
- Auto-saves to database
- Triggers bot restart automatically
- Masks sensitive values (TELEGRAM_TOKEN, API_KEY)

### 4. Auto-Restart Info
- Shows if auto-restart is enabled
- Current attempt count
- Next restart delay
- Timer status

### 5. Bot Details Sidebar
- Start time
- Uptime
- Last error (if any)
- System information

## Build & Deploy

```bash
# Production build
npm run build

# Preview production build
npm run preview

# Output in dist/ directory
```

## Environment Setup

No frontend environment variables needed! The API proxy is configured in `vite.config.ts` for development.

For production, serve the `dist/` folder and configure your reverse proxy to forward `/api` requests to your backend server.

## Development Tips

1. **Hot reload is enabled** - Changes auto-refresh
2. **API key is stored in localStorage** - Persists between refreshes
3. **Auto-refresh every 5 seconds** - Real-time updates
4. **TypeScript strict mode** - Type safety enforced
5. **Path aliases** - Use `@/` for imports (e.g., `@/utils/cx`)

## Troubleshooting

### Backend not connecting
- Ensure backend is running on `http://localhost:8080`
- Check browser console for errors
- Verify API_ENABLED=true in backend .env

### API key errors
- Get API_KEY from backend .env file
- Clear localStorage: `localStorage.removeItem('api_key')`
- Use "Change API Key" button in UI

### Styles not loading
- Check that tailwindcss is installed: `npm list tailwindcss`
- Verify postcss.config.js exists
- Clear node_modules and reinstall if needed

## Next Steps

The dashboard is production-ready! Just:

1. Start backend: `make run` (from root directory)
2. Start frontend: `npm run dev` (from frontend directory)
3. Open browser: http://localhost:5173
4. Enter API key from backend .env
5. Monitor and manage your bot! 🚀
