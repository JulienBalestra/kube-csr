- [v0.5.0](#v0.5.0)
- [v0.4.0](#v0.4.0)
- [v0.3.0](#v0.3.0)
- [v0.2.0](#v0.2.0)

## v0.5.0

### Features
* Introduce the renew feature #54
* Move the main to the root #48
* Add logging messages in the operation package #45

### Bugfix
* Dedup and sort SAN, validate all DNS names #51

### Other
* ci: bump p8s #55
* Upgrade dep to 0.5 #52
* Metrics-server is self registered #50
* ci: bump p8s to 0.7.0 #46
* Add the metrics-server as example #44
* example: add the query, test the issue.go in e2e #43

## v0.4.0

### Features
* Add an expired purge capability #40
* Include Kubernetes services DNS and IPs in SAN #41

### Other
* Add unittests in generate and pemio #39
* Improve ci #38
* Upgrade to p8s 0.6.1 #35

## v0.3.0

### Features
* Add a pprof http handler #31
* Add a prometheus exporter handler #23

### Bugfix
* tickers: must be non zero #22

### Other
* Create a ToC #30
* Add hyperkube version 1.11.0 #29
* Add hyperkube versions #26
* Add travis status badge #25
* Introduce pupernetes as e2e testing engine. #24
* readme: update the diagram, replace purge by delete #21
* Update the diagram and add the gc to the docs #20

## v0.2.0

### Features
* Introduce the Garbage Collect command #19
* Create a delete flag #15
* Be able to skip the fetch annotation updates #13
* Set annotations to track the activity of fetch #12

### Bugfix
* Build from the current working directory #18
* Fetch fail on denied #11

### Other
* Add a probe to the etcd example #17
* Add a misspel tool #16
* Create a diagram on the README #14
