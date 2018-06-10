# Asciinema script

```bash
./kube-csr issue etcd --generate --submit --approve --fetch --subject-alternative-names 192.168.1.1,example.com --kubeconfig-path ~/.kube/config
openssl x509 -in kube-csr.certificate -text | grep -B1 example.com
openssl verify -CAfile ca.crt kube-csr.certificate
./kube-csr issue etcd --generate --submit --fetch --subject-alternative-names 192.168.1.1,192.168.1.2,example.com --kubeconfig-path ~/.kube/config --override
# ^Z
kubectl certificate approve etcd-haf
fg
openssl x509 -in kube-csr.certificate -text | grep -B1 example.com
openssl verify -CAfile ca.crt kube-csr.certificate
./kube-csr gc --fetched --grace-period=0s --kubeconfig-path ~/.kube/config
```
