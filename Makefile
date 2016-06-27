PROG = corectl
DAEMON = corectld
ORGANIZATION = github.com/TheNewNormal
REPOSITORY = $(ORGANIZATION)/$(PROG)

GOARCH ?= $(shell go env GOARCH)
GOOS ?= $(shell go env GOOS)
CGO_ENABLED = 1
GO15VENDOREXPERIMENT = 0

BUILD_DIR ?= $(shell pwd)/bin
GOPATH := $(shell echo $(PWD) | \
        sed -e "s,src/$(REPOSITORY).*,,"):$(shell godep go env \
        | grep GOPATH | sed -e 's,",,g' -e "s,.*=,,")
GODEP = GOPATH=$(GOPATH) GO15VENDOREXPERIMENT=$(GO15VENDOREXPERIMENT) \
    GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) godep
GOBUILD = $(GODEP) go build

VERSION := $(shell git describe --abbrev=6 --dirty=+untagged --always --tags)
BUILDDATE = $(shell /bin/date "+%FT%T%Z")

HYPERKIT_GIT = "https://github.com/docker/hyperkit.git"
HYPERKIT_COMMIT = c42f126

ifeq ($(DEBUG),true)
    GO_GCFLAGS := $(GO_GCFLAGS) -N -l
else
    GO_LDFLAGS := $(GO_LDFLAGS) -w -s
endif

GO_LDFLAGS := $(GO_LDFLAGS) \
	-X $(REPOSITORY)/release.Version=$(VERSION) \
	-X $(REPOSITORY)/release.BuildDate=$(BUILDDATE)

all: cmd/client cmd/server docs
	@git status

cmd/client:
	mkdir -p $(BUILD_DIR)
	rm -rf $(BUILD_DIR)/$(PROG)
	cd $@; $(GOBUILD) -o $(BUILD_DIR)/$(PROG) \
		-gcflags "$(GO_GCFLAGS)" -ldflags "$(GO_LDFLAGS)"
	@touch $@

cmd/server:
	mkdir -p $(BUILD_DIR)
	rm -rf $(BUILD_DIR)/$(DAEMON)
	cd $@; $(GOBUILD) -o $(BUILD_DIR)/$(DAEMON) \
		-gcflags "$(GO_GCFLAGS)" -ldflags "$(GO_LDFLAGS)"
	@touch $@

assets:
	@cd assets; \
		rm -f assets_vfsdata.go ; \
		$(GODEP) go run assets_generator.go -tags=dev

clean: assets
	@rm -rf $(BUILD_DIR)/* documentation/

dependencies_update:
	@rm -rf Godeps/
	# XXX godep won't save this as a build dep run a runtime one so we cheat...
	/usr/bin/sed -i.bak \
		-e s"|github.com/deis/pkg/log|github.com/shurcooL/vfsgen|" \
		-e "s|import (|import ( \"github.com/shurcooL/httpfs/vfsutil\"|" \
			assets/assets.go
	$(GODEP) save ./...
	# ... and un-cheat
	cp assets/assets.go.bak assets/assets.go
	@rm -rf assets/assets.go.bak
	@git status

docs: cmd/server cmd/client documentation/markdown documentation/man

hyperkit: force
	mkdir -p $(BUILD_DIR)
	# implies...
	# - brew install opam
    # - opam init --yes
    # - opam pin add qcow-format git://github.com/mirage/ocaml-qcow#master --yes
    # - opam install --yes uri qcow-format ocamlfind
    # - make clean
    # - eval `opam config env` && make
	rm -rf $@
	git clone $(HYPERKIT_GIT)
	cd $@; git checkout $(HYPERKIT_COMMIT) ; make clean ; make all
	mkdir -p bin/
	cp $@/build/com.docker.hyperkit bin/corectld.runner

documentation/man: force
	@mkdir -p documentation/man
	bin/$(PROG) utils genManPages
	bin/$(DAEMON) utils genManPages
	@for p in $$(ls documentation/man/*.1); do \
		sed -i.bak "s/$$(/bin/date '+%h %Y')//" "$$p" ;\
		sed -i.bak "/spf13\/cobra$$/d" "$$p" ;\
		rm "$$p.bak" ;\
	done

documentation/markdown: force
	@mkdir -p documentation/markdown
	bin/$(PROG) utils genMarkdownDocs
	bin/$(DAEMON) utils genMarkdownDocs
	@for p in $$(ls documentation/markdown/*.md); do \
		sed -i.bak "/spf13\/cobra/d" "$$p" ;\
		rm "$$p.bak" ;\
	done

.PHONY: clean all docs force assets cmd/server cmd/client
