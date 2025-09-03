
# Check if temporal CLI is installed, otherwise install
install-temporal:
	@if ! command -v temporal >/dev/null 2>&1; then \
		echo "⚡ Installing Temporal CLI via Homebrew..."; \
		brew install temporal; \
	else \
		echo "✅ Temporal CLI already installed"; \
	fi

# Start temporal dev server in background
start: install-temporal
	@echo "🚀 Starting Temporal dev server..."
	@nohup temporal server start-dev > temporal.log 2>&1 & echo $$! > temporal.pid
	@sleep 5
	@echo "✅ Temporal server started (logs in temporal.log, PID=$$(cat temporal.pid))"

# Run docker-compose (workers + api)
up: start
	docker-compose up

# Kill temporal + remove all containers
kill:
	docker-compose down
