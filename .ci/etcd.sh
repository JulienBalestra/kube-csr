#!/usr/bin/env bash

while true
do
    echo "---"
    kubectl get statefulset,job,po,csr --all-namespaces -o wide
    kubectl get po -n default -o json -l app=etcdctl | jq -re '.items[] | select(.status.phase=="Succeeded")' && exit 0
    sleep 10
done
