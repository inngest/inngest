.PHONY: build
build: #get-library generate-library
	yarn build

.PHONY: dirty
dirty:
	go run ./library.go
	yarn prettier
	git diff --exit-code


.PHONY: get-library
get-library:
	echo "generating library json"
	go run ./getlibrary.go || ./getlibrary
	rm -rf ./library/
	echo "library json generated"

.PHONY: generate-library
generate-library:
	./generate-library.js

.PHONY: library
library: get-library generate-library
