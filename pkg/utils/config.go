package utils

import "github.com/choerodon/go-register-server/pkg/api/entity"

func ConfigMapProfileKey(profile string) string {
	application := "application"
	if profile != entity.DefaultProfile {
		application += "-" + profile
	}
	return application + ".yml"
}
