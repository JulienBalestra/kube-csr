CC=go
CFLAGS?=-i
GOOS=linux
CGO_ENABLED?=0

NAME=kube-csr

$(NAME):
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) $(CC) build $(CFLAGS) -o $@ cmd/main.go

clean:
	$(RM) $(NAME)
	$(RM) $(NAME).sha512sum

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

verify-license:
	./scripts/verify/license.sh

verify: verify-gofmt verify-docs verify-license

sha512sum: $(NAME)
	$@ ./$^ > $^.$@

# Everything but the $(NAME) target
.PHONY: clean re gofmt docs license check verify-gofmt verify-docs verify-license verify sha512sum