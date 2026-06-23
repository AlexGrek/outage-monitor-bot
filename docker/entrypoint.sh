#!/bin/sh
# Runs the Go backend and nginx side by side without a process supervisor.
# If either process exits, both are stopped and the container exits non-zero so
# the orchestrator (Kubernetes / Docker restart policy) restarts the pod.
#
# This intentionally avoids supervisord: it added a fragile Python dependency
# that broke on a newer Alpine base (Python 3.12), and two long-lived processes
# do not need a full process manager.
set -eu

BACKEND_PID=""
NGINX_PID=""

term() {
    [ -n "$BACKEND_PID" ] && kill -TERM "$BACKEND_PID" 2>/dev/null || true
    [ -n "$NGINX_PID" ] && kill -TERM "$NGINX_PID" 2>/dev/null || true
}
trap term TERM INT

# Run from /app so the default relative DB_PATH (data/state.db) resolves to
# /app/data, which is created and owned by appuser in the image.
cd /app

# Backend (REST API + monitor + bot) on :8080
/app/tg-monitor-bot &
BACKEND_PID=$!

# nginx serves the frontend on :80 and proxies /api + /health + /webhooks to :8080
nginx -g 'daemon off;' &
NGINX_PID=$!

# Exit as soon as either child dies.
while kill -0 "$BACKEND_PID" 2>/dev/null && kill -0 "$NGINX_PID" 2>/dev/null; do
    sleep 1
done

echo "entrypoint: a managed process exited, shutting down container" >&2
term
exit 1
