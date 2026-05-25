.PHONY: test web-build

test:
	go test ./...

web-build:
	cd web && bun run build
