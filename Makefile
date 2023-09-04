.PHONY: build
build:
	pnpm build

# Create a production build
.PHONY: build-prod
build-prod:
	pnpm build

.PHONY: cloudflare-build
cloudflare-build:
	make build-prod
	cp ./_redirects ./out/_redirects

.PHONY: dirty
dirty:
	pnpm prettier
	git diff --exit-code
