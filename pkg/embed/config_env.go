package embed

import (
	"github.com/flyleft/gprofile"
)

var Env *Config

func init() {
	env, err := gprofile.Profile(&Config{}, "static/application.yml", true)
	if err != nil {
		panic(err)
	}
	Env = env.(*Config)
	if err != nil {
		panic(err)
	}
}

type Config struct {
	RegisterServiceNamespace []string     `profile:"register.service.namespace"`
	RegisterServerNamespace  string       `profile:"register.server.namespace"`
	ConfigServer             ConfigServer `profile:"config.server"`
	Kubeconfig               string       `profile:"kubeconfig" profileDefault:""`
}

type ConfigServer struct {
	Enabled      bool     `profileDefault:"true"`
	GatewayNames []string `profile:"gateway.names" profileDefault:"[\"api-gateway\", \"gateway-helper\"]"`
}
