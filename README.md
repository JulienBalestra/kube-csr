# Kubernetes Certificate Signing Request 

All in one:
* generate a Certificate Signing Request (CSR)
* submit the generated CSR
* approve the submitted CSR
* fetch the generated certificate


But you can also choose to skip some steps like:
* generate the CSR
* submit the generated CSR
* <s>approve the submitted CSR</s>
* fetch the generated **externally approved** certificate

The `--override` flag allows to delete and re-submit an already submitted CSR.

Command line example:
```text
$ ./kube-csr etcd --generate --submit --approve --fetch --subject-alternative-names 192.168.1.1,etcd-0.default.svc.cluster.local --kubeconfig-path ~/.kube/config
 
I0527 19:39:29.542878   30219 csr.go:47] Added IP address 192.168.1.1
I0527 19:39:29.543072   30219 csr.go:52] Added DNS name etcd-0.default.svc.cluster.local
I0527 19:39:29.543078   30219 csr.go:61] CSR with 1 DNS names and 1 IP addresses
I0527 19:39:29.545655   30219 write.go:54] Wrote RSA PRIVATE KEY to /home/jb/go/src/github.com/JulienBalestra/kube-csr/kube-csr.private_key
I0527 19:39:29.545690   30219 write.go:54] Wrote CERTIFICATE REQUEST to /home/jb/go/src/github.com/JulienBalestra/kube-csr/kube-csr.csr
I0527 19:39:29.545713   30219 kubeclient.go:38] Building flags kube-config with /home/jb/.kube/config
I0527 19:39:29.559138   30219 submit.go:80] Successfully created csr/etcd-haf
I0527 19:39:29.559154   30219 submit.go:88] Approving csr/etcd-haf ...
I0527 19:39:29.562221   30219 submit.go:99] csr/etcd-haf is approved
I0527 19:39:29.562262   30219 kubeclient.go:38] Building flags kube-config with /home/jb/.kube/config
I0527 19:39:29.562837   30219 fetch.go:37] Start polling for certificate of csr/etcd-haf, every 1s, timeout after 10s
I0527 19:39:30.565534   30219 fetch.go:60] Certificate successfully fetched, writing 1216 chars to /home/jb/go/src/github.com/JulienBalestra/kube-csr/kube-csr.certificate
```

To get the following files:
```text
kube-csr.certificate  kube-csr.csr  kube-csr.private_key
```

```text
$ openssl x509 -in kube-csr.certificate -text -noout

Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number:
            03:56:10:1b:4f:ce:42:3d:40:ab:e4:30:be:41:42:2a:a3:10:4e:5f
    Signature Algorithm: sha256WithRSAEncryption
        Issuer: CN = p8s
        Validity
            Not Before: May 27 17:42:00 2018 GMT
            Not After : May 27 17:42:00 2019 GMT
        Subject: CN = etcd
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                Public-Key: (2048 bit)
                Modulus:
                    [...]
                Exponent: 65537 (0x10001)
        X509v3 extensions:
            X509v3 Key Usage: critical
                Digital Signature, Key Encipherment
            X509v3 Extended Key Usage: 
                TLS Web Server Authentication
            X509v3 Basic Constraints: critical
                CA:FALSE
            X509v3 Subject Key Identifier: 
                F2:BC:3E:83:30:3D:92:55:35:A4:88:48:97:1D:3F:AA:CF:DA:E2:F4
            X509v3 Authority Key Identifier: 
                keyid:4C:10:E2:7D:3A:22:EC:5A:34:54:69:8C:83:F9:20:22:7E:12:AF:38

            X509v3 Subject Alternative Name: 
                DNS:etcd-0.default.svc.cluster.local, IP Address:192.168.1.1
    Signature Algorithm: sha256WithRSAEncryption
         [...]

```

Observe in the controller-manager logs:
```text
$ kubectl logs po/kube-controller-manager -n kube-system

[...]
I0527 17:47:07.118840       1 logs.go:49] [INFO] signed certificate with serial number 19046239489823935503989490272030062933187776095
```

Have a look the the command line documentation [here](docs/kube-csr.md)

### In cluster

In this example, the all in one [etcd](examples/etcd.yaml) example is used.

Current cluster:
```text
$ kubectl get svc,deploy,ds,po --all-namespaces

NAMESPACE     NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)         AGE
default       kubernetes   ClusterIP   192.168.254.1   <none>        443/TCP         1m
kube-system   coredns      ClusterIP   192.168.254.2   <none>        53/UDP,53/TCP   1m

NAMESPACE     NAME      DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
kube-system   coredns   1         1         1            1           1m

NAMESPACE     NAME             DESIRED   CURRENT   READY     UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
kube-system   kube-proxy       1         1         1         1            1           <none>          1m
kube-system   kube-scheduler   1         1         1         1            1           <none>          1m

NAMESPACE     NAME                       READY     STATUS    RESTARTS   AGE
kube-system   coredns-747dbcf5df-zllhm   1/1       Running   0          1m
kube-system   kube-apiserver-v1704       1/1       Running   0          9s
kube-system   kube-controller-manager    1/1       Running   0          1m
kube-system   kube-proxy-2z9vw           1/1       Running   0          1m
kube-system   kube-scheduler-v8lwc       1/1       Running   0          1m

``` 

Deployment:
```text
$ kubectl apply -f examples/etcd.yaml 

serviceaccount "etcd" created
clusterrole.rbac.authorization.k8s.io "system:etcd" created
clusterrolebinding.rbac.authorization.k8s.io "system:etcd" created
statefulset.apps "etcd" created
job.batch "etcdctl" created
service "etcd" created
```

Produce:
```text
$ kubectl get po,csr

NAME            READY     STATUS      RESTARTS   AGE
etcd-0          1/1       Running     0          3m
etcdctl-fbcdj   0/1       Completed   0          3m

NAME          AGE       REQUESTOR                            CONDITION
etcd-etcd-0   3m        system:serviceaccount:default:etcd   Approved,Issued
```

Observe in detail the init container of etcd-0:
```text
$ kubectl logs etcd-0 kube-csr

I0529 09:08:14.553994       1 csr.go:46] Added IP address 172.17.0.3
I0529 09:08:14.554551       1 csr.go:51] Added DNS name etcd-0.etcd.svc.cluster.local
I0529 09:08:14.554602       1 csr.go:51] Added DNS name etcd.default.svc.cluster.local
I0529 09:08:14.554637       1 csr.go:60] CSR with 2 DNS names and 1 IP addresses
I0529 09:08:14.557671       1 write.go:54] Wrote RSA PRIVATE KEY to /etc/certs/etcd.private_key
I0529 09:08:14.557793       1 write.go:54] Wrote CERTIFICATE REQUEST to /etc/certs/etcd.csr
I0529 09:08:14.557844       1 kubeclient.go:27] Building inCluster kube-config
I0529 09:08:14.570322       1 submit.go:80] Successfully created csr/etcd-etcd-0
I0529 09:08:14.570422       1 submit.go:88] Approving csr/etcd-etcd-0 ...
I0529 09:08:14.573511       1 submit.go:99] csr/etcd-etcd-0 is approved
I0529 09:08:14.573612       1 kubeclient.go:27] Building inCluster kube-config
I0529 09:08:14.573865       1 fetch.go:37] Start polling for certificate of csr/etcd-etcd-0, every 1s, timeout after 10s
I0529 09:08:15.580527       1 fetch.go:60] Certificate successfully fetched, writing 1257 chars to /etc/certs/etcd.certificate
```

Confirm the etcd is running:
```text
$ kubectl logs etcd-0

[...]
2018-05-29 09:08:18.938102 I | embed: listening for peers on http://localhost:2380
2018-05-29 09:08:18.938182 I | embed: listening for client requests on 0.0.0.0:2379
[...]
2018-05-29 09:08:18.961374 I | etcdserver: advertise client URLs = https://172.17.0.3:2379
2018-05-29 09:08:18.961379 I | etcdserver: initial advertise peer URLs = http://localhost:2380
2018-05-29 09:08:18.961387 I | etcdserver: initial cluster = default=http://localhost:2380
[...]
2018-05-29 09:08:18.994836 I | embed: ClientTLS: cert = /etc/certs/etcd.certificate, key = /etc/certs/etcd.private_key, ca = , trusted-ca = /run/secrets/kubernetes.io/serviceaccount/ca.crt, client-cert-auth = false, crl-file = 
[...]
2018-05-29 09:08:19.474273 I | etcdserver: published {Name:default ClientURLs:[https://172.17.0.3:2379]} to cluster cdf818194e3a8c32
2018-05-29 09:08:19.474295 I | embed: ready to serve client requests
2018-05-29 09:08:19.478965 I | embed: serving client requests on [::]:2379
```

See the output of the completed Job:
```text
$ kubectl logs etcdctl-fbcdj

cluster may be unhealthy: failed to list members
Error:  client: etcd cluster is unavailable or misconfigured; error #0: client: endpoint https://etcd.default.svc.cluster.local:2379 exceeded header timeout
error #0: client: endpoint https://etcd.default.svc.cluster.local:2379 exceeded header timeout
---
member 8e9e05c52164694d is healthy: got healthy result from https://172.17.0.3:2379
cluster is healthy
```
