#!/bin/bash
set -ex

docker-compose -f ./opt/compose.yml up -d

# Wait for queues to be made...
sleep 5
