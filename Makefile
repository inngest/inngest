build:
	GOOS=linux GOARCH=amd64 go build -o ./target/inngestctl-linux-x64 ./cmd
	GOOS=darwin GOARCH=amd64 go build -o ./target/inngestctl-mac-x64 ./cmd
