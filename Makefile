.PHONY: build
build:
	go build cmd/build-plugins-json/build-plugins-json.go

.PHONY: run
run:
	go run cmd/build-plugins-json/build-plugins-json.go