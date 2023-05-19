#!/bin/bash
sh -c 'cd ./tests/js && yarn install > /dev/null 2> /dev/null'
sh -c 'cd ./tests/js && yarn dev > /dev/null 2> /dev/null' &

sleep 2

go run ./cmd/main.go dev --no-discovery > dev-stdout.txt 2> dev-stderr.txt &

sleep 2

INNGEST_SIGNING_KEY=test API_URL=http://127.0.0.1:8288 SDK_URL=http://127.0.0.1:3000/api/inngest go test ./tests -v -count=1
