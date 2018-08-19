// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018 Datadog, Inc.

package main

import (
	"flag"
	"fmt"
	"github.com/JulienBalestra/kube-csr/pkg/renew"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"os"
	"path"
	"sort"
)

func init() {
	flag.CommandLine.Parse([]string{})
	flag.Lookup("alsologtostderr").Value.Set("true")
	flag.Lookup("v").Value.Set("2")
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		glog.Exitln(err)
	}
	docDir := path.Join(cwd, "docs")
	_, err = os.Stat(docDir)
	if err != nil {
		glog.Exitf("Cannot create markdown in %s", docDir)
	}

	var metricsToWrite []string
	// renew
	err = renew.RegisterPrometheusMetrics(&renew.Renew{})
	if err != nil {
		glog.Exitf("%s", err)
	}
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		glog.Exitf("%s", err)
	}
	metricsToWrite = []string{}
	for _, m := range metrics {
		metricsToWrite = append(metricsToWrite, fmt.Sprintf("%q,%q,%q\n", m.GetName(), m.GetType(), m.GetHelp()))
	}
	metricFile, err := os.OpenFile(path.Join(docDir, "renew-metrics.csv"), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		glog.Exitf("%s", err)
	}
	defer metricFile.Close()
	metricFile.WriteString("name,type,help\n")
	sort.Strings(metricsToWrite)
	for _, elt := range metricsToWrite {
		metricFile.WriteString(elt)
	}
	metricFile.Sync()
	glog.Infof("Generated metrics file in %s", metricFile.Name())
}
