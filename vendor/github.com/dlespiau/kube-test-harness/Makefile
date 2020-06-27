all:

dep:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

lint:
	@.ci/go-lint

unit-tests:
	go build -i .
	go test -v . ./logger

integration-tests:
	go build -i .
	go test -v ./examples/...
	go test -v ./tests

.PHONY: dep integration-tests lint unit-tests
