.PHONY: build
build:
	yarn build

# Create a production build
.PHONY: build-prod
build-prod:
	yarn build

.PHONY: cloudflare-build
cloudflare-build:
	make build-prod
	cp ./_redirects ./out/_redirects

.PHONY: dirty
dirty:
	yarn prettier
	git diff --exit-code