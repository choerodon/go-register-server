package repository

import (
	"sync"

	"github.com/golang/glog"

	"github.com/choerodon/go-register-server/pkg/eureka/apps"
)

type ApplicationRepository struct {
	applicationStore *sync.Map
	namespaceStore   *sync.Map
	instanceStore    *sync.Map
}

func NewApplicationRepository() *ApplicationRepository {
	return &ApplicationRepository{
		applicationStore: &sync.Map{},
		namespaceStore:   &sync.Map{},
		instanceStore:    &sync.Map{},
	}
}

func (appRepo *ApplicationRepository) Register(instance *apps.Instance, key string) bool {
	//appStore := appRepo.applicationStore

	if _, ok := appRepo.namespaceStore.Load(key); ok {
		return false
	} else {
		appRepo.namespaceStore.Store(key, instance.InstanceId)
	}
	appRepo.instanceStore.Store(instance.InstanceId, instance)
	return true

	//var app *apps.Application
	//if value, ok := appStore.Load(instance.App); ok {
	//	app = value.(*apps.Application)
	//	app.Instances = append(app.Instances, instance)
	//} else {
	//	app = &apps.Application{
	//		Name:      instance.App,
	//		Instances: make([]*apps.Instance, 0, 10),
	//	}
	//	app.Instances = append(app.Instances, instance)
	//
	//}
	//appStore.Store(app.Name, app)
}

func (appRepo *ApplicationRepository) DeleteInstance(key string) *apps.Instance {
	if value, ok := appRepo.namespaceStore.Load(key); ok {
		instance, _ := appRepo.instanceStore.Load(value)
		appRepo.instanceStore.Delete(value)
		appRepo.namespaceStore.Delete(key)
		if instance != nil {
			glog.Infof("Delete instance by key %s", key)
			return instance.(*apps.Instance)
		}
		glog.Infof(" instance by key %s not exist but namespace exist", key)
	} else {
		glog.Infof("Delete instance by key %s not exist", key)
	}
	return nil

}

func (appRepo *ApplicationRepository) GetApplicationResources() *apps.ApplicationResources {
	appResource := &apps.ApplicationResources{
		Applications: &apps.Applications{
			VersionsDelta:   2,
			AppsHashcode:    "app_hashcode",
			ApplicationList: make([]*apps.Application, 0, 10),
		},
	}
	appMap := make(map[string]*apps.Application)
	appRepo.instanceStore.Range(func(instanceId, value interface{}) bool {
		instance := value.(*apps.Instance)
		if appMap[instance.App] == nil {
			app := &apps.Application{
				Name:      instance.App,
				Instances: make([]*apps.Instance, 0, 10),
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

	appStore := appRepo.applicationStore
	appStore.Range(func(key, value interface{}) bool {
		app := value.(*apps.Application)
		appResource.Applications.ApplicationList = append(appResource.Applications.ApplicationList, app)
		return true
	})
	return appResource

}

func (appRepo *ApplicationRepository) Renew(appName string, instanceId string) apps.Instance {
	if value, ok := appRepo.applicationStore.Load(appName); ok {
		app := value.(*apps.Application)
		return *app.Instances[0]
	}
	return apps.Instance{}
}
