package cmd

import (
	"fmt"
	"github.com/choerodon/go-register-server/pkg/embed"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	kubeInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/choerodon/go-register-server/cmd/options"
	"github.com/choerodon/go-register-server/pkg/api/repository"
	"github.com/choerodon/go-register-server/pkg/api/server"
	"github.com/choerodon/go-register-server/pkg/k8s"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func NewServerCommand() *cobra.Command {
	s := options.NewServerRunOptions()
	cmd := &cobra.Command{
		Use:  "register-server",
		Long: `Register Server`,
		Run: func(cmd *cobra.Command, args []string) {
			stopCh := signals.SetupSignalHandler()

			if err := Run(s, stopCh); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	s.AddFlag(cmd.Flags())

	return cmd
}

func Run(s *options.ServerRunOptions, stopCh <-chan struct{}) error {
	k8s.AppRepo = repository.NewApplicationRepository()

	registerServer := server.CreateRegisterServer(s.RegisterServerOptions)

	cfg, err := clientcmd.BuildConfigFromFlags(s.MasterURL, s.KubeConfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	k8s.KubeClient, err = kubernetes.NewForConfig(cfg)

	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	k8s.KubeInformerFactory = kubeInformers.NewSharedInformerFactory(k8s.KubeClient, time.Second*30)

	if embed.Env.ConfigServer.Enabled {
		go k8s.NewConfigMapOperator().StartMonitor(stopCh)
	}

	go k8s.NewPodAgent().StartMonitor(stopCh)


	k8s.KubeInformerFactory.Start(stopCh)

	return registerServer.PrepareRun().Run(stopCh)
}
