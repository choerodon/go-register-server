package embed

import (
	"encoding/json"
	"fmt"
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
	printConfig, _ := json.MarshalIndent(Env, "", "  ")
	fmt.Printf("Application config: %s", printConfig)
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
	Log          bool     `profileDefault:"false"`
}

func (config Config) IsRegisterServiceNamespace(ns string) bool {
	for _, n := range config.RegisterServiceNamespace {
		if n == ns {
			return true
		}
	}
	return false
}
