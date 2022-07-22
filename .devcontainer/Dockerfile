# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.241.1/containers/go/.devcontainer/base.Dockerfile

# [Choice] Go version (use -bullseye variants on local arm64/Apple Silicon): 1, 1.18, 1.17, 1-bullseye, 1.18-bullseye, 1.17-bullseye, 1-buster, 1.18-buster, 1.17-buster
ARG VARIANT="1.18-bullseye"
FROM mcr.microsoft.com/vscode/devcontainers/go:0-${VARIANT}

# Additional OS packages
RUN echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list \
  && apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends curl build-essential goreleaser

# Non-apt installs for user (go, Node, etc)
USER vscode
RUN go install cuelang.org/go/cmd/cue@latest
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.46.2
