.PHONY: deps
deps:
	docker compose run --rm golang go mod tidy

.PHONY: fmt
fmt:
	docker compose run --rm golang go fmt ./...

.PHONY: test
test:
	docker compose up -d test-server-1
	docker compose up -d test-server-2
	sleep 1
	docker compose run --rm golang go test -v ./...
	docker compose down

.PHONY: build
build:
	# possible values for target: https://github.com/golang/go/blob/58c5db3169c801737cb0e0ed4886554763c861eb/src/go/build/syslist.go#L14
	docker compose run --rm -e GOOS="$(target)" golang go build -ldflags="-s -w" -o ./bin/msdl
