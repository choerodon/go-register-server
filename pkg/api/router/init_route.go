package router

import (
	"github.com/choerodon/go-register-server/pkg/api/repository"
)

func InitRouters(appRepo *repository.ApplicationRepository) error {

	Register(appRepo)

	return nil
}
