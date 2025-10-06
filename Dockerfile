# syntax=docker/dockerfile:1.7

# Etapa de build
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build

# Instala dependências básicas
RUN apk add --no-cache build-base ca-certificates tzdata && update-ca-certificates

# Define diretório de trabalho
WORKDIR /app

# Copia arquivos de módulo e dependências primeiro (melhor cache)
COPY go.mod go.sum ./
COPY vendor/ ./vendor/
RUN --mount=type=cache,target=/go/pkg/mod go mod download || true

# Copia o resto do código
COPY . .

# Variáveis de build
ARG TARGETOS
ARG TARGETARCH

# Compila o binário a partir do diretório cmd (onde está o main.go)
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=vendor -trimpath -buildvcs=false \
      -ldflags="-s -w" \
      -o /go/bin/inngest ./cmd

# Etapa final: imagem mínima
FROM alpine:3.20 AS inngest

RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates

# Cria usuário não-root
RUN addgroup -S app && adduser -S app -G app
USER app

# Copia o binário do estágio anterior
COPY --from=build /go/bin/inngest /bin/inngest

ENTRYPOINT ["/bin/inngest"]
