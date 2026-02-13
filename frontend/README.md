# Outage Monitor Bot - Dashboard

A modern, responsive web dashboard for managing and monitoring the Outage Monitor Bot.

## Features

- **Real-time Health Monitoring**: Live status updates for bot, API, and system health
- **Configuration Management**: Edit all bot configuration via web interface
- **Auto-Restart Visibility**: Monitor automatic restart attempts and backoff delays
- **Secure API Key Management**: Local storage of API key with X-API-Key header authentication
- **Responsive Design**: Works seamlessly on desktop, tablet, and mobile devices
- **Auto-Refresh**: Dashboard data updates every 5 seconds

## Tech Stack

- **React 19** with TypeScript
- **Vite 8** for fast development and builds
- **Tailwind CSS** for styling
- **Untitled UI** design system
- **React Aria Components** for accessibility

## Development

### Prerequisites

- Node.js 18+ and npm
- Backend API running on http://localhost:8080

### Setup

1. Install dependencies:
   ```bash
   npm install
   ```

2. Start development server:
   ```bash
   npm run dev
   ```

3. Open http://localhost:5173 in your browser

4. Enter your API key (from backend `.env` file, `API_KEY` variable)

### Available Scripts

- `npm run dev` - Start development server with hot reload
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint

## API Proxy

In development mode, all `/api/*` requests are automatically proxied to `http://localhost:8080`. This is configured in `vite.config.ts`:

```typescript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
      rewrite: (path) => path.replace(/^\/api/, ''),
    },
  },
}
```

## Configuration

### Backend Connection

The dashboard connects to the backend API using these endpoints:

- `GET /health` - Health check (no auth)
- `GET /status` - Detailed status (requires API key)
- `GET /config` - List all config (requires API key)
- `PUT /config/:key` - Update config (requires API key)
- `POST /config/reload` - Reload bot (requires API key)

### API Key Storage

API keys are stored in browser localStorage under the `api_key` key. To clear:

```javascript
localStorage.removeItem('api_key')
```

Or use the "Change API Key" button in the dashboard.

## Dashboard Features

### Health Status Badge

Shows overall system health with color-coded indicators:
- 🟢 **Healthy** - All systems operational
- 🟡 **Degraded** - Bot not running
- 🔴 **Unhealthy** - Bot running with errors

### Status Cards

Four main metrics displayed at the top:
1. **System Uptime** - Total application runtime
2. **Bot Status** - Running/stopped with health indicator
3. **Active Sources** - Currently monitored sources
4. **API Status** - REST API availability

### Configuration Panel

- View all configuration settings
- Edit any config value inline
- Auto-saves to database
- Triggers automatic bot restart on changes
- Sensitive values (TELEGRAM_TOKEN, API_KEY) are masked

### Auto-Restart Info

Shows current auto-restart state:
- Enabled/disabled status
- Current attempt count vs max attempts
- Next restart delay (with exponential backoff)
- Timer active indicator

### Bot Details

- Start time and uptime
- Last error message (if any)
- Configuration snapshot

## Responsive Design

The dashboard uses Tailwind CSS breakpoints for adaptive layouts:

- **Mobile** (< 768px): Single column, stacked cards
- **Tablet** (768px - 1024px): Two column grid
- **Desktop** (> 1024px): Full three-column layout

## Building for Production

```bash
npm run build
```

Output will be in the `dist/` directory. Serve with any static file server:

```bash
npm run preview
```

Or use with nginx/Apache/Caddy as a static site.

### Production Environment

For production deployment, you'll need to configure the API proxy differently (e.g., using nginx reverse proxy) or set the API base URL to your backend server.

## Troubleshooting

### "Invalid API key" error

- Check that your API key matches the backend's `API_KEY` environment variable
- Clear localStorage and re-enter the key
- Verify backend is running and accessible

### Connection errors

- Ensure backend is running on http://localhost:8080
- Check browser console for CORS errors
- Verify proxy configuration in `vite.config.ts`

### Styling not loading

- Run `npm install` to ensure Tailwind CSS is installed
- Check that `tailwind.config.js` and `postcss.config.js` exist
- Verify `@tailwind` directives are in `src/index.css`
