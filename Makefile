all: test build
build:
	go build -o derek

docker:
	docker build -t derek:local .

test:
	go test -v $(shell go list ./... | grep -v /vendor/ | grep -v /build/ | grep -v /template/) -cover
