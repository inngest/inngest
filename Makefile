.PHONY: build
build: library blog

	yarn build
.PHONY: dirty
dirty:
	./blog.js
	go run ./library.go
	yarn prettier
	git diff --exit-code


.PHONY: library
library:
	echo "generating library json"
	go run ./library.go
	echo "library json generated"

.PHONY: blog
blog:
	./blog.js
