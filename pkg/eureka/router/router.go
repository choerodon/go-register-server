package router

import (
	"github.com/choerodon/go-register-server/pkg/eureka/repository"
)

func InitRouters(appRepo *repository.ApplicationRepository) error {

	eurekaAppsService := NewEurekaAppsService(appRepo)
	eurekaAppsService.Register()

	return nil
}
