
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

bin/sftpcli-$(GOOS)-$(GOARCH): cmd/sftpcli/main.go cmd/sftpcli/go.mod cmd/sftpcli/go.sum
	(cd $(dir $<); go build -o $(abspath $@) -ldflags='-s -w' $(notdir $<))
