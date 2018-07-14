#!/bin/bash

set -exuo pipefail

cd $(dirname $0)/..

REQUEST_HEADER_CLIENT_CA=$(kubectl get cm -n kube-system extension-apiserver-authentication -o json | jq '.data["requestheader-client-ca-file"]' -re | base64 | tr -d '\n')
sed s/REQUEST_HEADER_CLIENT_CA/"${REQUEST_HEADER_CLIENT_CA}"/g examples/metrics-server.yaml > ./.ci/metrics-server.yaml
kubectl apply -f ./.ci/metrics-server.yaml
kubectl apply -f examples/etcd.yaml

go run examples/issue.go

./kube-csr issue e2e --generate --submit --approve --fetch --kubeconfig-path $HOME/.kube/config
./kube-csr issue e2e --query-svc=default/kubernetes --generate --submit --approve --fetch --override --kubeconfig-path $HOME/.kube/config
./kube-csr issue e2e --query-svc=kubernetes --generate --submit --approve --fetch --override --kubeconfig-path $HOME/.kube/config
kubectl get secret -o json | jq -re '.items[].data["ca.crt"]' | base64 -d > ca.crt
openssl verify -CAfile ca.crt kube-csr.certificate

timeout 600 ./.ci/etcd.sh

./kube-csr garbage-collect --fetched --grace-period=1h --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --denied --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --expired --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --fetched --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --fetched --denied --expired --grace-period=0s --kubeconfig-path $HOME/.kube/config

timeout 600 ./.ci/metrics-server.sh
