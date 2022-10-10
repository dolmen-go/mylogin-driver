include ../../Makefile

.PHONY: install

install:
	$(go) install $(shell $(go) list -m)@$(shell $(MAKE) -f $(firstword $(MAKEFILE_LIST)) go-version)
