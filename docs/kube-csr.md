## kube-csr

Use this command to generate, approve and fetch Kubernetes certificates

### Synopsis

Use this command to generate, approve and fetch Kubernetes certificates

```
kube-csr command line [flags]
```

### Examples

```

# generate the private key and the csr
kube-csr my-app --generate

# generate the private key, the csr and then submit the csr
kube-csr my-app --generate --submit

# generate the private key, the csr, submit and approve the csr
kube-csr my-app --generate --submit --approve

# Generate the private key, the csr, submit, approve and fetch the csr
kube-csr my-app --generate --submit --approve --fetch
kube-csr my-app -gsaf
kube-csr my-app -gsaf --subject-alternative-names 192.168.1.1,etcd-0.default.svc.cluster.local

# Generate the private key, the csr, submit and fetch the csr when externally approved
kube-csr my-app --generate --submit --fetch --fetch-interval 10s --fetch-timeout 10m

# Generate the private key, the csr, submit, approve, fetch and purge the csr
kube-csr my-app --generate --submit --approve --fetch --purge 

# Generate the private key, the csr, submit, approve and fetch the csr. Override any existing and use a kubeconfig
kube-csr my-app -gsaf --override --kubeconfig-path ~/.kube/config

# Execute all steps with a custom kubernetes csr name
kube-csr skydns --csr-name kv-etcd -gsafp --override --kubeconfig-path ~/.kube/config

```

### Options

```
  -a, --approve                             Approve the CSR
      --certificate-file string             Certificate file target (default "kube-csr.certificate")
      --csr-file string                     Certificate Signing Request file target (default "kube-csr.csr")
      --csr-name string                     Kubernetes CSR name, leave empty for CN-hostname
  -f, --fetch                               Fetch the CSR
      --fetch-interval duration             Polling interval for certificate fetching (default 1s)
      --fetch-timeout duration              Polling timeout for certificate fetching (default 10s)
  -g, --generate                            Generate CSR
  -h, --help                                help for kube-csr
      --hostname string                     Hostname, leave empty to fulfill with hostname
      --kubeconfig-path string              Kubernetes config path, leave empty for inCluster config
      --override                            Override any existing file pem and k8s csr resource
      --private-key-file string             Private key file target (default "kube-csr.private_key")
  -p, --purge                               Purge the CSR from the kube-apiserver
      --rsa-bits string                     RSA bits for the private key (default "2048")
      --skip-fetch-annotate                 Skip the update of annotations when succesfully fetched the certificate
      --subject-alternative-names strings   Subject Alternative Names (SANs) comma separated
  -s, --submit                              Submit the CSR
  -v, --verbose int                         verbose level
```

