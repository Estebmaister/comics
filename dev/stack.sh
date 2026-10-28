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

# Max seconds to wait for API exit (override for slow WAL shutdown on large DBs).
DEV_SHUTDOWN_SEC="${DEV_SHUTDOWN_SEC:-90}"
# Max seconds for offline PRAGMA wal_checkpoint (0 = skip).
DEV_CHECKPOINT_SEC="${DEV_CHECKPOINT_SEC:-180}"

mkdir -p "$PID_DIR" "$LOG_DIR"

is_running() {
	local pidfile="$1"
	[[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null
}

sqlite_path() {
	local path="$ROOT/db/comics.db"
	if [[ -f "$ROOT/go_server/.env" ]]; then
		local from_env
		from_env="$(grep -E '^COMICS_SQLITE_PATH=' "$ROOT/go_server/.env" | head -1 | cut -d= -f2- || true)"
		if [[ -n "$from_env" ]]; then
			if [[ "$from_env" = /* ]]; then
				path="$from_env"
			else
				path="$(cd "$ROOT/go_server" && cd "$(dirname "$from_env")" && pwd)/$(basename "$from_env")"
			fi
		fi
	fi
	echo "$path"
}

go_api_pattern() {
	echo "$ROOT/go_server/tmp/main server"
}

go_api_running() {
	pgrep -f "$(go_api_pattern)" >/dev/null 2>&1
}

echo_progress() {
	local msg="$1"
	printf '%s\n' "$msg" >&2
}

wait_until() {
	local desc="$1"
	local max_sec="$2"
	local check_fn="$3"
	local elapsed=0
	while "$check_fn"; do
		if [[ $elapsed -ge $max_sec ]]; then
			echo_progress "$desc: timed out after ${max_sec}s"
			return 1
		fi
		if (( elapsed > 0 && elapsed % 10 == 0 )); then
			echo_progress "$desc... ${elapsed}s"
		fi
		sleep 1
		elapsed=$((elapsed + 1))
	done
	return 0
}

still_running_go_api() {
	go_api_running
}

run_sqlite_checkpoint() {
	local db_path="$1"
	local max_sec="$2"
	if [[ ! -f "$db_path" ]] || [[ "$max_sec" -le 0 ]]; then
		return 0
	fi
	if go_api_running; then
		echo_progress "Skipping WAL checkpoint: Go API still running."
		return 1
	fi
	echo_progress "Checkpointing SQLite WAL (TRUNCATE, max ${max_sec}s): $db_path"
	(
		sqlite3 "$db_path" "PRAGMA wal_checkpoint(TRUNCATE);"
	) &
	local cp_pid=$!
	local elapsed=0
	while kill -0 "$cp_pid" 2>/dev/null; do
		if [[ $elapsed -ge $max_sec ]]; then
			kill -TERM "$cp_pid" 2>/dev/null || true
			wait "$cp_pid" 2>/dev/null || true
			echo_progress "WAL checkpoint timed out after ${max_sec}s (run: make db-checkpoint-wal)"
			return 1
		fi
		if (( elapsed > 0 && elapsed % 15 == 0 )); then
			echo_progress "WAL checkpoint still running... ${elapsed}s"
		fi
		sleep 1
		elapsed=$((elapsed + 1))
	done
	wait "$cp_pid"
	local result
	result="$(sqlite3 "$db_path" "PRAGMA wal_checkpoint(PASSIVE);" 2>/dev/null || echo "?|?|?")"
	echo_progress "PRAGMA wal_checkpoint (passive): $result"
	ls -lh "$db_path" "${db_path}-wal" 2>/dev/null || ls -lh "$db_path"
}

checkpoint_sqlite_wal() {
	local db_path
	db_path="$(sqlite_path)"
	run_sqlite_checkpoint "$db_path" "$DEV_CHECKPOINT_SEC"
}

stop_go_api() {
	local pattern
	pattern="$(go_api_pattern)"
	if go_api_running; then
		echo_progress "Stopping Go API (SIGTERM)..."
		pkill -TERM -f "$pattern" 2>/dev/null || true
	fi
	if ! wait_until "Waiting for Go API to exit" "$DEV_SHUTDOWN_SEC" still_running_go_api; then
		echo_progress "Force-stopping Go API (SIGKILL)..."
		pkill -9 -f "$pattern" 2>/dev/null || true
		sleep 1
	fi
}

stop_pidfile() {
	local pidfile="$1"
	local label="$2"
	local max_sec="${3:-30}"
	if ! [[ -f "$pidfile" ]]; then
		return 0
	fi
	local pid
	pid="$(cat "$pidfile")"
	if kill -0 "$pid" 2>/dev/null; then
		kill -TERM "$pid" 2>/dev/null || true
		local elapsed=0
		while kill -0 "$pid" 2>/dev/null && [[ $elapsed -lt $max_sec ]]; do
			sleep 1
			elapsed=$((elapsed + 1))
		done
		if kill -0 "$pid" 2>/dev/null; then
			kill -9 "$pid" 2>/dev/null || true
			pkill -P "$pid" 2>/dev/null || true
		fi
	fi
	rm -f "$pidfile"
	echo_progress "Stopped $label (pid $pid)"
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
		echo_progress "Stopping local MongoDB container..."
		make -C "$ROOT/go_server" mongo-down 2>/dev/null || true
		rm -f "$MONGO_FLAG"
		echo_progress "Stopped local MongoDB container."
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
	echo_progress "Shutting down dev stack (shutdown ${DEV_SHUTDOWN_SEC}s, checkpoint ${DEV_CHECKPOINT_SEC}s)..."
	stop_pidfile "$FRONT_PID" "frontend" 20
	stop_pidfile "$GO_PID" "Go API (air)" 15
	stop_go_api
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
	if curl -sk -o /dev/null -w "" --max-time 3 https://localhost:8081/health 2>/dev/null; then
		echo "  Health     https://localhost:8081/health OK"
	elif curl -s -o /dev/null -w "" --max-time 3 http://localhost:8081/health 2>/dev/null; then
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
