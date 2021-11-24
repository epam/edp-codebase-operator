package model

type ConfigGoTemplating struct {
	Lang         string `json:"lang"`
	Name         string
	PlatformType string
	DnsWildcard  string
	Framework    string
	GitURL       string
}
