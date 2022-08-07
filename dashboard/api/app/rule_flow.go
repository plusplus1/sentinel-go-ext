package app

import (
	"strings"

	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/gin-gonic/gin"

	"github.com/plusplus1/sentinel-go-ext/dashboard/api/base"
	"github.com/plusplus1/sentinel-go-ext/source/reg"
)

type flowArgType struct {
	flow.Rule

	AppId string `json:"appId"`
}

func ListFlowRules(c *gin.Context) {
	var appId = c.Query(`app`)
	var res []string

	if s := c.Query(`res`); s != `` {
		res = strings.Split(c.Query(`res`), `,`)
	}

	var app, exist = base.FindApp(appId)
	if !exist {
		c.JSON(200, appResp{Code: 100, Msg: `App Not Found`})
		return
	}

	builder := reg.SourceBuilder(app.Type)
	if builder == nil {
		c.JSON(200, appResp{Code: 100, Msg: `Data Source Type not supported`})
		return
	}

	inst := builder(app)
	rules, err := inst.ListFlowRules(res...)
	if err != nil {
		c.JSON(200, appResp{Code: 999, Msg: err.Error()})
		return
	}

	c.JSON(200, appResp{Data: rules})
}

func DeleteFlowRule(c *gin.Context) {
	arg := flowArgType{}
	_ = c.ShouldBindJSON(&arg)
	if arg.AppId == `` || arg.Resource == `` {
		c.JSON(200, appResp{Code: 100, Msg: `参数错误`})
		return
	}

	app, exist := base.FindApp(arg.AppId)
	if !exist {
		c.JSON(200, appResp{Code: 100, Msg: `找不到应用`})
		return
	}
	builder := reg.SourceBuilder(app.Type)
	if builder == nil {
		c.JSON(200, appResp{Code: 100, Msg: `该应用数据源类型不支持`})
		return
	}

	inst := builder(app)
	if err := inst.DeleteFlowRule(arg.Rule); err != nil {
		c.JSON(200, appResp{Code: 599, Msg: `删除失败!` + err.Error()})
		return
	}
	c.JSON(200, appResp{Code: 0, Msg: `删除成功`})
}

func SaveOrUpdateFlowRule(c *gin.Context) {
	arg := flowArgType{}
	_ = c.ShouldBindJSON(&arg)
	if arg.AppId == `` || arg.Resource == `` {
		c.JSON(200, appResp{Code: 100, Msg: `参数错误`})
		return
	}
	if arg.ID == `` {
		arg.ID = arg.Resource
	}

	app, exist := base.FindApp(arg.AppId)
	if !exist {
		c.JSON(200, appResp{Code: 100, Msg: `找不到应用`})
		return
	}
	builder := reg.SourceBuilder(app.Type)
	if builder == nil {
		c.JSON(200, appResp{Code: 100, Msg: `该应用数据源类型不支持`})
		return
	}

	inst := builder(app)
	if err := inst.SaveOrUpdateFlowRule(arg.Rule); err != nil {
		c.JSON(200, appResp{Code: 599, Msg: `更新失败!` + err.Error()})
		return
	}
	c.JSON(200, appResp{Code: 0, Msg: `更新成功`})

}
