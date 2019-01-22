package service

import (
	"encoding/json"
	"errors"
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/choerodon/go-register-server/pkg/api/metrics"
	"github.com/choerodon/go-register-server/pkg/api/repository"
	"github.com/choerodon/go-register-server/pkg/embed"
	"github.com/choerodon/go-register-server/pkg/k8s"
	"github.com/choerodon/go-register-server/pkg/utils"
	"github.com/emicklei/go-restful"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/go-playground/validator.v9"
	"reflect"
)

type ConfigService struct {
	Validate *validator.Validate
	appRepo  *repository.ApplicationRepository
}

func NewConfigService(appRepo *repository.ApplicationRepository) *ConfigService {
	s := &ConfigService{
		Validate: validator.New(),
		appRepo:  appRepo,
	}
	return s
}

func (es *ConfigService) Create(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	dto := new(entity.CreateConfigDTO)
	err := request.ReadEntity(&dto)
	if err != nil {
		glog.Warningf("Create config readEntity error", err)
		_ = response.WriteErrorString(400, "invalid create configMap dto")
		return
	}
	err = es.Validate.Struct(dto)
	if err != nil {
		glog.Warningf("Create config invalid dto", err)
		_ = response.WriteErrorString(400, "invalid create configMap dto")
		return
	}

	source := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(dto.Yaml), &source)
	if err != nil {
		glog.Warningf("Create config invalid yaml", err)
		_ = response.WriteErrorString(400, "invalid yaml")
		return
	}

	if dto.Service == entity.ApiGatewayServiceName {
		err = es.createZuulRoutes(source, dto)
		if err != nil {
			glog.Warningf("Create config failed when create zuul route", err)
			_ = response.WriteErrorString(500, "create zuul-route configMap failed")
			return
		}
	}
	_, err = k8s.RegisterK8sClient.CreateOrUpdateConfigMap(dto)
	if err != nil {
		glog.Warningf("Create failed when operator configMap", err)
		_ = response.WriteErrorString(500, "create configMap failed")
	}

}

func (es *ConfigService) createZuulRoutes(gatewayConfig map[string]interface{}, gatewayDTO *entity.CreateConfigDTO) error {
	gatewayWithoutRouteMap, routeMap := separateRoute(gatewayConfig)
	routeBytes, err := yaml.Marshal(routeMap)
	if err != nil {
		return err
	}
	_, err = k8s.RegisterK8sClient.CreateOrUpdateConfigMap(&entity.CreateConfigDTO{
		Service:   entity.RouteConfigMap,
		Version:   gatewayDTO.Version,
		Profile:   entity.DefaultProfile,
		Namespace: gatewayDTO.Namespace,
		Yaml:      string(routeBytes),
	})
	if err != nil {
		return err
	}
	gatewayWithoutRouteMapBytes, err := yaml.Marshal(gatewayWithoutRouteMap)
	if err != nil {
		return err
	}
	gatewayDTO.Yaml = string(gatewayWithoutRouteMapBytes)
	return nil
}

func separateRoute(gatewayConfig map[string]interface{}) (map[string]interface{}, map[string]interface{}) {
	routeMap := make(map[string]interface{})
	for k, v := range gatewayConfig {
		if k == "zuul" && reflect.TypeOf(v).Kind() == reflect.Map {
			vm := v.(map[string]interface{})
			for rk, rv := range vm {
				if rk == "routes" {
					routeMap[rk] = rv
					delete(vm, rk)
				}
			}
		}
	}
	return gatewayConfig, map[string]interface{}{"zuul": routeMap}
}

func (es *ConfigService) Poll(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
	service := request.PathParameter("service")
	if service == "" {
		_ = response.WriteErrorString(400, "service is empty")
		return
	}
	version := request.PathParameter("version")
	if version == "" {
		_ = response.WriteErrorString(400, "version is empty")
		return
	}
	kvMap, configMapVersion, err := es.getConfigFromConfigMap(service, version)
	if err != nil {
		_ = response.WriteErrorString(404, "can't find correct configMap")
		glog.Warningf("Get config from configMap failed, service: %s", service, err)
		return
	}
	if isGateway(service) {
		routeMap, _, err := es.getConfigFromConfigMap(entity.RouteConfigMap, version)
		if err != nil {
			_ = response.WriteErrorString(404, "can't find zuul-route configMap")
			glog.Warningf("Get zuul-route from configMap failed", err)
			return
		}
		for k, v := range routeMap {
			kvMap[k] = v
		}
	}
	es.appendAddition(kvMap)
	env := &entity.Environment{
		Name:            service,
		Version:         configMapVersion,
		Profiles:        []string{version},
		PropertySources: []entity.PropertySource{{Name: service + "-" + version + "-" + configMapVersion, Source: kvMap}},
	}
	printConfig, _ := json.MarshalIndent(kvMap, "", "  ")
	glog.Infof("%s-%v pull config: %s", service, version, printConfig)
	err = response.WriteAsJson(env)
	if err != nil {
		glog.Warningf("GetConfig write apps.Environment as json error,  msg : %s", env, err)
	}
}

func (es *ConfigService) appendAddition(kvMap map[string]interface{}) {
	for k, v := range entity.ConfigServerAdditions {
		kvMap[k] = v
	}
}

func (es *ConfigService) getConfigFromConfigMap(service string, version string) (map[string]interface{}, string, error) {
	source := make(map[string]interface{})
	configMap := k8s.RegisterK8sClient.GetConfigMapByName(service)
	if configMap == nil {
		return nil, "", errors.New("can't find configMap")
	}
	application := "application"
	if version != entity.DefaultProfile {
		application += "-" + version
	}
	application += ".yml"
	yamlString := configMap.Data[application]
	if yamlString != "" {
		err := yaml.Unmarshal([]byte(yamlString), &source)
		if err != nil {
			return nil, "", err
		}
	}

	return utils.ConvertRecursiveMapToSingleMap(source), configMap.Annotations[entity.ChoerodonVersion], nil
}

func isGateway(service string) bool {
	for _, v := range embed.Env.ConfigServer.GatewayNames {
		if v == service {
			return true
		}
	}
	return false
}
