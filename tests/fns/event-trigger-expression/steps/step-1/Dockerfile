FROM golang:1.18.2-alpine AS build
WORKDIR /app
COPY vendor vendor
COPY . .
ARG TARGETARCH
ARG TARGETOS
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOFLAGS=-mod=vendor GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /go/bin/main cmd/main.go


FROM alpine:3.15 AS step
RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates
COPY --from=build /go/bin/main /bin/main
RUN chmod +x /bin/*
ENTRYPOINT ["main"]
