#!/usr/bin/env bash

set -exo pipefail

cd $(dirname $0)/..

./kube-csr issue e2e --generate --submit --approve --fetch --kubeconfig-path $HOME/.kube/config
./kube-csr issue e2e --query-svc=default/kubernetes --generate --submit --approve --fetch --override --kubeconfig-path $HOME/.kube/config
./kube-csr issue e2e --query-svc=kubernetes --generate --submit --approve --fetch --override --kubeconfig-path $HOME/.kube/config
kubectl get secret -o json | jq -re '.items[].data["ca.crt"]' | base64 -d > ca.crt
openssl verify -CAfile ca.crt kube-csr.certificate

kubectl apply -f examples/etcd.yaml
timeout 600 ./.ci/etcd.sh

./kube-csr garbage-collect --fetched --grace-period=1h --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --denied --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --fetched --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --fetched --denied --grace-period=0s --kubeconfig-path $HOME/.kube/config
