checks: build test pre-commit

build: oe

oe:
	go build ./cmd/$@

test:
	go test -cover -failfast ./...

clean:
	rm -f oe .cover.out

pre-commit:
	pre-commit run -a
.PHONY: oe test clean pre-commit

gocovsh:
	go test -v ./... -coverpkg=./... -coverprofile=.coverprofile
	gocovsh --profile .coverprofile
	rm .coverprofile
