package empty

import (
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

var log = ctrl.Log.WithName("empty_chain")

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

func (e Chain) ServeRequest(_ *codebaseApi.CodebaseBranch) error {
	if e.returnError {
		err := errors.New(e.logMessage)
		log.Error(err, err.Error())

		return err
	}

	log.Info(e.logMessage)

	return nil
}
