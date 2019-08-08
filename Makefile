
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

bin/sftpcli-$(GOOS)-$(GOARCH): cmd/sftpcli/main.go cmd/sftpcli/go.mod cmd/sftpcli/go.sum
	(cd $(dir $<); go build -o $(abspath $@) -ldflags='-s -w' $(notdir $<))

docker/root/usr/bin/sftpcli: cmd/sftpcli/main.go cmd/sftpcli/go.mod cmd/sftpcli/go.sum
	(cd $(dir $<); GOOS=linux ARCH=amd64 go build -o $(abspath $@) -ldflags='-s -w' $(notdir $<))

.PHONY: docker
docker: docker/root/usr/bin/sftpcli
	docker build $(abspath $@) -t docker.pkg.github.com/simon-engledew/sftpcli/sftpcli:v0.0.3
