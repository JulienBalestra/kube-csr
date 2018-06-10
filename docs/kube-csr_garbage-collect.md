## kube-csr garbage-collect

Garbage collect Kubernetes certificates on different parameters

### Synopsis

Garbage collect Kubernetes certificates on different parameters

```
kube-csr garbage-collect [flags]
```

### Examples

```

# Garbage collect all csr already fetched since 12 hours
kube-csr gc --fetched --grace-period=12h

# Garbage collect all csr denied over 15 minutes
kube-csr gc --fetched --grace-period=15m

# Garbage collect all csr denied and already fetched
kube-csr gc --fetched --grace-period=0s

```

### Options

```
      --denied                  delete any denied Kubernetes csr
      --fetched                 delete any already fetched Kubernetes csr, the state is tracked with kube-annotations "alpha.kube-csr/"
      --grace-period duration   duration to wait before deleting Kubernetes csr objects (default 48h0m0s)
  -h, --help                    help for garbage-collect
```

### Options inherited from parent commands

```
      --kubeconfig-path string   Kubernetes config path, leave empty for inCluster config
  -v, --verbose int              verbose level
```

### SEE ALSO

* [kube-csr](kube-csr.md)	 - Use this command to manage Kubernetes certificates

