#!/bin/bash

set -exuo pipefail

while true
do
    kubectl get deploy,rs,po,csr -n kube-system -o wide
    kubectl top no && exit 0
    for po in $(kubectl get po -n kube-system -l k8s-app=metrics-server -o json | jq -re .items[].metadata.name)
    do
        echo "===== metrics-server ====="
        kubectl logs -n kube-system ${po} metrics-server
        echo "===== metrics-server ====="
        echo "===== kube-csr-renew ====="
        kubectl logs -n kube-system ${po} metrics-server
        echo "===== kube-csr-renew ====="
    done
    sleep 10
done
