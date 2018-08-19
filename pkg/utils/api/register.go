package api

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"net/http/pprof"
	"time"
)

const (
	// PprofBindDefault is the default ip:port for the pprof listener
	PprofBindDefault       = "127.0.0.1:6060"
	prometheusExporterPath = "/metrics"
)

// RegisterAPI register:
// - prometheus exporter
// - pprof
func RegisterAPI(prometheusExporterBindAddress, pprofBind string) {
	if prometheusExporterBindAddress != "" {
		promRouter := mux.NewRouter()
		promRouter.Path(prometheusExporterPath).Methods("GET").Handler(promhttp.Handler())
		promServer := &http.Server{
			Handler:      promRouter,
			Addr:         prometheusExporterBindAddress,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		}
		glog.V(0).Infof("Starting prometheus exporter on %s%s", prometheusExporterBindAddress, prometheusExporterPath)
		go promServer.ListenAndServe()
	}

	// Known issue with Mux and the registering of pprof:
	// https://stackoverflow.com/questions/19591065/profiling-go-web-application-built-with-gorillas-mux-with-net-http-pprof
	pprofRouter := mux.NewRouter()
	pprofRouter.HandleFunc("/debug/pprof/", pprof.Index)
	pprofRouter.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofRouter.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofRouter.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofRouter.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	pprofServer := &http.Server{
		Handler:      pprofRouter,
		Addr:         pprofBind,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  60 * time.Second,
	}
	glog.V(0).Infof("Starting pprof on %s/debug/pprof", pprofBind)
	go pprofServer.ListenAndServe()
}
