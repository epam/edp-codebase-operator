package gitprovider

import "errors"

var (
	ErrWebHookNotFound = errors.New("webhook not found")
	ErrApiNotSupported = errors.New("api is not supported")
)
