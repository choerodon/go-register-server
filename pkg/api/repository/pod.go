package repository

import (
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/golang/glog"
	"sync"
)

type ApplicationRepository struct {
	ApplicationStore    *sync.Map
	NamespaceStore      *sync.Map
	InstanceStore       *sync.Map
	CustomInstanceStore *sync.Map
}

func NewApplicationRepository() *ApplicationRepository {
	return &ApplicationRepository{
		ApplicationStore:    &sync.Map{},
		NamespaceStore:      &sync.Map{},
		InstanceStore:       &sync.Map{},
		CustomInstanceStore: &sync.Map{},
	}
}

func (appRepo *ApplicationRepository) Register(instance *entity.Instance, key string) bool {
	//appStore := appRepo.ApplicationStore

	if _, ok := appRepo.NamespaceStore.Load(key); ok {
		return false
	} else {
		appRepo.NamespaceStore.Store(key, instance.InstanceId)
	}
	appRepo.InstanceStore.Store(instance.InstanceId, instance)
	return true
}

func (appRepo *ApplicationRepository) DeleteInstance(key string) *entity.Instance {
	if value, ok := appRepo.NamespaceStore.Load(key); ok {
		appRepo.CustomInstanceStore.Delete(value)
		appRepo.NamespaceStore.Delete(key)
		if instance, ok := appRepo.InstanceStore.Load(value); ok {
			appRepo.InstanceStore.Delete(value)
			if instance != nil {
				glog.Infof("Delete instance by key %s", key)
				return instance.(*entity.Instance)
			}
		}
		glog.Infof(" instance by key %s not exist but namespace exist", key)
	} else {
		glog.Infof("Delete instance by key %s not exist", key)
	}
	return nil

}

func (appRepo *ApplicationRepository) GetApplicationResources() *entity.ApplicationResources {
	appResource := &entity.ApplicationResources{
		Applications: &entity.Applications{
			VersionsDelta:   2,
			AppsHashcode:    "app_hashcode",
			ApplicationList: make([]*entity.Application, 0, 10),
		},
	}
	appMap := make(map[string]*entity.Application)
	appRepo.InstanceStore.Range(func(key, value interface{}) bool {
		instance := value.(*entity.Instance)
		if appMap[instance.App] == nil {
			app := &entity.Application{
				Name:      instance.App,
				Instances: make([]*entity.Instance, 0, 10),
			}

			instanceId := key.(string)
			if customInstance, ok := appRepo.CustomInstanceStore.Load(instanceId); ok {
				custom := customInstance.(*entity.Instance)
				instance.Status = custom.Status
				for k, v := range custom.Metadata {
					instance.Metadata[k] = v
				}
			}

			app.Instances = append(app.Instances, instance)
			appMap[instance.App] = app
			appResource.Applications.ApplicationList = append(appResource.Applications.ApplicationList, app)
		} else {
			app := appMap[instance.App]
			app.Instances = append(app.Instances, instance)

		}
		return true
	})

	appRepo.CustomInstanceStore.Range(func(key, value interface{}) bool {
		instance := value.(*entity.Instance)
		if _, ok := appRepo.InstanceStore.Load(instance.InstanceId); ok {
			return true
		}

		if appMap[instance.App] == nil {
			app := &entity.Application{
				Name:      instance.App,
				Instances: make([]*entity.Instance, 0, 10),
			}
			app.Instances = append(app.Instances, instance)
			appMap[instance.App] = app
			appResource.Applications.ApplicationList = append(appResource.Applications.ApplicationList, app)
		} else {
			app := appMap[instance.App]
			app.Instances = append(app.Instances, instance)

		}
		return true
	})

	appStore := appRepo.ApplicationStore
	appStore.Range(func(key, value interface{}) bool {
		app := value.(*entity.Application)
		appResource.Applications.ApplicationList = append(appResource.Applications.ApplicationList, app)
		return true
	})
	return appResource

}

func (appRepo *ApplicationRepository) Renew(appName string, instanceId string) entity.Instance {
	if value, ok := appRepo.ApplicationStore.Load(appName); ok {
		app := value.(*entity.Application)
		return *app.Instances[0]
	}
	return entity.Instance{}
}

func (appRepo *ApplicationRepository) GetInstanceIpsByService(service string) []string {
	instances := make([]string, 0, 10)
	appRepo.InstanceStore.Range(func(key, value interface{}) bool {
		instance := value.(*entity.Instance)
		instanceId := key.(string)
		if customInstance, ok := appRepo.CustomInstanceStore.Load(instanceId); ok {
			custom := customInstance.(*entity.Instance)
			instance.Status = custom.Status
			for k, v := range custom.Metadata {
				instance.Metadata[k] = v
			}
		}
		if instance.App == service && instance.Status == "UP" {
			instances = append(instances, instance.HomePageUrl)
		}
		return true
	})
	appRepo.CustomInstanceStore.Range(func(key, value interface{}) bool {
		instanceId := key.(string)
		if _, ok := appRepo.InstanceStore.Load(instanceId); ok {
			return true
		}
		instance := value.(*entity.Instance)
		if instance.App == service && instance.Status == "UP" {
			instances = append(instances, instance.HomePageUrl)
		}
		return true
	})
	return instances
}

func (appRepo *ApplicationRepository) GetInstancesByService(service string) []*entity.Instance {
	instances := make([]*entity.Instance, 0)
	appRepo.InstanceStore.Range(func(key, value interface{}) bool {
		instance := value.(*entity.Instance)
		instanceId := key.(string)
		if customInstance, ok := appRepo.CustomInstanceStore.Load(instanceId); ok {
			custom := customInstance.(*entity.Instance)
			instance.Status = custom.Status
			for k, v := range custom.Metadata {
				instance.Metadata[k] = v
			}
		}
		if instance.App == service && instance.Status == "UP" {
			instances = append(instances, instance)
		}
		return true
	})
	appRepo.CustomInstanceStore.Range(func(key, value interface{}) bool {
		instanceId := key.(string)
		if _, ok := appRepo.InstanceStore.Load(instanceId); ok {
			return true
		}
		instance := value.(*entity.Instance)
		if instance.App == service && instance.Status == "UP" {
			instances = append(instances, instance)
		}
		return true
	})
	return instances
}
