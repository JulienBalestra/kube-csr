#!/bin/bash

set -exuo pipefail

cd $(dirname $0)/..
rm -fv kube-csr.certificate kube-csr.csr kube-csr.private_key /tmp/foo.certificate /tmp/foo.csr /tmp/foo.private_key existing-key

kubectl get csr

kubectl apply -f examples/metrics-server.yaml
kubectl apply -f examples/etcd.yaml

go run examples/issue.go

echo "=== issue ==="
./kube-csr issue e2e --generate --submit --approve --fetch --kubeconfig-path $HOME/.kube/config
./kube-csr issue e2e --query-svc=default/kubernetes --generate --submit --approve --fetch --override --kubeconfig-path $HOME/.kube/config
./kube-csr issue e2e --query-svc=kubernetes --generate --submit --approve --fetch --override --kubeconfig-path $HOME/.kube/config
kubectl get secret -o json | jq -re '.items[].data["ca.crt"]' | base64 -d > ca.crt
openssl verify -CAfile ca.crt kube-csr.certificate
echo "=== issue ==="

rm -fv kube-csr.certificate kube-csr.csr kube-csr.private_key

echo "=== load-private-key ==="
openssl genrsa 1024 > existing-key
cp -av existing-key kube-csr.private_key
./kube-csr issue existing-key --load-private-key --private-key-file=kube-csr.private_key --query-svc=kubernetes --generate --submit --approve --fetch --kubeconfig-path $HOME/.kube/config
diff existing-key kube-csr.private_key
openssl verify -CAfile ca.crt kube-csr.certificate
echo "=== load-private-key ==="

echo "=== renew ==="
cp -av kube-csr.certificate reference.certificate
./kube-csr issue existing-key \
    --renew --approve \
    --renew-threshold=8760h --renew-check-interval=5s --renew-exit --renew-command "openssl verify -CAfile ca.crt kube-csr.certificate" \
    --kubeconfig-path $HOME/.kube/config
test "$(diff kube-csr.certificate reference.certificate)"
echo "=== renew ==="

timeout 600 ./.ci/etcd.sh

echo "=== garbage-collect ==="
./kube-csr garbage-collect --fetched --grace-period=1h --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --denied --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --expired --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --fetched --grace-period=0s --kubeconfig-path $HOME/.kube/config
./kube-csr garbage-collect --fetched --denied --expired --grace-period=0s --kubeconfig-path $HOME/.kube/config
echo "=== garbage-collect ==="

timeout 600 ./.ci/metrics-server.sh
