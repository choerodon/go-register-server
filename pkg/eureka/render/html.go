package render

import (
	"github.com/choerodon/go-register-server/pkg/eureka/apps"
	"html/template"
)

func InstanceHtml(instances []*apps.Instance) template.HTML {
	html := ""
	for i := 0; i < len(instances); i++ {
		html = html + "<a href=\"" + instances[i].StatusPageUrl +
			"\" target=\"_blank\">" + instances[i].InstanceId + "</a>&#12288;"
	}
	return template.HTML(html)
}

