package router

import (
	"github.com/choerodon/go-register-server/pkg/eureka/monitor"
	"html/template"
	"net/http"
	"path"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/choerodon/go-register-server/pkg/eureka/apps"
	"github.com/choerodon/go-register-server/pkg/eureka/metrics"
	"github.com/choerodon/go-register-server/pkg/eureka/repository"
)

type EurekaAppsService struct {
	appRepo *repository.ApplicationRepository
}

func NewEurekaAppsService(appRepo *repository.ApplicationRepository) *EurekaAppsService {
	s := &EurekaAppsService{
		appRepo: appRepo,
	}

	return s
}

func (es *EurekaAppsService) Register() {
	glog.Info("Register eureka app APIs")

	ws := new(restful.WebService)
	ws.Path("").Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("/static/{subpath:*}").To(staticFromPathParam))
	ws.Route(ws.GET("/static").To(staticFromQueryParam))

	ws.Route(ws.GET("").To(es.home).
		Doc("Get home page")).Produces("text/html; charset=utf-8")

	//ws.Route(ws.GET("/lastn").To(es.lastn).
	//	Doc("Get home page")).Produces("text/html; charset=utf-8")

	ws.Route(ws.GET("/eureka/apps").To(es.listEurekaApps).
		Doc("Get all apps")).Produces("application/json")

	ws.Route(ws.GET("/eureka/apps/delta").To(es.listEurekaAppsDelta).
		Doc("Get all apps delta")).Produces("application/json")

	ws.Route(ws.POST("/eureka/apps/{app-name}").To(es.registerEurekaApp).
		Doc("get a user").Produces("application/json").
		Param(ws.PathParameter("app-name", "app name").DataType("string")))

	ws.Route(ws.PUT("/eureka/apps/{app-name}/{instance-id}").To(es.renew).
		Doc("renew").
		Param(ws.PathParameter("app-name", "app name").DataType("string")).
		Param(ws.PathParameter("instance-id", "instance id").DataType("string")))
	restful.Add(ws)
}

func (es *EurekaAppsService) listEurekaApps(request *restful.Request, response *restful.Response) {
	start := time.Now()

	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	applicationResources := es.appRepo.GetApplicationResources()
	response.WriteAsJson(applicationResources)

	finish := time.Now()
	cost := finish.Sub(start).Nanoseconds()

	metrics.FetchProcessTime.Set(float64(cost))
}

type Message struct {
	Text string
}

func staticFromPathParam(req *restful.Request, resp *restful.Response) {
	actual := path.Join("static", req.PathParameter("subpath"))
	http.ServeFile(
		resp.ResponseWriter,
		req.Request,
		actual)
}

func staticFromQueryParam(req *restful.Request, resp *restful.Response) {
	http.ServeFile(
		resp.ResponseWriter,
		req.Request,
		path.Join("static", req.QueryParameter("resource")))
}

func (es *EurekaAppsService) home(req *restful.Request, resp *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": req.Request.RequestURI}).Inc()
	t, _ := template.ParseFiles("templates/eureka.html")
	register, eurekaInstances := monitor.GetEurekaApplicationInfos(es.appRepo.GetApplicationResources().Applications.ApplicationList)
	t.Execute(resp.ResponseWriter, &apps.EurekaPage{
		GeneralInfo:        monitor.GetGeneralInfo(),
		InstanceInfo:       monitor.GetInstanceInfo(),
		CurrentTime:        time.Now(),
		AvailableRegisters: register,
		EurekaInstances:    eurekaInstances,
	})
}

//func (es *EurekaAppsService) lastn(req *restful.Request, resp *restful.Response) {
//	metrics.RequestCount.With(prometheus.Labels{"path": req.Request.RequestURI}).Inc()
//	t, _ := template.ParseFiles("templates/lastn.html")
//	t.Execute(resp.ResponseWriter, "Hello world")
//}

func (es *EurekaAppsService) listEurekaAppsDelta(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	applicationResources := &apps.ApplicationResources{
		Applications: &apps.Applications{
			VersionsDelta:   2,
			AppsHashcode:    "app_hashcode",
			ApplicationList: make([]*apps.Application, 0),
		},
	}
	response.WriteAsJson(applicationResources)
}

func (es *EurekaAppsService) registerEurekaApp(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	glog.Info("Receive registry from ", request.PathParameter("app-name"))
}

func (es *EurekaAppsService) renew(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
}
