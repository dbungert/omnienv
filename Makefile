checks: build test pre-commit
.PHONY: checks

build: oe
.PHONY: build

oe:
	go build ./cmd/$@
.PHONY: oe

test:
	go test -cover -failfast ./...
.PHONY: test

clean:
	rm -f oe .coverprofile
.PHONY: clean

pre-commit:
	pre-commit run -a
.PHONY: pre-commit

gocovsh:
	go test -v ./... -coverpkg=./... -coverprofile=.coverprofile
	gocovsh --profile .coverprofile
	rm .coverprofile
.PHONY: gocovsh

ci-tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
.PHONY: ci-tools
