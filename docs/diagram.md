# HOW TO update the diagram


#### Diagram text
```text
kubecsr->KubeApiserver: Get services SAN
Note over kubecsr: Generate:\n* Private Key\n* CSR
kubecsr->KubeApiserver: Submit csr
Note over KubeApiserver: Pending
kubecsr->KubeApiserver: Approve csr
Note over KubeApiserver: Approved
KubeControllerManager-->KubeApiserver: Issue certificate
Note over KubeApiserver: Approved,Issued
kubecsr->KubeApiserver: Fetch certificate
Note left of kubecsr: The csr is annotate as\n*Fetched* if not deleted
kubecsr->KubeApiserver: Delete csr
```

1. open [js-sequence-diagrams](https://bramp.github.io/js-sequence-diagrams/)
2. copy/paste the [diagram text](#diagram-text) in the demo text box
3. try some changes
4. *download as SVG*
5. replace the changes in the [diagram text](#diagram-text) section
6. replace the [diagram](docs/diagram.svg)
