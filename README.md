# Kubernetes Certificate Signing Request 

[![CircleCI](https://circleci.com/gh/JulienBalestra/kube-csr.svg?style=svg)](https://circleci.com/gh/JulienBalestra/kube-csr) [![Go Report Card](https://goreportcard.com/badge/github.com/JulienBalestra/kube-csr)](https://goreportcard.com/report/github.com/JulienBalestra/kube-csr) [![Docker Repository on Quay](https://quay.io/repository/julienbalestra/kube-csr/status "Docker Repository on Quay")](https://quay.io/repository/julienbalestra/kube-csr)

All in one:
* generate
    * Private Key - **stay on disk**
    * Certificate Signing Request (CSR)
* submit the generated CSR
* approve the submitted CSR
* fetch the generated certificate
* purge the kubernetes csr resource


But you can also choose to select the steps you want to execute.

For example, you can do the following actions:
* generate the PK, CSR
* submit the generated CSR
* <s>approve the submitted CSR</s>
* fetch the generated **externally approved** certificate
* <s>purge the kubernetes csr resource</s>

![diagram](docs/diagram.svg)

[![asciicast](https://asciinema.org/a/sjcTvHmsdwFNPxZ9TGELrHK53.png)](https://asciinema.org/a/sjcTvHmsdwFNPxZ9TGELrHK53)

## Docker image

Available at `quay.io/julienbalestra/kube-csr:latest`

The tag `latest` is up to date with master.

Please, have a look to the [release  page](https://github.com/JulienBalestra/kube-csr/releases) to get a more stable image tag.

## Command line

Command line example:
```text
$ ./kube-csr etcd --generate --submit --approve --fetch --subject-alternative-names 192.168.1.1,etcd-0.default.svc.cluster.local --kubeconfig-path ~/.kube/config

I0602 22:48:12.880405    5241 generate.go:56] Added IP address 192.168.1.1
I0602 22:48:12.880429    5241 generate.go:61] Added DNS name etcd-0.default.svc.cluster.local
I0602 22:48:12.880433    5241 generate.go:70] CSR with 1 DNS names and 1 IP addresses
I0602 22:48:12.880439    5241 generate.go:91] Generating CSR with CN=etcd
I0602 22:48:12.883018    5241 write.go:54] Wrote RSA PRIVATE KEY to /home/jb/go/src/github.com/JulienBalestra/kube-csr/kube-csr.private_key
I0602 22:48:12.883069    5241 write.go:54] Wrote CERTIFICATE REQUEST to /home/jb/go/src/github.com/JulienBalestra/kube-csr/kube-csr.csr
I0602 22:48:12.893741    5241 submit.go:87] Successfully created csr/etcd-haf 4428adb4-66a6-11e8-94af-5404a66983a9
I0602 22:48:12.893759    5241 approve.go:34] Approving csr/etcd-haf ...
I0602 22:48:12.895857    5241 approve.go:45] csr/etcd-haf is approved
I0602 22:48:12.895953    5241 fetch.go:43] Start polling for certificate of csr/etcd-haf, every 1s, timeout after 10s
I0602 22:48:13.898763    5241 fetch.go:66] Certificate successfully fetched, writing 1216 chars to /home/jb/go/src/github.com/JulienBalestra/kube-csr/kube-csr.certificate
```

The `--override` flag allows to delete and re-submit an already submitted CSR.

To get the following files:
```text
kube-csr.certificate kube-csr.csr kube-csr.private_key
```

```text
$ openssl x509 -in kube-csr.certificate -text -noout

Certificate:
        Issuer: CN = p8s
        Subject: CN = etcd
            X509v3 Subject Alternative Name: 
                DNS:etcd-0.default.svc.cluster.local, IP Address:192.168.1.1
```

Observe in the controller-manager logs:
```text
$ kubectl logs po/kube-controller-manager -n kube-system

[INFO] signed certificate with serial number [...]
```

Have a look the the command line documentation [here](docs/kube-csr.md)

The current Kubernetes setup is deployed with [pupernetes](https://github.com/DataDog/pupernetes)

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
kube-system   kube-apiserver-haf         1/1       Running   0          9s
kube-system   kube-controller-manager    1/1       Running   0          1m
kube-system   kube-proxy-2z9vw           1/1       Running   0          1m
kube-system   kube-scheduler-v8lwc       1/1       Running   0          1m
``` 

Apply the manifests:
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
$ kubectl get csr,po --show-all

NAME                                              AGE       REQUESTOR                            CONDITION
etcd-0-b5722b13-66a1-11e8-94af-5404a66983a9   36m       system:serviceaccount:default:etcd   Approved,Issued
etcd-1-b87e7bc2-66a1-11e8-94af-5404a66983a9   36m       system:serviceaccount:default:etcd   Approved,Issued
etcd-2-bbac1ba4-66a1-11e8-94af-5404a66983a9   36m       system:serviceaccount:default:etcd   Approved,Issued


NAME            READY     STATUS      RESTARTS   AGE
etcd-0          1/1       Running     0          35m
etcd-1          1/1       Running     0          35m
etcd-2          1/1       Running     0          35m
etcdctl-hkp25   0/1       Completed   0          35m
```

Observe the logs of the init container kube-csr:
```text
$ kubectl logs etcd-0 kube-csr

I0602 20:15:36.844699       1 generate.go:56] Added IP address 172.17.0.3
I0602 20:15:36.845018       1 generate.go:61] Added DNS name etcd-0.etcd.default.svc.cluster.local
I0602 20:15:36.845025       1 generate.go:61] Added DNS name etcd.default.svc.cluster.local
I0602 20:15:36.845029       1 generate.go:70] CSR with 2 DNS names and 1 IP addresses
I0602 20:15:36.845033       1 generate.go:91] Generating CSR with CN=etcd-0
I0602 20:15:36.847608       1 write.go:54] Wrote RSA PRIVATE KEY to /etc/certs/etcd.private_key
I0602 20:15:36.847657       1 write.go:54] Wrote CERTIFICATE REQUEST to /etc/certs/etcd.csr
I0602 20:15:36.857854       1 submit.go:88] Successfully created csr/etcd-0-b5722b13-66a1-11e8-94af-5404a66983a9 b6453f62-66a1-11e8-94af-5404a66983a9
I0602 20:15:36.857873       1 approve.go:34] Approving csr/etcd-0-b5722b13-66a1-11e8-94af-5404a66983a9 ...
I0602 20:15:36.860071       1 approve.go:45] csr/etcd-0-b5722b13-66a1-11e8-94af-5404a66983a9 is approved
I0602 20:15:36.860089       1 fetch.go:43] Start polling for certificate of csr/etcd-0-b5722b13-66a1-11e8-94af-5404a66983a9, every 1s, timeout after 10s
I0602 20:15:37.862585       1 fetch.go:66] Certificate successfully fetched, writing 1241 chars to /etc/certs/etcd.certificate
```

See the output of the completed Job:
```text
$ kubectl logs etcdctl-${ID}

[...]
member 6c254b8f2d60eb6a is healthy: got healthy result from https://etcd-2.etcd.default.svc.cluster.local:2379
member 6ca04f5d282b7cd5 is healthy: got healthy result from https://etcd-0.etcd.default.svc.cluster.local:2379
member 891a4cb0531d4224 is healthy: got healthy result from https://etcd-1.etcd.default.svc.cluster.local:2379
cluster is healthy
```

## Library

Please see an example to use *kube-csr* as library [here](examples/example.go)

```bash
go get github.com/JulienBalestra/kube-csr
```