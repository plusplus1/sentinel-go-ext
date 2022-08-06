package app

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/plusplus1/sentinel-go-ext/dashboard/api/base"
	"github.com/plusplus1/sentinel-go-ext/source/reg"
)

func ListCircuitbreakerRules(c *gin.Context) {
	var appId = c.Query(`app`)
	var res []string

	if s := c.Query(`res`); s != `` {
		res = strings.Split(c.Query(`res`), `,`)
	}

	var app, exist = base.FindApp(appId)
	if !exist {
		c.JSON(200, appResp{Status: 100, Msg: `App Not Found`})
		return
	}

	builder := reg.SourceBuilder(app.Type)
	if builder == nil {
		c.JSON(200, appResp{Status: 100, Msg: `Data Source Type not supported`})
		return
	}

	inst := builder(app)
	rules, err := inst.ListCircuitbreakerRules(res...)
	if err != nil {
		c.JSON(200, appResp{Status: 999, Msg: err.Error()})
		return
	}

	c.JSON(200, appResp{Data: rules})
}
