FROM --platform=$BUILDPLATFORM golang:1.20.2-alpine AS build
RUN apk add build-base
WORKDIR /app
COPY vendor vendor
COPY . .
ARG TARGETARCH
ARG TARGETOS
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOFLAGS=-mod=vendor GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /go/bin/inngest cmd/main.go

FROM alpine:3.16 AS inngest
RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates
COPY --from=build /go/bin/inngest /bin/inngest
CMD ["inngest"]
