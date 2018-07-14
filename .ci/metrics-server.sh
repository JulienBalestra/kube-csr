#!/bin/bash

while true
do
    kubectl get deploy,rs,po,csr -n kube-system -o wide
    kubectl top no && kubectl top po --all-namespaces && exit 0
    echo "===== metrics-server ====="
    kubectl logs -n kube-system -l k8s-app=metrics-server
    echo "===== metrics-server ====="
    sleep 10
done
