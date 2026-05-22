checks: build test pre-commit
.PHONY: checks

build: oe
.PHONY: build

oe:
	go build ./cmd/$@

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
