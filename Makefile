.PHONY: test build clean

build: test
	go build .

test:
	go test -o printenv ./...

clean:
	rm -f printenv
