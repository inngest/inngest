#!/usr/bin/env bash

set -ev

if [ ! -z "$CC_TEST_REPORTER_ID" ]; then
  curl -L https://codeclimate.com/downloads/test-reporter/test-reporter-latest-$(go env GOOS)-$(go env GOARCH) > ./cc-test-reporter
  chmod +x ./cc-test-reporter
  ./cc-test-reporter before-build
fi

docker-compose up -d
go test -coverprofile=./c.out -v -race -timeout 30m ./...
docker-compose down -v

if [ ! -z "$CC_TEST_REPORTER_ID" ]; then
  ./cc-test-reporter after-build -p $(go list -m)
fi

if [ ! -z "$CODECOV_TOKEN" ]; then
  cp c.out coverage.txt
  bash <(curl -s https://codecov.io/bash)
fi