#!/bin/bash
set -e

docker-compose -f ./opt/compose.yml up -d

sleep 1;
