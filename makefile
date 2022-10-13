# gomaker v1.3
# uses *existing* go.mod file
# aerth 22
module != go list -m
name != basename ${module}
gofiles != find . -name '*.go'
VERSION=$(shell git describe --tags 2>/dev/null)
ifeq (,$(VERSION))
VERSION=0.0.1
endif
COMMIT=$(shell git rev-parse --verify --short HEAD 2>/dev/null)
ifeq (,$(COMMIT))
COMMIT=none
endif

# EDIT ME files to embed
EMBED_ARGS=./makefile

# EDIT ME version and commit variables can exist and be used
ldflags ?= -w -s -X main.version=${VERSION} -X main.commit=${COMMIT}
goflags ?= -v -ldflags '$(ldflags)'
ifeq (command-line-arguments,$(module))
name=
endif

ifneq ($(shell ls ./cmd/ 2>/dev/null || true | wc -l), 0)
cmdpath ?= ./cmd/...
else
cmdpath ?= 
endif
buildfunc = go build -o . $(goflags) ${flags} $(1)$(cmdpath)
.PHONY += goassets
bin/${name}: go.mod ./assets/assets.go $(gofiles)
	@echo using deps $^
	mkdir -p bin
	cd bin && $(call buildfunc,../)
./assets/assets.go: ${EMBED_ARGS}
	type go-bindata || go install github.com/aerth/go-bindata
	go-bindata -pkg assets -o $@ ${EMBED_ARGS}
# cross compile release
crossdirs ?= bin/linux bin/freebsd bin/osx bin/windows
cross: go.mod $(gofiles)
	mkdir -v -p $(crossdirs) 
	# unroll here if needed
	cd bin/linux && $(call buildfunc,../../)
	cd bin/freebsd && GOOS=freebsd $(call buildfunc,../../)
	cd bin/osx && GOOS=darwin $(call buildfunc,../../)
	cd bin/windows && GOOS=windows $(call buildfunc,../../)
help:
	@echo "name:    ${name}"
	@echo "module:  ${module}"
	@echo "gofiles: ${gofiles}"
	@echo "goflags: ${goflags}"
run: bin/$(name)
	test -x bin/$(name)
	$^ &>>debug.log
test:
	go test ${flags} -v ./...
go.mod:
	@echo "run 'go mod init myprojectname'"
	@exit 1
clean:
	${RM} -r bin assets/assets.go
