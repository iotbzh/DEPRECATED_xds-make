# Makefile used to build xds-make and xds-exec commands

# Application Version
VERSION := 1.0.0


# Retrieve git tag/commit to set sub-version string
ifeq ($(origin SUB_VERSION), undefined)
	SUB_VERSION := $(shell git describe --tags 2>/dev/null | sed 's/^v//')
	ifneq ($(SUB_VERSION), )
		VERSION := $(firstword $(subst -, ,$(SUB_VERSION)))
		SUB_VERSION := $(word 2,$(subst -, ,$(SUB_VERSION)))
	else
		SUB_VERSION := $(shell git describe --tags --always  | sed 's/^v//')
		ifeq ($(SUB_VERSION), )
			SUB_VERSION := unknown-dev
		endif
	endif
endif

HOST_GOOS=$(shell go env GOOS)
HOST_GOARCH=$(shell go env GOARCH)
ARCH=$(HOST_GOOS)-$(HOST_GOARCH)

EXT=
ifeq ($(HOST_GOOS), windows)
	EXT=.exe
endif


mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
ROOT_SRCDIR := $(patsubst %/,%,$(dir $(mkfile_path)))
BINDIR := $(ROOT_SRCDIR)/bin
ROOT_GOPRJ := $(abspath $(ROOT_SRCDIR)/../../../..)
PACKAGE_DIR := $(ROOT_SRCDIR)/package

export GOPATH := $(shell go env GOPATH):$(ROOT_GOPRJ)
export PATH := $(PATH):$(ROOT_SRCDIR)/tools

VERBOSE_1 := -v
VERBOSE_2 := -v -x

REPOPATH=github.com/iotbzh/xds-make
TARGET := xds-make

all: xds-make xds-exec

build: xds-make

xds-make: vendor
	@echo "### Build $@ (version $(VERSION), subversion $(SUB_VERSION))";
	@cd $(ROOT_SRCDIR); $(BUILD_ENV_FLAGS) go build $(VERBOSE_$(V)) -i -o $(BINDIR)/$@$(EXT) -ldflags "-X main.AppName=$@ -X main.AppVersion=$(VERSION) -X main.AppSubVersion=$(SUB_VERSION)" .
	@([ "$(HOST_GOOS)" = "linux" ] && { cd $(BINDIR) && ln -sf $@ $(subst xds-,,$@); } || { true; } )

xds-exec: vendor
	@echo "### Build $@ (version $(VERSION), subversion $(SUB_VERSION))";
	@cd $(ROOT_SRCDIR); $(BUILD_ENV_FLAGS) go build $(VERBOSE_$(V)) -i -o $(BINDIR)/$@$(EXT) -ldflags "-X main.AppName=$@ -X main.AppVersion=$(VERSION) -X main.AppSubVersion=$(SUB_VERSION)" .
	@([ "$(HOST_GOOS)" = "linux" ] && { cd $(BINDIR) && ln -sf $@ $(subst xds-,,$@); } || { true; } )

test: tools/glide
	go test --race $(shell ./tools/glide novendor)

vet: tools/glide
	go vet $(shell ./tools/glide novendor)

fmt: tools/glide
	go fmt $(shell ./tools/glide novendor)

.PHONY: clean
clean:
	rm -rf $(BINDIR)/* debug $(ROOT_GOPRJ)/pkg/*/$(REPOPATH) $(PACKAGE_DIR)

distclean: clean
	rm -rf $(BINDIR) tools glide.lock vendor $(ROOT_SRCDIR)/*.zip

package: clean all
	@mkdir -p $(PACKAGE_DIR)/xds-make
	cp -a $(BINDIR)/*make$(EXT) $(PACKAGE_DIR)/xds-make
	cd $(PACKAGE_DIR) && zip  --symlinks -r $(ROOT_SRCDIR)/xds-make_$(ARCH)-v$(VERSION)_$(SUB_VERSION).zip ./xds-make
	@mkdir -p $(PACKAGE_DIR)/xds-exec
	cp -a $(BINDIR)/*exec$(EXT) $(PACKAGE_DIR)/xds-exec
	cd $(PACKAGE_DIR) && zip  --symlinks -r $(ROOT_SRCDIR)/xds-exec_$(ARCH)-v$(VERSION)_$(SUB_VERSION).zip ./xds-exec

.PHONY: package-all
package-all:
	@echo "# Build linux amd64..."
	GOOS=linux GOARCH=amd64 RELEASE=1 make -f $(ROOT_SRCDIR)/Makefile package
	@echo "# Build windows amd64..."
	GOOS=windows GOARCH=amd64 RELEASE=1 make -f $(ROOT_SRCDIR)/Makefile package

#release: releasetar
#	goxc -d ./release -tasks-=go-vet,go-test -os="linux darwin" -pv=$(VERSION)  -arch="386 amd64 arm arm64" -build -ldflags "-X main.AppName=$@ -X main.AppVersion=$(VERSION) -X main.AppSubVersion=$(SUB_VERSION)" -resources-include="README.md,Documentation,LICENSE,contrib" -main-dirs-exclude="vendor"

#releasetar:
#	mkdir -p release/$(VERSION)
#	glide install --strip-vcs --strip-vendor --update-vendored --delete
#	glide-vc --only-code --no-tests --keep="**/*.json.in"
#	git ls-files > /tmp/$(TARGET)-build
#	find vendor >> /tmp/$(TARGET)-build
#	find webapp/ -path webapp/node_modules -prune -o -print >> /tmp/$(TARGET)-build
#	tar -cvf release/$(VERSION)/$(TARGET)_$(VERSION)_src.tar -T /tmp/$(TARGET)-build --transform 's,^,$(TARGET)_$(VERSION)/,'
#	rm /tmp/$(TARGET)-build
#	gzip release/$(VERSION)/$(TARGET)_$(VERSION)_src.tar

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
