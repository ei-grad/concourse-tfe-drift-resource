%PHONY: test

build: check in out

concourse-tfe-drift-resource: *.go
	go build

LINKS := check in out
$(LINKS): concourse-tfe-drift-resource
	ln -fs ./concourse-tfe-drift-resource $@

# go install go.uber.org/mock/mockgen@latest
makemocks:
	mkdir -p mock-go-tfe
	mockgen github.com/hashicorp/go-tfe Workspaces,Runs,Variables,StateVersions > mock-go-tfe/mocks.go

test: *.go makemocks
	go test -v -coverprofile cover.out -covermode=atomic
	go tool cover -html=cover.out -o coverage.html

lint: check
	golangci-lint run

clean:
	rm -rf concourse-tfe-drift-resource check in out cover.out coverage.html test_output
