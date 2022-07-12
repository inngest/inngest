#!/bin/bash
set -ex

echo "yea"
docker-compose -f ./opt/compose.yml up -d

# Wait for queues to be made...
sleep 5
