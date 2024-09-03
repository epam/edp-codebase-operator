package model

type UserSettings struct {
	DnsWildcard string `json:"dns_wildcard"`
}

const (
	StatusInit       = "initialized"
	StatusInProgress = "in progress"
	StatusFailed     = "failed"
	StatusFinished   = "created"
)
