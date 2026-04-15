FROM --platform=$BUILDPLATFORM golang:1.26-alpine@sha256:1fb7391fd54a953f15205f2cfe71ba48ad358c381d4e2efcd820bfca921cd6c6 AS build
RUN apk upgrade --no-cache && apk add --no-cache build-base
WORKDIR /app
COPY vendor vendor
COPY . .
ARG TARGETARCH
ARG TARGETOS
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOFLAGS=-mod=vendor GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /go/bin/inngest cmd/main.go

FROM alpine:3.23@sha256:c69a6ff7c24d1ffa913798501d0e7104e0e9764e28eb44a930939f91ef829e64 AS inngest
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata && update-ca-certificates
RUN addgroup -g 1000 -S inngest && adduser -u 1000 -S -G inngest -s /sbin/nologin inngest
COPY --from=build /go/bin/inngest /bin/inngest
USER inngest
CMD ["inngest"]
