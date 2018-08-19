package operation

import (
	"github.com/golang/glog"

	"github.com/JulienBalestra/kube-csr/pkg/operation/approve"
	"github.com/JulienBalestra/kube-csr/pkg/operation/fetch"
	"github.com/JulienBalestra/kube-csr/pkg/operation/generate"
	"github.com/JulienBalestra/kube-csr/pkg/operation/purge"
	"github.com/JulienBalestra/kube-csr/pkg/operation/query"
	"github.com/JulienBalestra/kube-csr/pkg/operation/submit"
)

// Config an operation
type Config struct {
	SourceConfig *generate.Config

	Query    *query.Query
	Generate *generate.Generator
	Submit   *submit.Submit
	Approve  *approve.Approval
	Fetch    *fetch.Fetch
	Purge    *purge.Purge
}

// Operation state
type Operation struct {
	*Config

	approved bool
}

// NewOperation instantiate an Operation to potentially
// - query
// - generate
// - submit
// - approve
// - fetch
// certificates through the kubernetes API.
func NewOperation(conf *Config) *Operation {
	return &Operation{
		Config: conf,
	}
}

func (o *Operation) submit() error {
	r, err := o.Submit.Submit(o.SourceConfig)
	if err != nil {
		return err
	}
	if o.Approve == nil {
		return nil
	}
	err = o.Approve.ApproveCSR(r)
	if err != nil {
		return err
	}
	o.approved = true
	return nil
}

// Run executes all the configured operations
func (o *Operation) Run() error {
	glog.V(0).Infof("Running operations ...")
	o.approved = false
	if o.Query != nil {
		sans, err := o.Query.GetKubernetesServicesSubjectAlternativeNames()
		if err != nil {
			return err
		}
		o.SourceConfig.Hosts = append(o.SourceConfig.Hosts, sans...)
	}
	if o.Generate != nil {
		err := o.Generate.Generate()
		if err != nil {
			return err
		}
	}
	if o.Submit != nil {
		err := o.submit()
		if err != nil {
			return err
		}
	}
	if o.Approve != nil && !o.approved {
		err := o.Approve.GetAndApproveCSR(o.SourceConfig.Name)
		if err != nil {
			return err
		}
	}
	if o.Fetch != nil {
		err := o.Fetch.Fetch(o.SourceConfig.Name)
		if err != nil {
			return err
		}
	}
	if o.Purge != nil {
		err := o.Purge.Delete(o.SourceConfig.Name)
		if err != nil {
			return err
		}
	}
	glog.V(0).Infof("Successfully finished operations")
	return nil
}
