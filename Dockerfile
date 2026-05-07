FROM --platform=$BUILDPLATFORM golang:1.26-alpine@sha256:e58f92c51f4cc98f2ef4780ba8708fe914f20e6175720a19c1c85ff029a6b9fc AS build
RUN apk upgrade --no-cache && apk add --no-cache build-base
WORKDIR /app
COPY vendor vendor
COPY . .
ARG TARGETARCH
ARG TARGETOS
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOFLAGS=-mod=vendor GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /go/bin/inngest cmd/main.go

FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS inngest
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata && update-ca-certificates
RUN addgroup -g 1000 -S inngest && adduser -u 1000 -S -G inngest -s /sbin/nologin inngest
COPY --from=build /go/bin/inngest /bin/inngest
USER inngest
CMD ["inngest"]
