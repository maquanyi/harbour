BUILDTAGS=
export GOPATH:=$(CURDIR)/Godeps/_workspace:$(GOPATH)

all:
		go build -tags "$(BUILDTAGS)" -o harbour .

install:
		cp harbour /usr/local/bin/harbour
clean:
		rm harbour
