package router

import (
	"encoding/json"
	"github.com/choerodon/go-register-server/pkg/api/render"
	"github.com/choerodon/go-register-server/pkg/convertor"
	"github.com/choerodon/go-register-server/pkg/k8s"
	"github.com/ghodss/yaml"
	"html/template"
	"net/http"
	"path"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/choerodon/go-register-server/pkg/api/apps"
	"github.com/choerodon/go-register-server/pkg/api/metrics"
	"github.com/choerodon/go-register-server/pkg/api/repository"
)

type RegisterService struct {
	appRepo *repository.ApplicationRepository
}

func newRegisterService(appRepo *repository.ApplicationRepository) *RegisterService {
	s := &RegisterService{
		appRepo: appRepo,
	}

	return s
}

func (es *RegisterService) Register() {
	glog.Info("Register eureka app APIs")

	ws := new(restful.WebService)

	ws.Path("/").Produces(restful.MIME_JSON, restful.MIME_XML)

	ws.Route(ws.GET("").To(es.home).Doc("Get home page"))

	ws.Route(ws.GET("/static/{subpath:*}").To(staticFromPathParam))

	ws.Route(ws.GET("/static").To(staticFromQueryParam))

	// GET /eureka/apps
	ws.Route(ws.GET("eureka/apps").To(es.listEurekaApps).
		Doc("Get all apps")).Produces("application/json")

	ws.Route(ws.GET("eureka/apps/delta").To(es.listEurekaAppsDelta).
		Doc("Get all apps delta")).Produces("application/json")

	ws.Route(ws.POST("eureka/apps/{app-name}").To(es.registerEurekaApp).
		Doc("get a user").Produces("application/json").
		Param(ws.PathParameter("app-name", "app name").DataType("string")))

	ws.Route(ws.PUT("eureka/apps/{app-name}/{instance-id}").To(es.renew).
		Doc("renew").
		Param(ws.PathParameter("app-name", "app name").DataType("string")).
		Param(ws.PathParameter("instance-id", "instance id").DataType("string")))

	ws.Route(ws.GET("{service}/{version}").To(es.getConfig).
		Doc("Get config")).Produces("application/json")

	restful.Add(ws)
}

func (es *RegisterService) listEurekaApps(request *restful.Request, response *restful.Response) {
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

func (es *RegisterService) home(req *restful.Request, resp *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": req.Request.RequestURI}).Inc()
	t := template.Must(template.ParseFiles("templates/api.html"))
	register, eurekaInstances := render.GetEurekaApplicationInfos(es.appRepo.GetApplicationResources().Applications.ApplicationList)
	err := t.Execute(resp.ResponseWriter, &apps.EurekaPage{
		GeneralInfo:        render.GetGeneralInfo(),
		InstanceInfo:       render.GetInstanceInfo(),
		CurrentTime:        time.Now(),
		AvailableRegisters: register,
		EurekaInstances:    eurekaInstances,
	})
	if err != nil {
		glog.Fatalf("Error Get Home Page: %s", err.Error())
	}
}

func (es *RegisterService) listEurekaAppsDelta(request *restful.Request, response *restful.Response) {
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

func (es *RegisterService) registerEurekaApp(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	glog.Info("Receive registry from ", request.PathParameter("app-name"))
}

func (es *RegisterService) renew(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
}

func (es *RegisterService) getConfig(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	service := request.PathParameter("service")
	if service == "" {
		return
	}
	version := request.PathParameter("version")
	if version == "" {
		return
	}
	application := "application"
	if version != "default" {
		application += "-" + version
	}
	application += ".yml"
	source := make(map[string]interface{})
	configMap := k8s.RegisterK8sClient.GetConfigMapByName(service)
	if configMap != nil {
		yamlString := configMap.Data[application]
		if yamlString != "" {
			err := yaml.Unmarshal([]byte(yamlString), &source)
			if err != nil {
				glog.Warningf("Parse yaml from configMap %s error,  msg : %s", service, err)
			}
		}
	}
	trueSourceMap := convertor.ConvertRecursiveMapToSingleMap(source)
	env := &apps.Environment{
		Name:            service,
		Version:         version,
		Profiles:        []string{version},
		PropertySources: []apps.PropertySource{{Name: service + "-" + version + "-null", Source: trueSourceMap}},
	}
	printConfig,_ := json.MarshalIndent(trueSourceMap, "", "  ")
	glog.Infof("%s-%v pull config: %s", service, version, printConfig)
	err := response.WriteAsJson(env)
	if err != nil {
		glog.Warningf("GetConfig write apps.Environment as json error,  msg : %s", env, err)
	}
}