#!/bin/bash
set -e

docker-compose -f ./opt/compose.yml up -d

end=$((SECONDS+60))

while [ $SECONDS -lt $end ]; do
	echo "check"
	# There shold be 6 lines with 2 queues made.
	num=$(docker exec -ti localstack_main sh -c 'awslocal sqs list-queues' | wc -l)
	if [ $num -gt 5 ];
	then
		exit
	fi;
done
