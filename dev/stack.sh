#!/usr/bin/env bash
# Start/stop local dev: Go API (air) + Vite frontend; optional Docker Mongo.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEV_DIR="$ROOT/.dev"
PID_DIR="$DEV_DIR/pids"
LOG_DIR="$DEV_DIR/logs"
GO_PID="$PID_DIR/go.pid"
FRONT_PID="$PID_DIR/front.pid"
MONGO_FLAG="$PID_DIR/mongo.started"

mkdir -p "$PID_DIR" "$LOG_DIR"

is_running() {
	local pidfile="$1"
	[[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null
}

sqlite_path() {
	local path="../db/comics.db"
	if [[ -f "$ROOT/go_server/.env" ]]; then
		local from_env
		from_env="$(grep -E '^COMICS_SQLITE_PATH=' "$ROOT/go_server/.env" | head -1 | cut -d= -f2- || true)"
		if [[ -n "$from_env" ]]; then
			if [[ "$from_env" = /* ]]; then
				path="$from_env"
			else
				path="$ROOT/go_server/$from_env"
			fi
		fi
	fi
	echo "$path"
}

checkpoint_sqlite_wal() {
	local db_path
	db_path="$(sqlite_path)"
	if [[ ! -f "$db_path" ]]; then
		return 0
	fi
	local attempt=0
	local max=30
	while [[ $attempt -lt $max ]]; do
		attempt=$((attempt + 1))
		if pgrep -f "$ROOT/go_server/tmp/main server" >/dev/null 2>&1; then
			echo "Waiting for Go API to exit (attempt $attempt/$max)..."
			sleep 2
			continue
		fi
		if lsof "$db_path" >/dev/null 2>&1; then
			echo "Waiting for release of $db_path (attempt $attempt/$max)..."
			sleep 2
			continue
		fi
		echo "Checkpointing SQLite WAL (TRUNCATE): $db_path"
		local result
		result="$(sqlite3 "$db_path" "PRAGMA wal_checkpoint(TRUNCATE);")"
		echo "PRAGMA wal_checkpoint: $result"
		local busy="${result%%|*}"
		if [[ "$busy" == "0" ]]; then
			ls -lh "$db_path" "${db_path}-wal" 2>/dev/null || ls -lh "$db_path"
			if command -v git >/dev/null 2>&1 && git -C "$ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
				head_bytes="$(git -C "$ROOT" show "HEAD:${db_path#$ROOT/}" 2>/dev/null | wc -c | tr -d ' ')"
				work_bytes="$(wc -c < "$db_path" | tr -d ' ')"
				echo "comics.db vs git HEAD: $head_bytes -> $work_bytes bytes"
				git -C "$ROOT" status --short "${db_path#$ROOT/}" || true
			fi
			return 0
		fi
		sleep 2
	done
	echo "WAL checkpoint did not complete (database busy)."
	return 1
}

stop_go_api() {
	local pattern="$ROOT/go_server/tmp/main server"
	local db_path
	db_path="$(sqlite_path)"

	if pgrep -f "$pattern" >/dev/null 2>&1; then
		echo "Stopping Go API process (SIGTERM, allows WAL checkpoint on exit)..."
		pkill -TERM -f "$pattern" 2>/dev/null || true
	fi

	if [[ -f "$db_path" ]]; then
		local pids=""
		pids="$(lsof -t "$db_path" 2>/dev/null || true)"
		if [[ -n "$pids" ]]; then
			echo "Sending SIGTERM to processes holding $(basename "$db_path")..."
			# shellcheck disable=SC2086
			kill -TERM $pids 2>/dev/null || true
		fi
	fi

	local i=0
	while { pgrep -f "$pattern" >/dev/null 2>&1 || { [[ -f "$db_path" ]] && lsof -t "$db_path" >/dev/null 2>&1; }; } && [[ $i -lt 3000 ]]; do
		sleep 0.2
		i=$((i + 1))
	done
}

stop_pidfile() {
	local pidfile="$1"
	local label="$2"
	if ! [[ -f "$pidfile" ]]; then
		return 0
	fi
	local pid
	pid="$(cat "$pidfile")"
	if kill -0 "$pid" 2>/dev/null; then
		kill -TERM "$pid" 2>/dev/null || true
		local i=0
		while kill -0 "$pid" 2>/dev/null && [[ $i -lt 150 ]]; do
			sleep 0.2
			i=$((i + 1))
		done
		if kill -0 "$pid" 2>/dev/null; then
			kill -9 "$pid" 2>/dev/null || true
			pkill -P "$pid" 2>/dev/null || true
		fi
	fi
	rm -f "$pidfile"
	echo "Stopped $label (pid $pid)"
}

ensure_go_env() {
	if [[ ! -f "$ROOT/go_server/.env" ]]; then
		cp "$ROOT/go_server/local.env" "$ROOT/go_server/.env"
		echo "Created go_server/.env from local.env (set DB_PASS for Atlas if needed)."
	fi
}

maybe_start_mongo() {
	local db_addr=""
	if [[ -f "$ROOT/go_server/.env" ]]; then
		db_addr="$(grep -E '^DB_ADDR=' "$ROOT/go_server/.env" | head -1 | cut -d= -f2- || true)"
	fi
	case "$db_addr" in
		mongodb://localhost*|mongodb://127.0.0.1*|mongodb://mongodb:*)
			if ! command -v docker >/dev/null 2>&1; then
				echo "Warning: local Mongo in .env but Docker is not available."
				return 0
			fi
			make -C "$ROOT/go_server" mongo-up
			touch "$MONGO_FLAG"
			echo "MongoDB (Docker) started on localhost:27017"
			;;
		*)
			echo "Using remote Mongo ($db_addr); skipping local mongo-up."
			;;
	esac
}

maybe_stop_mongo() {
	if [[ -f "$MONGO_FLAG" ]]; then
		make -C "$ROOT/go_server" mongo-down 2>/dev/null || true
		rm -f "$MONGO_FLAG"
		echo "Stopped local MongoDB container."
	fi
}

cmd_up() {
	if is_running "$GO_PID" || is_running "$FRONT_PID"; then
		echo "Dev stack already running. Run: make dev-down"
		cmd_status
		exit 1
	fi

	ensure_go_env
	maybe_start_mongo

	echo "Starting Go API (air)..."
	(
		cd "$ROOT/go_server"
		exec go tool -modfile=tool.go.mod air -c .air.plain.toml server
	) >"$LOG_DIR/go.log" 2>&1 &
	echo $! >"$GO_PID"

	echo "Starting Vite frontend..."
	(
		cd "$ROOT"
		exec npm run start
	) >"$LOG_DIR/front.log" 2>&1 &
	echo $! >"$FRONT_PID"

	echo "Waiting for listeners..."
	sleep 4
	cmd_status
	echo ""
	echo "Logs: make dev-logs   Stop: make dev-down"
}

cmd_down() {
	stop_pidfile "$FRONT_PID" "frontend"
	stop_go_api
	stop_pidfile "$GO_PID" "Go API (air)"
	pkill -TERM -f "$ROOT/go_server/tmp/main server" 2>/dev/null || true
	sleep 1
	pkill -9 -f "$ROOT/go_server/tmp/main server" 2>/dev/null || true
	checkpoint_sqlite_wal || true
	maybe_stop_mongo
	echo "Dev stack stopped."
}

cmd_status() {
	local go_ok=0 front_ok=0
	if is_running "$GO_PID"; then
		go_ok=1
	fi
	if is_running "$FRONT_PID"; then
		front_ok=1
	fi

	echo "Dev stack:"
	if [[ $go_ok -eq 1 ]]; then
		echo "  Go API     running (pid $(cat "$GO_PID"), log $LOG_DIR/go.log)"
	else
		echo "  Go API     not running"
	fi
	if [[ $front_ok -eq 1 ]]; then
		echo "  Frontend   running (pid $(cat "$FRONT_PID"), log $LOG_DIR/front.log)"
	else
		echo "  Frontend   not running"
	fi
	if [[ -f "$MONGO_FLAG" ]]; then
		echo "  MongoDB    Docker (started by dev-up)"
	fi
	echo ""
	echo "  App        http://localhost:3000/comics/"
	echo "  API        https://localhost:8081 (or http if no TLS certs)"
	if curl -sk -o /dev/null -w "" https://localhost:8081/health 2>/dev/null; then
		echo "  Health     https://localhost:8081/health OK"
	elif curl -s -o /dev/null -w "" http://localhost:8081/health 2>/dev/null; then
		echo "  Health     http://localhost:8081/health OK"
	fi
}

cmd_logs() {
	tail -f "$LOG_DIR/go.log" "$LOG_DIR/front.log"
}

usage() {
	echo "Usage: $0 {up|down|status|logs}"
}

case "${1:-}" in
	up) cmd_up ;;
	down) cmd_down ;;
	status) cmd_status ;;
	logs) cmd_logs ;;
	*) usage; exit 1 ;;
esac
