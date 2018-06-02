package operation

import (
	"github.com/JulienBalestra/kube-csr/pkg/operation/approve"
	"github.com/JulienBalestra/kube-csr/pkg/operation/fetch"
	"github.com/JulienBalestra/kube-csr/pkg/operation/generate"
	"github.com/JulienBalestra/kube-csr/pkg/operation/submit"
)

type Config struct {
	SourceConfig *generate.Config

	Generate *generate.Generator
	Submit   *submit.Submit
	Approve  *approve.Approval
	Fetch    *fetch.Fetch
}

type Operation struct {
	*Config
}

func NewOperation(conf *Config) *Operation {
	return &Operation{
		conf,
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
	o.Approve = nil
	return nil
}

func (o *Operation) Run() error {
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
	if o.Approve != nil {
		err := o.Approve.GetAndApproveCSR(o.SourceConfig.Name)
		if err != nil {
			return err
		}
	}
	if o.Fetch != nil {
		err := o.Fetch.Fetch(o.SourceConfig)
		if err != nil {
			return err
		}
	}
	return nil
}
