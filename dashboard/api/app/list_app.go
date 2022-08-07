package app

import (
	"github.com/gin-gonic/gin"

	"github.com/plusplus1/sentinel-go-ext/dashboard/api/base"
)

type appResp struct {
	Code int         `json:"code"`
	Msg  string      `json:"message,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func ListApps(c *gin.Context) {
	var items = base.ListAll()
	c.JSON(200, appResp{Data: items})
}
