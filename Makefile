# gomaker
# Makefile generated by Gomaker 1c6f87d


NAME=gomaker
VERSION=
PREFIX ?= /usr/local/bin
export CGO_ENABLED=0
COMMIT=$(shell git rev-parse --verify --short HEAD)
RELEASE=${VERSION}${COMMIT}


all:	build


build:
	@echo 'Building ${NAME} version ${RELEASE}'

	go build -o ${NAME} -x --ldflags "-s -X main.version=${RELEASE} "
	@echo 'Successfully built ${NAME}'



install:
	@echo 'PREFIX=${PREFIX}'

	@mkdir -p ${PREFIX}
	@mv ${NAME} ${PREFIX}/${NAME}
	@echo 'Successfully installed ${NAME} to ${PREFIX}'

