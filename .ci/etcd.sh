#!/bin/bash

while true
do
    kubectl get statefulset,job,po,csr -n default -o wide
    kubectl get po -n default -o json -l app=etcdctl | jq -re '.items[] | select(.status.phase=="Succeeded")' && exit 0
    for i in {0..2}
    do
        echo "===== etcd-${i} kube-csr ====="
        kubectl logs -n default etcd-${i} kube-csr
        echo "===== etcd-${i} kube-csr ====="
    done
    sleep 10
done
