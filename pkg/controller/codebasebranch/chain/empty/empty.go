package empty

import (
	"errors"

	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("empty_chain")

type Chain struct {
	logMessage  string
	returnError bool
}

func MakeChain(logMessage string, returnError bool) Chain {
	return Chain{
		logMessage:  logMessage,
		returnError: returnError,
	}
}

func (e Chain) ServeRequest(*edpv1alpha1.CodebaseBranch) error {
	if e.returnError {
		err := errors.New(e.logMessage)
		log.Error(err, err.Error())
		return err
	}

	log.Info(e.logMessage)
	return nil
}
