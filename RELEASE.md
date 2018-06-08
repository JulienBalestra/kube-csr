# Release process

### Submit a PR

Updated master branch:
```bash
git checkout master
git pull
```

Clean:
```bash
git clean . -fdx -e .idea
make clean
```

Compile statically the binary and generate the sha512sum:
```bash
CGO_ENABLED=0 make sha512sum

# or using docker
docker run --rm -v "$GOPATH":/go -w /go/src/github.com/JulienBalestra/kube-csr golang:1.10 make sha512sum
```

Check the shared object dependencies:
```bash
ldd kube-csr
	not a dynamic executable
echo $?
1

# or using docker
docker run --rm -v "$GOPATH":/go -w /go/src/github.com/JulienBalestra/kube-csr golang:1.10 sh -c 'ldd kube-csr ; echo $?'
	not a dynamic executable
1
```

Check the sha512sum:
```bash
sha512sum -c kube-csr.sha512sum 
./kube-csr: OK
```


### Push tags

After validation, merge your PR and checkout the latest master branch:
```bash
git checkout master
git pull
```

Tag the release with the new version (e.g. v0.3.0):
```bash
git tag v0.3.0
git push --tags
```

kube-csr follows Semantic Versionning ([_SemVer_](https://semver.org/)):
```text
# Patch update
0.3.0 -> 0.3.1

# Minor update
0.3.0 -> 0.4.0
0.9.0 -> 0.10.0

# Major update
0.3.0 -> 1.0.0
1.15.4 -> 2.0.0
...
```

Then upload `kube-csr` + `kube-csr.sha512sum` in the release page.

The release must be marked as pre-release.
