# generated by gomaker v1.2.0-2ccc0ee
# https://github.com/aerth/gomaker
buildflags ?= -v -ldflags '-w -s'
COMMIT=$(shell git rev-parse --verify --short HEAD 2>/dev/null)
VERSION=$(shell git describe --tags 2>/dev/null)
buildflags := $(buildflags) -ldflags '-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)'

gomaker: *.go
	go build $(buildflags) -o $@
clean:
	rm -vf  gomaker
.PHONY += clean
