#!/bin/bash
set -e

docker-compose -f ./opt/compose.yml up -d

for i in {1..120}; do
	echo "check"
	# There shold be 6 lines with 2 queues made.
	num=$(docker exec localstack_main sh -c 'awslocal sqs list-queues' | wc -l)
	if [ $num -gt 5 ]; then
		exit
	fi;
	sleep 1;
done
