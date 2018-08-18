## kube-csr issue

Use this command to generate, approve, fetch and self-delete Kubernetes certificates

### Synopsis

Use this command to generate, approve, fetch and self-delete Kubernetes certificates

```
kube-csr issue [flags]
```

### Examples

```

# generate the private key and the csr
kube-csr issue my-app --generate

# generate the private key, the csr and then submit the csr
kube-csr issue my-app --generate --submit

# generate the private key, the csr, submit and approve the csr
kube-csr issue my-app --generate --submit --approve

# Generate the private key, the csr, submit, approve and fetch the csr
kube-csr issue my-app --generate --submit --approve --fetch
kube-csr issue my-app -gsaf
kube-csr issue my-app -gsaf --subject-alternative-names 192.168.1.1,etcd-0.default.svc.cluster.local

# Generate the private key, the csr, submit and fetch the csr when externally approved
kube-csr issue my-app --generate --submit --fetch --fetch-interval 10s --fetch-timeout 10m

# Generate the private key, the csr, submit, approve, fetch and delete the csr
kube-csr issue my-app --generate --submit --approve --fetch --delete 

# Generate the private key, the csr, submit, approve and fetch the csr. Override any existing and use a kubeconfig
kube-csr issue my-app -gsaf --override --kubeconfig-path ~/.kube/config

# Execute all steps with a custom kubernetes csr name
kube-csr issue skydns --csr-name kv-etcd -gsafd --override --kubeconfig-path ~/.kube/config

```

### Options

```
  -a, --approve                             Approve the CSR
      --certificate-file string             Certificate file target (default "kube-csr.certificate")
      --csr-file string                     Certificate Signing Request file target (default "kube-csr.csr")
      --csr-name string                     Kubernetes CSR name, leave empty for CN-hostname
  -d, --delete                              Delete the given CSR from the kube-apiserver
  -f, --fetch                               Fetch the CSR
      --fetch-interval duration             Polling interval for certificate fetching (default 1s)
      --fetch-timeout duration              Polling timeout for certificate fetching (default 10s)
  -g, --generate                            Generate CSR
  -h, --help                                help for issue
      --hostname string                     Hostname, leave empty to fulfill with hostname
      --load-private-key                    Load the private key file instead of generating one
      --override                            Override any existing file pem and k8s csr resource
      --private-key-file string             Private key file target (default "kube-csr.private_key")
      --query-interval duration             Polling interval for kube-service query (default 2s)
  -q, --query-svc strings                   Query the kube-apiserver services to get additional SAN (namespaceName/serviceName) comma separated
      --query-timeout duration              Polling timeout for kube-service query (default 20s)
      --rsa-bits string                     RSA bits for the private key (default "2048")
      --skip-fetch-annotate                 Skip the update of annotations when successfully fetched the certificate
      --subject-alternative-names strings   Subject Alternative Names (SANs) comma separated
  -s, --submit                              Submit the CSR
```

### Options inherited from parent commands

```
      --kubeconfig-path string   Kubernetes config path, leave empty for inCluster config
  -v, --verbose int              verbose level
```

### SEE ALSO

* [kube-csr](kube-csr.md)	 - Use this command to manage Kubernetes certificates

