CC=go
CFLAGS?=-i
GOOS=linux
CGO_ENABLED?=0

NAME=kube-csr

$(NAME):
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) $(CC) build $(CFLAGS) -o $@

clean:
	$(RM) $(NAME) example $(NAME).sha512sum

re: clean $(NAME)

gofmt:
	./scripts/update/gofmt.sh

docs:
	$(CC) run ./scripts/update/docs.go

license:
	./scripts/update/license.sh

check:
	$(CC) test -v ./pkg/...

verify-gofmt:
	./scripts/verify/gofmt.sh

verify-docs:
	./scripts/verify/docs.sh

verify-examples:
	$(CC) build $(CFLAGS) -o example examples/issue.go

verify-license:
	./scripts/verify/license.sh

# Private targets
PKG=.cmd .docs .examples .pkg .scripts
$(PKG): %:
	@# remove the leading '.'
	ineffassign $(subst .,,$@)
	golint -set_exit_status $(subst .,,$@)/...
	misspell -error $(subst .,,$@)

verify-misc: goget $(PKG)

verify: verify-misc verify-gofmt verify-docs verify-license verify-examples

goget:
	@which ineffassign || go get github.com/gordonklaus/ineffassign
	@which golint || go get golang.org/x/lint/golint
	@which misspell || go get github.com/client9/misspell/cmd/misspell

sha512sum: $(NAME)
	$@ ./$^ > $^.$@

$(NAME)-docker:
	docker run --rm --net=host -v $(PWD):/go/src/github.com/JulienBalestra/kube-csr -w /go/src/github.com/JulienBalestra/kube-csr golang:1.10 make

ci-e2e:
	./.ci/e2e.sh

# Everything but the $(NAME) target
.PHONY: clean re gofmt docs license check verify-gofmt verify-docs verify-license verify-misc verify-examples verify sha512sum goget
