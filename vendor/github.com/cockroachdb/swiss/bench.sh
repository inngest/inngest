#!/bin/bash

go test -c
./swiss.test -test.v -test.run - -test.bench . -test.count 10 -test.benchmem -test.timeout 10h | tee out
grep -v swissMap out | sed 's,/runtimeMap,,g' > out.runtime
grep -v runtimeMap out | sed 's,/swissMap,,g' > out.swiss
benchstat out.runtime out.swiss
