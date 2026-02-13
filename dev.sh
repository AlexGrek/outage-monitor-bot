#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Outage Monitor Bot - Development Environment${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Auto-create .env from .env.example if it doesn't exist
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠ .env not found, creating from .env.example...${NC}"
    if [ -f .env.example ]; then
        cp .env.example .env

        # Generate API key automatically
        API_KEY_GENERATED=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p)

        # Replace placeholder API key with generated one
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            sed -i '' "s/your-secret-api-key-here/$API_KEY_GENERATED/" .env
        else
            # Linux
            sed -i "s/your-secret-api-key-here/$API_KEY_GENERATED/" .env
        fi

        echo -e "${GREEN}✓ Created .env with auto-generated API key${NC}"
        echo -e "${YELLOW}  ⚠ IMPORTANT: Set TELEGRAM_TOKEN in .env before continuing!${NC}"
        echo -e "${YELLOW}     Get it from: ${GREEN}https://t.me/BotFather${NC}"
        echo ""
        exit 1
    else
        echo -e "${RED}✗ Error: .env.example not found${NC}"
        exit 1
    fi
fi

# Load .env file
export $(cat .env | grep -v '^#' | xargs)

# Check if TELEGRAM_TOKEN is set
if [ -z "$TELEGRAM_TOKEN" ] || [ "$TELEGRAM_TOKEN" = "your_bot_token_here" ]; then
    echo -e "${YELLOW}⚠ TELEGRAM_TOKEN not configured - running in web-only mode${NC}"
    echo -e "${BLUE}  The Telegram bot will be disabled.${NC}"
    echo -e "${BLUE}  You can still use the web dashboard and REST API to manage sources.${NC}"
    echo -e "${YELLOW}  To enable Telegram notifications, get a token from: ${GREEN}https://t.me/BotFather${NC}"
    echo ""
fi

# Check if API_KEY is set (regenerate if needed)
if [ -z "$API_KEY" ] || [ "$API_KEY" = "your-secret-api-key-here" ]; then
    echo -e "${YELLOW}⚠ API_KEY not set, generating new one...${NC}"
    API_KEY_GENERATED=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p)

    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/API_KEY=.*/API_KEY=$API_KEY_GENERATED/" .env
    else
        sed -i "s/API_KEY=.*/API_KEY=$API_KEY_GENERATED/" .env
    fi

    # Reload env
    export API_KEY=$API_KEY_GENERATED
    echo -e "${GREEN}✓ Generated new API key${NC}"
fi

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}Shutting down...${NC}"

    # Kill backend
    if [ ! -z "$BACKEND_PID" ]; then
        echo -e "${BLUE}  Stopping backend (PID: $BACKEND_PID)...${NC}"
        kill $BACKEND_PID 2>/dev/null
    fi

    # Kill frontend
    if [ ! -z "$FRONTEND_PID" ]; then
        echo -e "${BLUE}  Stopping frontend (PID: $FRONTEND_PID)...${NC}"
        kill $FRONTEND_PID 2>/dev/null
    fi

    # Kill tail process
    if [ ! -z "$TAIL_PID" ]; then
        kill $TAIL_PID 2>/dev/null
    fi

    echo -e "${GREEN}✓ Shutdown complete${NC}"
    exit 0
}

# Set trap for cleanup on Ctrl+C
trap cleanup SIGINT SIGTERM

# Kill any existing processes
echo -e "${YELLOW}Checking for existing processes...${NC}"

# Kill backend processes
EXISTING_BACKEND=$(pgrep -f "bin/tg-monitor-bot" || true)
if [ ! -z "$EXISTING_BACKEND" ]; then
    echo -e "${YELLOW}⚠ Found existing backend processes (PIDs: $EXISTING_BACKEND)${NC}"
    pkill -f "bin/tg-monitor-bot" || true
fi

# Kill orphaned tail processes
EXISTING_TAIL=$(pgrep -f "tail -f logs/backend.log" || true)
if [ ! -z "$EXISTING_TAIL" ]; then
    echo -e "${YELLOW}⚠ Found orphaned tail processes (PIDs: $EXISTING_TAIL)${NC}"
    pkill -f "tail -f logs/backend.log" || true
fi

if [ -z "$EXISTING_BACKEND" ] && [ -z "$EXISTING_TAIL" ]; then
    echo -e "${GREEN}✓ No existing processes found${NC}"
else
    sleep 1
    echo -e "${GREEN}✓ Existing processes stopped${NC}"
fi
echo ""

# Create data directory if it doesn't exist
mkdir -p data

# Build backend
echo -e "${BLUE}Building backend...${NC}"
make build > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Backend build failed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Backend built${NC}"

# Set capabilities for ICMP if needed
if ! getcap bin/tg-monitor-bot | grep -q cap_net_raw; then
    echo -e "${YELLOW}⚠ Setting ICMP capabilities (requires sudo)...${NC}"
    sudo make setcap > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ ICMP capabilities set${NC}"
    else
        echo -e "${YELLOW}  Warning: Could not set ICMP capabilities. Ping checks may fail.${NC}"
    fi
fi

# Start backend
echo -e "${BLUE}Starting backend on port $API_PORT...${NC}"
./bin/tg-monitor-bot > logs/backend.log 2>&1 &
BACKEND_PID=$!

# Wait for backend to start
sleep 2

# Check if backend is running
if ! kill -0 $BACKEND_PID 2>/dev/null; then
    echo -e "${RED}✗ Backend failed to start${NC}"
    echo -e "${YELLOW}  Check logs/backend.log for details${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Backend started (PID: $BACKEND_PID)${NC}"

# Install frontend dependencies if needed
if [ ! -d "frontend/node_modules" ]; then
    echo -e "${BLUE}Installing frontend dependencies...${NC}"
    cd frontend && npm install > /dev/null 2>&1
    cd ..
    echo -e "${GREEN}✓ Frontend dependencies installed${NC}"
fi

# Start frontend
echo -e "${BLUE}Starting frontend on port 5173...${NC}"
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

sleep 2
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  ✓ Development environment ready!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}  Backend API:${NC}    http://localhost:${API_PORT}"
echo -e "${BLUE}  Frontend:${NC}       http://localhost:5173"
echo -e "${BLUE}  API Key:${NC}        ${API_KEY:0:8}...${API_KEY: -4}"
echo ""
echo -e "${YELLOW}  Press Ctrl+C to stop both servers${NC}"
echo ""

# Follow backend logs
tail -f logs/backend.log &
TAIL_PID=$!

# Wait for interrupt
wait $FRONTEND_PID
