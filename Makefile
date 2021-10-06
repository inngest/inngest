.PHONY: build
build: get-library generate-library blog
	yarn build

.PHONY: dirty
dirty:
	./blog.js
	go run ./library.go
	yarn prettier
	git diff --exit-code


.PHONY: get-library
get-library:
	echo "generating library json"
	go run ./getlibrary.go
	rm -rf ./library/
	echo "library json generated"

.PHONY: generate-library
generate-library:
	./generate-library.js

.PHONY: library
library: get-library generate-library

.PHONY: blog
blog:
	./blog.js
