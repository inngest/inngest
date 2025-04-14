#!/bin/bash
sh -c 'cd ./tests/js && pnpm install > /dev/null 2> /dev/null'
sh -c 'cd ./tests/js && pnpm dev > /dev/null 2> /dev/null' &

sleep 2

export ENABLE_TEST_API=true

go run ./cmd/main.go dev --no-discovery > dev-stdout.txt 2> dev-stderr.txt &

sleep 5

# Run JS SDK tests
INNGEST_SIGNING_KEY=test API_URL=http://127.0.0.1:8288 SDK_URL=http://127.0.0.1:3000/api/inngest go test ./tests -v -count=1

# Run Golang SDK tests
go test ./tests/golang -v -count=1
