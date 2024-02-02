BINARY_NAME := docker-event-monitor
GIT_COMMIT ?= $(shell git --no-pager describe --always --abbrev=8 --dirty)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_VERSION ?= $(shell git --no-pager describe --tags --always --abbrev=8 --dirty)
GIT_DATE ?= $(shell git --no-pager show --date=short --format=\"%at\" --name-only | head -n 1)
ifdef GIT_TAG
override GIT_VERSION = ${GIT_TAG}
endif
DATE := $(shell date +%s)
build:
	CGO_ENABLED=0 go build -ldflags "-s -w -X 'main.version=${GIT_VERSION}' -X 'main.gitdate=${GIT_DATE}' -X 'main.date=${DATE}' -X 'main.commit=${GIT_COMMIT}' -X 'main.branch=${GIT_BRANCH}'" -o=${BINARY_NAME} docker-event-monitor.go


