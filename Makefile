.PHONY: build
build: #get-library generate-library
	yarn build

# Create a production build
.PHONY: build-prod
build-prod:
	yarn build
	yarn next export
	yarn render-social-preview-images

.PHONY: cloudflare-build
cloudflare-build:
	make build-prod
	cp ./_redirects ./out/_redirects

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
