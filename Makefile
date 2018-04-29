emi2t.so: goget
	@go build -buildmode=plugin -o emi2t.so emi2t/*

.PHONY: goget
goget:
	@go get emersyx.net/emersyx/api
	@go get emersyx.net/emersyx/api/ircapi
	@go get emersyx.net/emersyx/api/tgapi
	@go get github.com/BurntSushi/toml
	@go get github.com/golang/lint/golint

.PHONY: test
test: emi2t.so
	@echo "Running the tests with gofmt, go vet and golint..."
	@test -z $(shell gofmt -s -l emi2t/*.go)
	@go vet ./...
	@golint -set_exit_status $(shell go list ./...)
	@cd emi2t; go test -v -conffile ../config.toml
