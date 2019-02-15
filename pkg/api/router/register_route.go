package router

import (
	"github.com/choerodon/go-register-server/pkg/api/service"
	"github.com/choerodon/go-register-server/pkg/embed"
	"github.com/choerodon/go-register-server/pkg/k8s"
	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"net/http"
	"path"
)

func Register() {
	appRepo := k8s.AppRepo

	rs := service.NewEurekaServerServiceImpl(appRepo)

	ps := service.NewEurekaPageServiceImpl(appRepo)

	glog.Info("Register eureka app APIs")

	ws := new(restful.WebService)

	ws.Path("/").Produces(restful.MIME_JSON, restful.MIME_XML)

	// eureka注册信息首页
	ws.Route(ws.GET("").To(ps.HomePage).Doc("Get home page"))

	// eureka页面所需静态文件的服务器
	ws.Route(ws.GET("/static/{subpath:*}").To(staticFromPathParam))

	ws.Route(ws.GET("/static").To(staticFromQueryParam))

	// 获取eureka注册信息、模拟注册、心跳接口
	ws.Route(ws.GET("eureka/apps").To(rs.Apps).
		Doc("Get all apps")).Produces("application/json")

	ws.Route(ws.GET("eureka/apps/delta").To(rs.AppsDelta).
		Doc("Get all apps delta")).Produces("application/json")

	ws.Route(ws.POST("eureka/apps/{app-name}").To(rs.Register).
		Doc("Get a app").Produces("application/json").
		Param(ws.PathParameter("app-name", "app name").DataType("string")))

	ws.Route(ws.PUT("eureka/apps/{app-name}/{instance-id}").To(rs.Renew).
		Doc("renew").
		Param(ws.PathParameter("app-name", "app name").DataType("string")).
		Param(ws.PathParameter("instance-id", "instance id").DataType("string")))

	if embed.Env.ConfigServer.Enabled {
		cs := service.NewConfigServiceImpl(appRepo)
		// 拉取配置
		ws.Route(ws.GET("{service}/{version}").To(cs.Poll).
			Doc("Get config")).Produces("application/json")
		// 创建配置或者更新配置
		ws.Route(ws.POST("configs").To(cs.Save).
			Doc("Create a config").Produces("application/json"))

	}

	restful.Add(ws)
}

type Message struct {
	Text string
}

func staticFromPathParam(req *restful.Request, resp *restful.Response) {
	actual := path.Join("static", req.PathParameter("subpath"))
	http.ServeFile(
		resp.ResponseWriter,
		req.Request,
		actual)
}

func staticFromQueryParam(req *restful.Request, resp *restful.Response) {
	http.ServeFile(
		resp.ResponseWriter,
		req.Request,
		path.Join("static", req.QueryParameter("resource")))
}
