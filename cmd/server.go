package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/choerodon/go-register-server/cmd/options"
	"github.com/choerodon/go-register-server/pkg/controller"
	"github.com/choerodon/go-register-server/pkg/eureka/apps"
	"github.com/choerodon/go-register-server/pkg/eureka/event"
	"github.com/choerodon/go-register-server/pkg/eureka/repository"
	"github.com/choerodon/go-register-server/pkg/eureka/server"
	"github.com/choerodon/go-register-server/pkg/signals"
)

func NewServerCommand() *cobra.Command {
	s := options.NewServerRunOptions()
	cmd := &cobra.Command{
		Use:  "register-server",
		Long: `Register Server`,
		Run: func(cmd *cobra.Command, args []string) {
			stopCh := signals.SetupSignalHandler()

			if err := Run(s, stopCh); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	s.AddFlag(cmd.Flags())

	return cmd
}

func Run(s *options.ServerRunOptions, stopCh <-chan struct{}) error {
	appRepo := repository.NewApplicationRepository()

	registerServer := server.CreateRegisterServer(s.RegisterServerOptions)

	cfg, err := clientcmd.BuildConfigFromFlags(s.MasterURL, s.KubeConfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)

	var instance = make(chan apps.Instance, 100)

	go event.NewEventSender(kubeClient, instance, stopCh)

	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)

	podController := controller.NewController(kubeClient, kubeInformerFactory, appRepo)

	go kubeInformerFactory.Start(stopCh)
	go podController.Run(instance, stopCh)

	return registerServer.PrepareRun().Run(appRepo, stopCh)
}
