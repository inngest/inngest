FROM --platform=$BUILDPLATFORM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS build
RUN apk upgrade --no-cache && apk add --no-cache build-base
WORKDIR /app
COPY vendor vendor
COPY . .
ARG TARGETARCH
ARG TARGETOS
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOFLAGS=-mod=vendor GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /go/bin/inngest ./cmd/

FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS inngest
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata && update-ca-certificates
RUN addgroup -g 1000 -S inngest && adduser -u 1000 -S -G inngest -s /sbin/nologin inngest
COPY --from=build /go/bin/inngest /bin/inngest
USER inngest
CMD ["inngest"]
