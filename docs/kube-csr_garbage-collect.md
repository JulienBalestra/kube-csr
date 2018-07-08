## kube-csr garbage-collect

Garbage collect Kubernetes certificates on different parameters

### Synopsis

Garbage collect Kubernetes certificates on different parameters

```
kube-csr garbage-collect [flags]
```

### Examples

```

# Garbage collect all csr already fetched with a grace period of 12 hours
kube-csr gc --fetched --grace-period=12h

# Garbage collect all csr denied with a grace period of 15 minutes
kube-csr gc --fetched --grace-period=15m

# Garbage collect now all csr already fetched
kube-csr gc --fetched --grace-period=0s

# Garbage collect every 10min all csr already fetched with a grace period of 1 hour
kube-csr gc --fetched --daemon polling-period=10m --grace-period=1h

```

### Options

```
      --daemon                        continually gc Kubernetes csr, paired with --polling-period
      --denied                        delete any denied Kubernetes csr
      --disable-prometheus-exporter   disable /metrics, paired with --daemon
      --expired                       delete any Kubernetes csr with an expired certificate
      --fetched                       delete any already fetched Kubernetes csr, the state is tracked with kube-annotations "alpha.kube-csr/"
      --grace-period duration         duration to wait before deleting Kubernetes csr objects (default 48h0m0s)
  -h, --help                          help for garbage-collect
      --polling-period duration       duration to wait between each gc call, paired with --daemon (default 10m0s)
      --prometheus-exporter-bind      prometheus exporter bind address, paired with --daemon
```

### Options inherited from parent commands

```
      --kubeconfig-path string   Kubernetes config path, leave empty for inCluster config
  -v, --verbose int              verbose level
```

### SEE ALSO

* [kube-csr](kube-csr.md)	 - Use this command to manage Kubernetes certificates

