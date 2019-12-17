package k8s

import (
	"encoding/json"
	"fmt"
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/choerodon/go-register-server/pkg/api/repository"
	"github.com/choerodon/go-register-server/pkg/embed"
	"github.com/choerodon/go-register-server/pkg/utils"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listerV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ConfigMapClient *ConfigMapOperatorImpl

type ConfigMapOperator interface {
	QueryConfigMapByName(name string) *v1.ConfigMap
	QueryConfigMapAndNamespaceByName(name string) (*v1.ConfigMap, string)
	CreateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error)
	UpdateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error)
	QueryConfigMap(name string, namespace string) *v1.ConfigMap
	StartMonitor(stopCh <-chan struct{})
}

type ConfigMapOperatorImpl struct {
	queue workqueue.RateLimitingInterface
	// workerLoopPeriod is the time between worker runs. The workers process the queue of configMap and pod changes.
	workerLoopPeriod time.Duration
	lister           listerV1.ConfigMapLister
	configMapsSynced cache.InformerSynced
	configMapCache   *sync.Map
	kubeV1Client     coreV1.CoreV1Interface
	notify           chan string
	appRepo          *repository.ApplicationRepository
	appNamespace     *sync.Map
}

func NewConfigMapOperator() ConfigMapOperator {
	if ConfigMapClient != nil {
		return ConfigMapClient
	}
	configMapInformer := KubeInformerFactory.Core().V1().ConfigMaps()
	ConfigMapClient = &ConfigMapOperatorImpl{
		queue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "cofigmap"),
		workerLoopPeriod: time.Second,
		lister:           configMapInformer.Lister(),
		notify:           make(chan string, 50),
		appRepo:          AppRepo,
		appNamespace:     &sync.Map{},
		kubeV1Client:     KubeClient.CoreV1(),
		configMapCache:   &sync.Map{},
	}
	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ConfigMapClient.enqueueConfigMap,
		UpdateFunc: func(old, new interface{}) {
			newConfigMap := new.(*v1.ConfigMap)
			oldConfigMap := old.(*v1.ConfigMap)
			if newConfigMap.ResourceVersion == oldConfigMap.ResourceVersion {
				return
			}
			ConfigMapClient.enqueueConfigMap(new)
		},
		DeleteFunc: ConfigMapClient.enqueueConfigMap,
	})
	ConfigMapClient.configMapsSynced = configMapInformer.Informer().HasSynced
	return ConfigMapClient
}

func (c *ConfigMapOperatorImpl) StartMonitor(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()
	if ok := cache.WaitForCacheSync(stopCh, c.configMapsSynced); !ok {
		glog.Fatal("failed to wait for caches to sync")
	}
	glog.Info("Starting k8s configMap monitor")
	for i := 0; i < 3; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	glog.Info("Started k8s configMap monitor")
	go func() {
		for {
			if d, ok := <-c.notify; ok {
				glog.Infof("ConfigMap %s Changes detected", d)
				instances := make([]*entity.Instance, 0)
				if entity.RouteConfigMap == d {
					for _, gateway := range embed.Env.ConfigServer.GatewayNames {
						instances = append(instances, c.appRepo.GetInstancesByService(gateway)...)
					}
				} else {
					instances = c.appRepo.GetInstancesByService(d)
				}
				go c.notifyRefresh(instances)
			}
		}
	}()
	<-stopCh
	glog.V(1).Info("Shutting down k8s configMap monitor")
}
func (c *ConfigMapOperatorImpl) enqueueConfigMap(obj interface{}) {
	var key string
	var err error
	if key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

func (c *ConfigMapOperatorImpl) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *ConfigMapOperatorImpl) processNextWorkItem() bool {
	key, shutdown := c.queue.Get()

	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	forget, err := c.syncHandler(key.(string))
	if err == nil {
		if forget {
			c.queue.Forget(key)
		}
		return true
	}

	runtime.HandleError(fmt.Errorf("error syncing '%s': %s", key, err.Error()))
	c.queue.AddRateLimited(key)

	return true
}

func (c *ConfigMapOperatorImpl) syncHandler(key string) (bool, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return true, nil
	}

	if key == fmt.Sprintf("%s/%s",
		embed.Env.RegisterServerNamespace, entity.RegisterServerName) {
		updateInstance(c)
		return true, nil
	}

	if !embed.Env.IsRegisterServiceNamespace(namespace) {
		return true, nil
	}

	configMap, err := c.lister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			c.configMapCache.Delete(name)
			glog.Warningf("configMap '%s' in work queue no longer exists", key)
			return true, nil
		}
		return false, err
	}

	if configMap.Annotations[entity.ChoerodonFeature] == entity.ChoerodonFeatureConfig {
		sha, ok := c.configMapCache.Load(configMap.Name)
		newSha := utils.Sha256Map(configMap.Data)
		if ok {
			if sha != newSha {
				c.configMapCache.Store(name, newSha)
				c.notify <- name
			}
		} else {
			glog.Infof("configMap '%s' is being monitored", key)
			c.configMapCache.Store(name, newSha)
		}
	}

	return true, nil
}

func isMonitorNamespace(namespace string) bool {
	for _, ns := range embed.Env.RegisterServiceNamespace {
		if strings.Compare(ns, namespace) == 0 {
			return true
		}
	}
	return false
}

func (c *ConfigMapOperatorImpl) notifyRefresh(instance []*entity.Instance) {
	if len(instance) < 1 {
		return
	}
	for _, v := range instance {
		noticeUri := "http://" + v.IPAddr + ":" + strconv.Itoa(int(v.Port.Port))
		context := v.Metadata[entity.ChoerodonContextPathLabel]
		if context != "" {
			noticeUri = noticeUri + "/" + context
		}
		noticeUri += "/choerodon/config"
		req, _ := http.NewRequest("PUT", noticeUri, nil)
		req.Header.Add("Authorization", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9."+
			"eyJwYXNzd29yZCI6InVua25vd24gcGFzc3dvcmQiLCJ1c2VybmFtZSI6ImRlZmF1bHQiLCJhdXRob3"+
			"JpdGllcyI6W10sImFjY291bnROb25FeHBpcmVkIjp0cnVlLCJhY2NvdW50Tm9uTG9ja2VkIjp0cnVlL"+
			"CJjcmVkZW50aWFsc05vbkV4cGlyZWQiOnRydWUsImVuYWJsZWQiOnRydWUsInVzZXJJZCI6MCwiZW1h"+
			"aWwiOm51bGwsInRpbWVab25lIjoiQ1RUIiwibGFuZ3VhZ2UiOiJ6aF9DTiIsIm9yZ2FuaXphdGlvbklkI"+
			"joxLCJhZGRpdGlvbkluZm8iOm51bGwsImFkbWluIjpmYWxzZX0.Bw96KnS4ZRyEY-77zIetuObbqcu2LR7J03MqwPS6pLI")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			glog.Warningf("Notify instance %s refresh config failed, error: %s", v.InstanceId, err.Error())
		} else if 200 <= res.StatusCode && res.StatusCode < 300 {
			glog.Infof("Notify instance %s refresh config success", v.InstanceId)
		} else {
			glog.Warningf("Notify instance %s refresh config failed, statusCode: %d", v.InstanceId, res.StatusCode)
		}
	}
}

func (c *ConfigMapOperatorImpl) QueryConfigMapByName(name string) *v1.ConfigMap {
	if v, ok := c.appNamespace.Load(name); ok {
		configMap, err := c.kubeV1Client.ConfigMaps(v.(string)).Get(name, metaV1.GetOptions{})
		if err == nil {
			return configMap
		}
	}
	for _, namespace := range embed.Env.RegisterServiceNamespace {
		configMap, err := c.kubeV1Client.ConfigMaps(namespace).Get(name, metaV1.GetOptions{})
		if err == nil {
			c.appNamespace.Store(name, namespace)
			return configMap
		}
	}
	return nil
}

func (c *ConfigMapOperatorImpl) QueryConfigMapAndNamespaceByName(name string) (*v1.ConfigMap, string) {
	if v, ok := c.appNamespace.Load(name); ok {
		configMap, err := c.kubeV1Client.ConfigMaps(v.(string)).Get(name, metaV1.GetOptions{})
		if err == nil {
			return configMap, fmt.Sprintf("%v", v)
		}
	}
	for _, namespace := range embed.Env.RegisterServiceNamespace {
		configMap, err := c.kubeV1Client.ConfigMaps(namespace).Get(name, metaV1.GetOptions{})
		if err == nil {
			c.appNamespace.Store(name, namespace)
			return configMap, namespace
		}
	}
	return nil, ""
}

func (c *ConfigMapOperatorImpl) CreateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error) {
	configMapOperator := c.kubeV1Client.ConfigMaps(dto.Namespace)
	createConfigMap, err := configMapOperator.Create(newV1ConfigMap(dto))
	if err != nil {
		return nil, err
	}
	c.appNamespace.Store(dto.Service, dto.Namespace)
	glog.Infof("Create configMap: %s, namespace: %s success", dto.Service, dto.Namespace)
	return createConfigMap, nil
}

func (c *ConfigMapOperatorImpl) UpdateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error) {
	configMapOperator := c.kubeV1Client.ConfigMaps(dto.Namespace)
	updateConfigMap, err := configMapOperator.Update(newV1ConfigMap(dto))
	if err != nil {
		return nil, err
	}
	c.appNamespace.Store(dto.Service, dto.Namespace)
	glog.Infof("Update configMap: %s, namespace: %s success", dto.Service, dto.Namespace)
	return updateConfigMap, nil
}

func (c *ConfigMapOperatorImpl) QueryConfigMap(name string, namespace string) *v1.ConfigMap {
	configMapOperator := c.kubeV1Client.ConfigMaps(namespace)
	configMap, err := configMapOperator.Get(name, metaV1.GetOptions{})
	if err == nil && configMap != nil {
		return configMap
	}
	return nil
}

func newV1ConfigMap(dto *entity.SaveConfigDTO) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: dto.Namespace,
			Name:      dto.Service,
			Annotations: map[string]string{
				entity.ChoerodonService: dto.Service,
				entity.ChoerodonVersion: dto.Version,
				entity.ChoerodonFeature: entity.ChoerodonFeatureConfig,
			},
		},
		Data: map[string]string{utils.ConfigMapProfileKey(dto.Profile): dto.Yaml},
	}
}

func DeleteInstanceFromConfigMap(key string) {
	cmClient := KubeClient.CoreV1().ConfigMaps(embed.Env.RegisterServerNamespace)
	configMap, err := cmClient.Get(entity.RegisterServerName, metaV1.GetOptions{})
	if err == nil {
		delete(configMap.Data, strings.ReplaceAll(key, ":", "-"))
		_, err := cmClient.Update(configMap)
		if err != nil {
			glog.Error("%+v", err)
		}
	} else {
		glog.Error("%+v", err)
	}
}

func updateInstance(c *ConfigMapOperatorImpl) {
	configMap, err := c.kubeV1Client.
		ConfigMaps(embed.Env.RegisterServerNamespace).Get(entity.RegisterServerName, metaV1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			glog.Warningf("configMap '%s' in work queue no longer exists",
				fmt.Sprintf("%s/%s", embed.Env.RegisterServerNamespace, entity.RegisterServerName))
		}
		return
	}

	// 遍历查找被删除的instance
	deleteList := make([]string, 0)
	c.appRepo.CustomInstanceStore.Range(func(key, value interface{}) bool {
		instanceId := key.(string)
		if _, ok := configMap.Data[strings.ReplaceAll(instanceId, ":", "-")]; !ok {
			deleteList = append(deleteList, instanceId)
		}
		return true
	})
	// 从内存中删除
	for _, d := range deleteList {
		c.appRepo.CustomInstanceStore.Delete(d)
	}
	// 更新instance
	for key, value := range configMap.Data {
		instance := new(entity.Instance)
		e := json.Unmarshal([]byte(value), instance)
		if e != nil {
			glog.Infof("Unmarshal register server config map of instancesJson error: %+v %s", e, key)
			return
		}
		c.appRepo.CustomInstanceStore.Store(instance.InstanceId, instance)
	}
}
