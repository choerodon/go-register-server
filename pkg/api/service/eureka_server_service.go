package service

import (
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/choerodon/go-register-server/pkg/api/metrics"
	"github.com/choerodon/go-register-server/pkg/api/repository"
	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type EurekaServerService interface {
	Apps(request *restful.Request, response *restful.Response)
	AppsDelta(request *restful.Request, response *restful.Response)
	Register(request *restful.Request, response *restful.Response)
	Renew(request *restful.Request, response *restful.Response)
}
type EurekaServerServiceImpl struct {
	appRepo *repository.ApplicationRepository
}

func NewEurekaServerServiceImpl(appRepo *repository.ApplicationRepository) *EurekaServerServiceImpl {
	s := &EurekaServerServiceImpl{
		appRepo: appRepo,
	}
	return s
}

func (es *EurekaServerServiceImpl) Apps(request *restful.Request, response *restful.Response) {
	start := time.Now()

	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	applicationResources := es.appRepo.GetApplicationResources()
	_ = response.WriteAsJson(applicationResources)

	finish := time.Now()
	cost := finish.Sub(start).Nanoseconds()

	metrics.FetchProcessTime.Set(float64(cost))
}

func (es *EurekaServerServiceImpl) AppsDelta(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	applicationResources := &entity.ApplicationResources{
		Applications: &entity.Applications{
			VersionsDelta:   2,
			AppsHashcode:    "app_hashcode",
			ApplicationList: make([]*entity.Application, 0),
		},
	}
	_ = response.WriteAsJson(applicationResources)
}

func (es *EurekaServerServiceImpl) Register(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	glog.Info("Receive registry from ", request.PathParameter("app-name"))
}

func (es *EurekaServerServiceImpl) Renew(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
}
