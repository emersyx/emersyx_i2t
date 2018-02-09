.PHONY: emi2t-dep emi2t-goget test

emi2t-dep:
	dep ensure
	go build -buildmode=plugin -o emi2t.so emi2t/*

emi2t-goget:
	go get -t -buildmode=plugin ./emi2t

test:
	@echo "Running the tests with gofmt, go vet and golint..."
	-@dep ensure # dep is not available in travis-ci
	@test -z $(shell gofmt -s -l emi2t/*.go)
	@go vet ./...
	@golint -set_exit_status $(shell go list ./...)
	@cd emi2t; go test -v -conffile ../config.toml
