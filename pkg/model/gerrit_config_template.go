package model

type ConfigGoTemplating struct {
	Lang         string `json:"lang"`
	Name         string
	PlatformType string
	DnsWildcard  string
	Framework    string
	GitURL       string
	// IngressController selects the scaffolded exposure: "nginx" -> Ingress, "envoy" -> HTTPRoute.
	IngressController string
	// GatewayName/GatewayNamespace target the parent Gateway; used only when IngressController is "envoy".
	GatewayName      string
	GatewayNamespace string
}
