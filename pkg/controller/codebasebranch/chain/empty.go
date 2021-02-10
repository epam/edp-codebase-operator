package chain

import (
	"errors"

	edpv1alpha1 "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1alpha1"
)

type Empty struct {
	logMessage  string
	returnError bool
}

func MakeEmptyChain(logMessage string, returnError bool) Empty {
	return Empty{
		logMessage:  logMessage,
		returnError: returnError,
	}
}

func (e Empty) ServeRequest(*edpv1alpha1.CodebaseBranch) error {
	if e.returnError {
		err := errors.New(e.logMessage)
		log.Error(err, err.Error())
		return err
	}

	log.Info(e.logMessage)
	return nil
}
