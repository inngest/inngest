# Testing

### E2E testing, local builds

1. `go run ./cmd/main.go dev --no-discovery`: Run the dev server using your local build: 
2. `cd tests/js && yarn dev`: Run the JS SDK
3. `INNGEST_SIGNING_KEY=test API_URL=http://127.0.0.1:8288 SDK_URL=http://127.0.0.1:3000/api/inngest go test ./tests -v -count=1`: Run tests

To filter tests:
`INNGEST_SIGNING_KEY=test API_URL=http://127.0.0.1:8288 SDK_URL=http://127.0.0.1:3000/api/inngest go test ./tests -v -count=1 -test.run TestSDKCancelNotReceived`
