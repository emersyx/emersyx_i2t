emersyx-irc2telegram.so:
	@go build -buildmode=plugin -o emersyx-irc2telegram.so internal/irc2telegram/*

.PHONY: test
test: emersyx-irc2telegram.so
	@echo "Running the tests with gofmt..."
	@test -z $(shell gofmt -s -l internal/irc2telegram/*.go)
	@echo "Running the tests with go vet..."
	@go vet ./...
	@echo "Running the tests with golint..."
	@golint -set_exit_status $(shell go list ./...)
	@echo "Running the unit tests..."
	@cd internal/irc2telegram; go test -v -conffile ../../config/config-template.toml
