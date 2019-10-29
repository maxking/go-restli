SHELL := /bin/bash

PACKAGE_PREFIX := github.com/PapaCharlie/go-restli/generated

test: clean imports
	mkdir -p tmp
	go run main.go \
		--package-prefix $(PACKAGE_PREFIX) \
		--output-dir tmp \
		--snapshot-mode \
		$(sort $(wildcard rest.li-test-suite/client-testsuite/snapshots/*))
	mv tmp/$(PACKAGE_PREFIX) .
	rm -r tmp
	go test -count=1 $(PACKAGE_PREFIX)

clean:
	git -C rest.li-test-suite reset --hard origin/master
	rm -rf generated

imports:
	goimports -w main.go codegen d2 protocol