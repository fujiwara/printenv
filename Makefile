GIT_VER := $(shell git describe --tags)
.PHONY: test build clean image release-image

build: test
	go build .

test:
	go test -o printenv ./...

clean:
	rm -f printenv

image:
	docker build \
        --tag ghcr.io/fujiwara/printenv:$(GIT_VER) \
        .

release-image: image
	docker push ghcr.io/fujiwara/printenv:$(GIT_VER)
