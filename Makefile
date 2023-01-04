.PHONY: build
build:
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
	yarn prettier
	git diff --exit-code