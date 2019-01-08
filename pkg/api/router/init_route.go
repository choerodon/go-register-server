package router

import (
	"github.com/choerodon/go-register-server/pkg/api/repository"
)

func InitRouters(appRepo *repository.ApplicationRepository) error {

	eurekaAppsService := newRegisterService(appRepo)
	eurekaAppsService.Register()

	return nil
}
