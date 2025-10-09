#!/bin/bash

# Kill any existing processes on port 8288
echo "Checking for existing processes on port 8288..."
if lsof -ti:8288 >/dev/null 2>&1; then
    echo "WARNING: Killing existing processes on port 8288"
    lsof -ti:8288 | xargs kill -9 2>/dev/null || true
else
    echo "No processes found on port 8288"
fi

# Store background process PIDs
pids=()

# Function to kill all background processes
cleanup() {
    echo "Cleaning up background processes..."
    # Print tracked PIDs before cleanup
    echo "Background process PIDs: ${pids[*]}"

    for pid in "${pids[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            # Kill entire process tree
            # technically we might need to recurse and kill all the PIDs but killing the results of
            # pgrep -P seems to at least leave no stranglers on port 3000 and 8288
            children=$(pgrep -P "$pid" 2>/dev/null || true)
            if [ -n "$children" ]; then
                echo "Killing child PIDs: $children"
                pkill -P "$pid" 2>/dev/null || true
            fi
            echo "Killing parent PID: $pid"
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
    wait
    exit 0
}

# Trap SIGINT and SIGTERM to cleanup
trap cleanup SIGINT SIGTERM

# Accept optional test pattern argument
TEST_PATTERN="${1:-}"

sh -c 'cd ./tests/js && pnpm install --frozen-lockfile --prod > /dev/null 2> /dev/null'
sh -c 'cd ./tests/js && pnpm dev > /dev/null 2> /dev/null' &
pids+=($!)

sleep 2

export TEST_MODE=true

go run ./cmd dev --no-discovery > dev-stdout.txt 2> dev-stderr.txt &
pids+=($!)

# Check that dev server started successfully
echo "Waiting for dev server to start..."
max_attempts=10
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if curl -s -f http://127.0.0.1:8288/health > /dev/null 2>&1; then
        echo "Dev server started successfully"
        break
    fi
    echo "Attempt $((attempt + 1))/$max_attempts: Health check failed, retrying..."
    if [ $attempt -eq $((max_attempts - 1)) ]; then
        echo "ERROR: Dev server failed to start after 5 seconds"
        echo "Dev server stdout:"
        cat dev-stdout.txt
        echo "Dev server stderr:"
        cat dev-stderr.txt
        cleanup
    fi
    sleep 1
    attempt=$((attempt + 1))
done

# Run JS SDK tests
if [ -n "$TEST_PATTERN" ]; then
    INNGEST_SIGNING_KEY=test API_URL=http://127.0.0.1:8288 SDK_URL=http://127.0.0.1:3000/api/inngest go test ./tests -v -count=1 -run "$TEST_PATTERN"
else
    INNGEST_SIGNING_KEY=test API_URL=http://127.0.0.1:8288 SDK_URL=http://127.0.0.1:3000/api/inngest go test ./tests -v -count=1
fi

# Run Golang SDK tests
if [ -n "$TEST_PATTERN" ]; then
    go test ./tests/golang -v -count=1 -run "$TEST_PATTERN"
else
    go test ./tests/golang -v -count=1
fi

# Cleanup at the end
cleanup
