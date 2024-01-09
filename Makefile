%PHONY: test

build: check in out

concourse-tfe-drift-resource: *.go
	go build

LINKS := check in out
$(LINKS): concourse-tfe-drift-resource
	ln -fs ./concourse-tfe-drift-resource $@

# go install go.uber.org/mock/mockgen@latest
mockgen_test.go:
	mockgen \
		-package main \
		-destination mockgen_test.go \
		github.com/hashicorp/go-tfe Workspaces,Runs,Variables,StateVersions

test: *.go mockgen_test.go
	go test -v -coverprofile cover.out -covermode=atomic
	go tool cover -html=cover.out -o coverage.html

lint: check
	golangci-lint run

ARTIFACTS := concourse-tfe-drift-resource check in out cover.out coverage.html test_output mockgen_test.go
clean:
	rm -rf $(ARTIFACTS)
