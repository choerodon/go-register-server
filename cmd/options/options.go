package options

import (
	"github.com/spf13/pflag"

	"github.com/choerodon/go-register-server/pkg/api/server"
)

// ServerRunOptions runs a server
type ServerRunOptions struct {
	RegisterServerOptions *server.RegisterServerOptions
	MasterURL             string
	KubeConfig            string
}

func NewServerRunOptions() *ServerRunOptions {
	s := &ServerRunOptions{
		RegisterServerOptions: server.NewRegisterServerOptions(),
	}

	return s
}

func (s *ServerRunOptions) AddFlag(fs *pflag.FlagSet) {
	fs.StringVar(&s.MasterURL, "master", "",
		"The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	fs.StringVar(&s.KubeConfig, "kubeconfig", "",
		"Path to a kubeconfig. Only required if out-of-cluster.")
}
