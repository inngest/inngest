build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./target/inngestctl-linux-x64 ./cmd
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./target/inngestctl-mac-x64 ./cmd
