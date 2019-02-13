package server

import (
	"github.com/choerodon/go-register-server/pkg/api/router"
	"net/http"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func (s *PreparedRegisterServer) Run(stopCh <-chan struct{}) error {

	router.Register()

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		glog.Fatal(http.ListenAndServe(":8000", nil))
	}()

	glog.Info("Started server")
	<-stopCh
	glog.Info("Shutting down server")

	return nil
}
