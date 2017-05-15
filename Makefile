# Makefile used to build XDS make command

# Application Version
VERSION := 1.0.0


# Retrieve git tag/commit to set sub-version string
ifeq ($(origin SUB_VERSION), undefined)
	SUB_VERSION := $(shell git describe --tags --always | sed 's/^v//')
	ifeq ($(SUB_VERSION), )
		SUB_VERSION=unknown-dev
	endif
endif

HOST_GOOS=$(shell go env GOOS)
HOST_GOARCH=$(shell go env GOARCH)

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
ROOT_SRCDIR := $(patsubst %/,%,$(dir $(mkfile_path)))
ROOT_GOPRJ := $(abspath $(ROOT_SRCDIR)/../../../..)

export GOPATH := $(shell go env GOPATH):$(ROOT_GOPRJ)
export PATH := $(PATH):$(ROOT_SRCDIR)/tools

VERBOSE_1 := -v
VERBOSE_2 := -v -x

REPOPATH=github.com/iotbzh/xds-make
TARGET := xds-make
TARGET_NATIVE := $(subst xds-,,$(TARGET))

all: build

build: vendor
	@echo "### Build $(TARGET) (version $(VERSION), subversion $(SUB_VERSION))";
	@cd $(ROOT_SRCDIR); $(BUILD_ENV_FLAGS) go build $(VERBOSE_$(V)) -i -o bin/$(TARGET) -ldflags "-X main.AppVersion=$(VERSION) -X main.AppSubVersion=$(SUB_VERSION)" .
	@(cd bin && ln -sf $(TARGET) $(TARGET_NATIVE))

test: tools/glide
	go test --race $(shell ./tools/glide novendor)

vet: tools/glide
	go vet $(shell ./tools/glide novendor)

fmt: tools/glide
	go fmt $(shell ./tools/glide novendor)

clean:
	rm -rf ./bin/* debug $(ROOT_GOPRJ)/pkg/*/$(REPOPATH)

distclean: clean
	rm -rf bin tools glide.lock vendor

# FIXME - package webapp
release: releasetar
	goxc -d ./release -tasks-=go-vet,go-test -os="linux darwin" -pv=$(VERSION)  -arch="386 amd64 arm arm64" -build -ldflags "-X main.AppVersion=$(VERSION) -X main.AppSubVersion=$(SUB_VERSION)" -resources-include="README.md,Documentation,LICENSE,contrib" -main-dirs-exclude="vendor"

releasetar:
	mkdir -p release/$(VERSION)
	glide install --strip-vcs --strip-vendor --update-vendored --delete
	glide-vc --only-code --no-tests --keep="**/*.json.in"
	git ls-files > /tmp/$(TARGET)-build
	find vendor >> /tmp/$(TARGET)-build
	find webapp/ -path webapp/node_modules -prune -o -print >> /tmp/$(TARGET)-build
	tar -cvf release/$(VERSION)/$(TARGET)_$(VERSION)_src.tar -T /tmp/$(TARGET)-build --transform 's,^,$(TARGET)_$(VERSION)/,'
	rm /tmp/$(TARGET)-build
	gzip release/$(VERSION)/$(TARGET)_$(VERSION)_src.tar

vendor: tools/glide glide.yaml
	./tools/glide install --strip-vendor

vendor/debug: vendor
	(cd vendor/github.com/iotbzh && rm -rf xds-server && ln -s ../../../../xds-server)

tools/glide:
	@echo "Downloading glide"
	mkdir -p tools
	curl --silent -L https://glide.sh/get | GOBIN=./tools  sh

goenv:
	@go env

help:
	@echo "Main supported rules:"
	@echo "  build               (default)"
	@echo "  release"
	@echo "  clean"
	@echo "  distclean"
	@echo ""
	@echo "Influential make variables:"
	@echo "  V                 - Build verbosity {0,1,2}."
	@echo "  BUILD_ENV_FLAGS   - Environment added to 'go build'."
