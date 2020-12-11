define baseinfo
    @echo "Build start"
    @echo "Build time: $(DATETIME)"
    @echo "Build target: arch $(1) for $(2)"
    @echo "GOVERSION: ${GOLANG_VERSION}"
    @echo "GOPATH:" ${GOPATHS}
    @echo
endef

PROG_NAME = redis-trib
DATETIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GITURL = https://github.com/PoplarYang/redis-trib
ifneq (,$(wildcard .git/.*))
    COMMIT = $(shell git rev-parse HEAD 2> /dev/null || true)
    VERSION = $(shell git describe --tags --abbrev=0 2> /dev/null)
    BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
else
    COMMIT = "unknown"
    VERSION = "unknown"
    BRANCH = "unkown"
endif

GO=go
GOLANG_VERSION = $(shell go version | cut -d ' ' -f3 | cut -c 3-)
#Replaces ":" (*nix), ";" (windows) with newline for easy parsing
GOPATHS = $(shell echo ${GOPATH} | tr ":" "\n" | tr ";" "\n")

# See Golang issue re: '-trimpath': https://github.com/golang/go/issues/13809
GO_GCFLAGS=$(shell				        \
	set -- ${GOPATHS};			        \
	echo "-gcflags=-trimpath=$${1}";	\
	)

GO_ASMFLAGS=$(shell				        \
	set -- ${GOPATHS};			        \
	echo "-asmflags=-trimpath=$${1}";	\
	)

GO_LDFLAGS="-extldflags '-static' -s -w \
	-X main.version=$(VERSION) \
	-X main.gitURL=$(GITURL) \
	-X main.gitCommit=$(COMMIT) \
	-X main.Branch=$(BRANCH) \
	-X main.BuildTime=$(DATETIME)"

DARWIN-AMD64   = $(PROG_NAME).$(VERSION).darwin-amd64
LINUX-386      = $(PROG_NAME).$(VERSION).linux-386
LINUX-AMD64    = $(PROG_NAME).$(VERSION).linux-amd64
LINUX-ARM64    = $(PROG_NAME).$(VERSION).linux-arm64
LINUX-MIPS64EL = $(PROG_NAME).$(VERSION).linux-mips64el
WIN-386        = $(PROG_NAME).$(VERSION).windows-386.exe
WIN-AMD64      = $(PROG_NAME).$(VERSION).windows-amd64.exe

## Make linux windows darwin bins for $PROG_NAME
all: linux windows darwin

## Make for darwin (arch: amd64)
darwin:
	$(call baseinfo,amd64,darwin)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GO_GCFLAGS) $(GO_ASMFLAGS) -ldflags $(GO_LDFLAGS) -o $(DARWIN-AMD64)

## Make for linux (arch: 386, amd64, arm, mips64le)
linux:
	$(call baseinfo,386,linux)
	CGO_ENABLED=0 GOOS=linux GOARCH=386 $(GO) build $(GO_GCFLAGS) $(GO_ASMFLAGS) -ldflags $(GO_LDFLAGS) -o $(LINUX-386)
	$(call baseinfo,amd64,linux)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GO_GCFLAGS) $(GO_ASMFLAGS) -ldflags $(GO_LDFLAGS) -o $(LINUX-AMD64)
	$(call baseinfo,arm,linux)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GO_GCFLAGS) $(GO_ASMFLAGS) -ldflags $(GO_LDFLAGS) -o $(LINUX-ARM64)
	$(call baseinfo,mips64le,linux)
	CGO_ENABLED=0 GOOS=linux GOARCH=mips64le $(GO) build $(GO_GCFLAGS) $(GO_ASMFLAGS) -ldflags $(GO_LDFLAGS) -o $(LINUX-MIPS64EL)

## Make for windows (arch: 386, amd64)
windows:
	$(call baseinfo,386,windows)
	CGO_ENABLED=0 GOOS=windows GOARCH=386 $(GO) build $(GO_GCFLAGS) $(GO_ASMFLAGS) -ldflags $(GO_LDFLAGS) -o $(WIN-386)
	$(call baseinfo,amd64,windows)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GO_GCFLAGS) $(GO_ASMFLAGS) -ldflags $(GO_LDFLAGS) -o $(WIN-AMD64)

## Build debug trace for $PROG_NAME.
debug:
	$(GO) build -n -v -i -ldflags "-X main.gitCommit=${COMMIT} -X main.version=${VERSION}" -o ${MODULE} .

deps:
	$(GO) mod download

## Run test case for this go project.
test:
	$(GO) test $($(GO) list ./...)

## Clean the project
clean:
	$(GO) clean
	-rm $(DARWIN-AMD64) $(LINUX-386) $(LINUX-AMD64) $(LINUX-ARM64) $(LINUX-MIPS64EL) $(WIN-386) $(WIN-AMD64) 2> /dev/null || true

## Print help
help: # Some kind of magic from https://gist.github.com/rcmachado/af3db315e31383502660
	$(info Available targets)
	@awk '/^[a-zA-Z\-_0-9]+:/ {                                                 \
		nb = sub( /^## /, "", helpMsg );                                        \
		if(nb == 0) {                                                           \
			helpMsg = $$0;                                                      \
			nb = sub( /^[^:]*:.* ## /, "", helpMsg );                           \
		}                                                                       \
		if (nb) {                                                               \
			h = sub( /[^ ]*PROG_NAME/, "'${PROG_NAME}'", helpMsg );                   \
			printf "   \033[1;31m%-" width "s\033[0m %s\n", $$1, helpMsg;       \
		}															            \
	}                                                                           \
	{ helpMsg = $$0 }'                                                          \
	width=$$(grep -o '^[a-zA-Z_0-9]\+:' $(MAKEFILE_LIST) | wc -L 2> /dev/null)  \
	$(MAKEFILE_LIST)
