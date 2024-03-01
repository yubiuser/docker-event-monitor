BINARY_NAME = docker-event-monitor

# using the ?= assignment operator: Assign only if variable is not set (e.g. via environment) yet
# this allows overwriting via CI
GIT_COMMIT ?= $(shell git --no-pager describe --always --abbrev=8 --dirty)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_VERSION ?= $(shell git --no-pager describe --tags --always --abbrev=8 --dirty)
GIT_DATE ?= $(shell git --no-pager show --date=short --format=%at --name-only | head -n 1)

# GIT_TAG is only set when a CI build is trigged via release tag
ifdef GIT_TAG
override GIT_VERSION = ${GIT_TAG}
endif

# in case 'git' or the repo is not available, GIT_XXX is set empty via the assignment above
# so we set them explicitly
ifeq ($(GIT_DATE),)
GIT_DATE = 0
endif
ifeq ($(GIT_COMMIT),)
GIT_COMMIT = "n/a"
endif
ifeq ($(GIT_BRANCH),)
GIT_BRANCH = "n/a"
endif
ifeq ($(GIT_VERSION),)
GIT_VERSION = "n/a"
endif

DATE = $(shell date +%s)
build:
	CGO_ENABLED=0 go build -ldflags "-s -w -X 'main.version=${GIT_VERSION}' -X 'main.gitdate=${GIT_DATE}' -X 'main.date=${DATE}' -X 'main.commit=${GIT_COMMIT}' -X 'main.branch=${GIT_BRANCH}'" -o=${BINARY_NAME}
