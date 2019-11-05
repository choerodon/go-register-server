package service

import (
	"encoding/json"
	"fmt"
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/choerodon/go-register-server/pkg/api/metrics"
	"github.com/choerodon/go-register-server/pkg/api/repository"
	"github.com/choerodon/go-register-server/pkg/embed"
	"github.com/choerodon/go-register-server/pkg/k8s"
	"github.com/choerodon/go-register-server/pkg/utils"
	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"strings"
	"time"
)

type EurekaServerService interface {
	Apps(request *restful.Request, response *restful.Response)
	AppsDelta(request *restful.Request, response *restful.Response)
	Register(request *restful.Request, response *restful.Response)
	Delete(request *restful.Request, response *restful.Response)
	Renew(request *restful.Request, response *restful.Response)
	UpdateMateData(request *restful.Request, response *restful.Response)
}
type EurekaServerServiceImpl struct {
	appRepo           *repository.ApplicationRepository
	configMapOperator k8s.ConfigMapOperator
	podOperator       k8s.PodOperatorInterface
}

func NewEurekaServerServiceImpl(appRepo *repository.ApplicationRepository) *EurekaServerServiceImpl {
	s := &EurekaServerServiceImpl{
		appRepo:           appRepo,
		configMapOperator: k8s.NewConfigMapOperator(),
		podOperator:       k8s.NewPodAgent(),
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

func (es *EurekaServerServiceImpl) Renew(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()
}

func (es *EurekaServerServiceImpl) Register(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()

	// 获取 instance
	instance := new(entity.Instance)
	err := request.ReadEntity(&instance)
	if err != nil {
		glog.Warningf("Register app failed when readEntity", err)
		_ = response.WriteErrorString(http.StatusBadRequest, "invalid entity Instance")
		return
	}
	// 过滤无效请求
	if instance.Port.Port == 0 {
		return
	}

	// 获取 service name
	appName := request.PathParameter("app-name")
	instance.App = appName

	// 生成并设置 instance id
	instance.InstanceId = fmt.Sprintf("%s:%s:%d", instance.IPAddr, instance.App, instance.Port.Port)

	// 保存 instance
	if err := es.RegisterCustomApp(instance); err != nil {
		_ = response.WriteErrorString(http.StatusInternalServerError,
			fmt.Sprintf("Register Instance Error: %s", err.Error()))
		return
	}

	// 设置响应码
	response.WriteHeader(http.StatusNoContent)

	glog.Info("Receive registry from ", request.PathParameter("app-name"))
}

func (es *EurekaServerServiceImpl) RegisterCustomApp(instance *entity.Instance) error {
	now := uint64(time.Now().Unix())
	var instanceFlag *entity.Instance
	// 获取是否存储过该实例
	if customInstance, ok := es.appRepo.CustomInstanceStore.Load(instance.InstanceId); ok {
		// 若有存储，则只执行更新
		if old, ok := customInstance.(*entity.Instance); ok {
			instance.LastUpdatedTimestamp = now
			utils.ImpInstance(instance, old)
			instanceFlag = old
			es.appRepo.CustomInstanceStore.Store(old.InstanceId, old)
		}
	} else {
		// 查找pod生成的事例中是否有该实例
		if podInstance, ok := es.appRepo.InstanceStore.Load(instance.InstanceId); ok {
			if old, ok := podInstance.(*entity.Instance); ok {
				instance.LastUpdatedTimestamp = now
				utils.ImpInstance(instance, old)
				instanceFlag = old
				es.appRepo.CustomInstanceStore.Store(old.InstanceId, old)
			}
		} else {
			// 若没有存储过，则初始化改实例并保存
			instance.LeaseInfo.RegistrationTimestamp = now
			utils.ImpInstance(instance, nil)
			es.appRepo.NamespaceStore.Store(
				fmt.Sprintf("%s/%s", utils.CUSTOM_APP_PREFIX, instance.InstanceId),
				instance.InstanceId,
			)
			instanceFlag = instance
			es.appRepo.CustomInstanceStore.Store(instance.InstanceId, instance)
		}
	}
	return es.StorageCustomAppToConfigMap(instanceFlag)
}

func (es *EurekaServerServiceImpl) StorageCustomAppToConfigMap(instance *entity.Instance) error {
	// 获取cm
	registerConfigMapClient := k8s.KubeClient.CoreV1().ConfigMaps(embed.Env.RegisterServerNamespace)
	configMap, err := registerConfigMapClient.Get(entity.RegisterServerName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// 提取数据
	instances := configMap.Data
	if instances == nil {
		instances = make(map[string]string, 5)
	}
	// instance转换为json
	bytes, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	// 保存instance至cm
	instances[strings.ReplaceAll(instance.InstanceId, ":", "-")] = string(bytes)
	configMap.Data = instances
	_, err = registerConfigMapClient.Update(configMap)
	if err != nil {
		return err
	}
	return nil
}

func (es *EurekaServerServiceImpl) Delete(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()

	// 获取instance Id
	instanceId := request.PathParameter("instance-id")
	// 从内存中删除instance
	es.appRepo.DeleteInstance(fmt.Sprintf("%s/%s", utils.CUSTOM_APP_PREFIX, instanceId))
	// 从cm中删除instance
	registerConfigMapClient := k8s.KubeClient.CoreV1().ConfigMaps(embed.Env.RegisterServerNamespace)
	configMap, err := registerConfigMapClient.Get(entity.RegisterServerName, metav1.GetOptions{})
	if err != nil {
		_ = response.WriteErrorString(http.StatusInternalServerError,
			fmt.Sprintf("Delete Instance Error: %s", err.Error()))
		return
	}
	delete(configMap.Data, strings.ReplaceAll(instanceId, ":", "-"))
	// 更新cm
	_, err = registerConfigMapClient.Update(configMap)
	if err != nil {
		_ = response.WriteErrorString(http.StatusInternalServerError,
			fmt.Sprintf("Delete Instance Error: %s", err.Error()))
		return
	}
}

func (es *EurekaServerServiceImpl) RegisterCustomAppFromConfigMap() {
	// 获取自定义app列表cm
	registerConfigMapClient := k8s.KubeClient.CoreV1().ConfigMaps(embed.Env.RegisterServerNamespace)
	configMap, e := registerConfigMapClient.Get(entity.RegisterServerName, metav1.GetOptions{})
	if e == nil {
		removeList := make([]string, 3)
		// 取出自定义列表数据
		for key, instanceJson := range configMap.Data {
			var instance entity.Instance
			e = json.Unmarshal([]byte(instanceJson), &instance)
			if e != nil {
				glog.Infof("Unmarshal register server config map of instancesJson error: %+v %s", e, key)
				return
			}
			// 判断当前环境中是否存在该pod实例
			// 不存在则标记为删除
			if podSelfLink, ok := instance.Metadata["podSelfLink"]; ok {
				namespace, name, err := cache.SplitMetaNamespaceKey(podSelfLink)
				if err != nil {
					removeList = append(removeList, key)
					break
				} else {
					_, err = k8s.KubeClient.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
					if errors.IsNotFound(err) {
						removeList = append(removeList, key)
						break
					}
				}
			}
			es.RegisterCustomApp(&instance)
		}

		if len(removeList) > 0 {
			for _, k := range removeList {
				delete(configMap.Data, k)
			}
			registerConfigMapClient.Update(configMap)
		}
	} else {
		// 若没有相应cm，则创建
		if errors.IsNotFound(e) {
			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: embed.Env.RegisterServerNamespace,
					Name:      entity.RegisterServerName,
				},
			}
			_, e = registerConfigMapClient.Create(cm)
			if e != nil {
				glog.Infof("create register server config map error: %+v", e)
			}
		} else {
			glog.Infof("get register server config map error: %+v", e)
		}
	}
}

func (es *EurekaServerServiceImpl) UpdateMateData(request *restful.Request, response *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": request.Request.RequestURI}).Inc()

	// 获取 instance
	mateData := make(map[string]map[string]string, 3)
	err := request.ReadEntity(&mateData)
	if err != nil {
		glog.Warningf("invalid entity instance matedata", err)
		_ = response.WriteErrorString(http.StatusBadRequest, "invalid entity instance matedata")
		return
	}

	// 获取自定义app列表cm
	registerConfigMapClient := k8s.KubeClient.CoreV1().ConfigMaps(embed.Env.RegisterServerNamespace)
	configMap, err := registerConfigMapClient.Get(entity.RegisterServerName, metav1.GetOptions{})
	if err != nil {
		glog.Warningf("Get configmap err %s", err.Error())
		_ = response.WriteErrorString(http.StatusBadRequest, fmt.Sprintf("Get configmap err: %s", err.Error()))
		return
	}

	for instanceId, instanceMateData := range mateData {
		if i, ok := es.appRepo.CustomInstanceStore.Load(instanceId); ok {
			instance := i.(*entity.Instance)
			for key, value := range instanceMateData {
				if len(value) == 0 {
					delete(instance.Metadata, key)
				} else {
					instance.Metadata[key] = value
				}
			}

			// instance转换为json
			bytes, err := json.Marshal(instance)
			if err != nil {
				glog.Warningf("Instance to json err %s", err.Error())
				_ = response.WriteErrorString(http.StatusBadRequest,
					fmt.Sprintf("Instance to json err: %s", err.Error()))
				return
			}
			configMap.Data[strings.ReplaceAll(instanceId, ":", "-")] = string(bytes)
		} else {
			if i, ok := es.appRepo.InstanceStore.Load(instanceId); ok {
				instance := i.(*entity.Instance)
				for key, value := range instanceMateData {
					if len(value) == 0 {
						delete(instance.Metadata, key)
					} else {
						instance.Metadata[key] = value
					}
				}
				// instance转换为json
				bytes, err := json.Marshal(instance)
				if err != nil {
					glog.Warningf("Instance to json err %s", err.Error())
					_ = response.WriteErrorString(http.StatusBadRequest,
						fmt.Sprintf("Instance to json err: %s", err.Error()))
					return
				}
				configMap.Data[strings.ReplaceAll(instanceId, ":", "-")] = string(bytes)
			}
		}
		registerConfigMapClient.Update(configMap)
	}
}
