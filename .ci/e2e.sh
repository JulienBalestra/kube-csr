#!/bin/bash

set -exuo pipefail

cd $(dirname $0)/..
rm -fv kube-csr.certificate kube-csr.csr kube-csr.private_key /tmp/foo.certificate /tmp/foo.csr /tmp/foo.private_key existing-key

kubectl get csr

kubectl apply -f examples/metrics-server.yaml
kubectl apply -f examples/etcd.yaml

go run examples/issue.go

./kube-csr issue e2e --generate --submit --approve --fetch --kubeconfig-path $HOME/.kube/config
./kube-csr issue e2e --query-svc=default/kubernetes --generate --submit --approve --fetch --override --kubeconfig-path $HOME/.kube/config
./kube-csr issue e2e --query-svc=kubernetes --generate --submit --approve --fetch --override --kubeconfig-path $HOME/.kube/config
kubectl get secret -o json | jq -re '.items[].data["ca.crt"]' | base64 -d > ca.crt
openssl verify -CAfile ca.crt kube-csr.certificate

rm -fv kube-csr.certificate kube-csr.csr kube-csr.private_key

openssl genrsa 2048 > existing-key
cp -av existing-key kube-csr.private_key
./kube-csr issue existing-key --load-private-key --private-key-file=kube-csr.private_key --query-svc=kubernetes --generate --submit --approve --fetch --kubeconfig-path $HOME/.kube/config
diff existing-key kube-csr.private_key
openssl verify -CAfile ca.crt kube-csr.certificate

timeout 600 ./.ci/etcd.sh

./kube-csr garbage-collect --fetched --grace-period=1h --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --denied --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --expired --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --fetched --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --fetched --denied --expired --grace-period=0s --kubeconfig-path $HOME/.kube/config

timeout 600 ./.ci/metrics-server.sh
