package server

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/choerodon/go-register-server/pkg/api/repository"
	"github.com/choerodon/go-register-server/pkg/api/router"
)

type RegisterServer struct {
	Config *RegisterServerOptions
}

type PreparedRegisterServer struct {
	*RegisterServer
}

func CreateRegisterServer(options *RegisterServerOptions) *RegisterServer {
	s := &RegisterServer{
		Config: options,
	}

	return s
}

func (s *RegisterServer) PrepareRun() *PreparedRegisterServer {
	return &PreparedRegisterServer{s}
}

func (s *PreparedRegisterServer) Run(appRepo *repository.ApplicationRepository, stopCh <-chan struct{}) error {

	if err := router.InitRouters(appRepo); err != nil {
		return err
	}

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		glog.Fatal(http.ListenAndServe(":9000", nil))
	}()

	glog.Info("Started server")
	<-stopCh
	glog.Info("Shutting down server")

	return nil
}
