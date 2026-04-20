.PHONY: all ui harness worker test clean run

all: worker harness

ui:
	cd ui && npm install --no-audit --no-fund && npm run build

harness: ui
	go build -o bin/harness ./cmd/harness

worker:
	go build -o bin/worker-go ./cmd/worker-go

test:
	go test ./...

run: all
	./bin/harness --worker ./bin/worker-go --db ./loadtest.db --port 9010

clean:
	rm -rf bin internal/uiembed/dist/assets ui/dist loadtest.db
