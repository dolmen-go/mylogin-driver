include ../../Makefile

.PHONY: install

install:
	$(go) install $(shell $(go) list -m)@$(shell $(MAKE) -f $(firstword $(MAKEFILE_LIST)) go-version)

upgrade-flagx:
	$(shell cd $(shell go env GOPATH)/src/github.com/dolmen-go/flagx; $(MAKE) go-get)
	$(go) mod tidy
