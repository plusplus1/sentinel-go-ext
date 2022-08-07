package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plusplus1/sentinel-go-ext/dashboard/api/app"
	"github.com/plusplus1/sentinel-go-ext/dashboard/config"
	"github.com/plusplus1/sentinel-go-ext/dashboard/dist"
)

func InstallApi(router gin.IRouter) {

	router.GET("/version", func(c *gin.Context) { c.String(200, string(dist.Version)) })
	router.GET("/favicon.ico", func(c *gin.Context) { c.Data(200, `image/x-icon`, dist.FavIco) })

	// set auth
	auth := gin.BasicAuth(config.AppSettings().Accounts)
	// set
	staticGroup := router.Group("/web", auth)
	{
		staticGroup.StaticFS("/", http.FS(dist.DistFiles))
		if eg := router.(*gin.Engine); eg != nil {
			eg.NoRoute(func(c *gin.Context) {
				c.Redirect(302, "/web/dist/home.html")
			})
		}
	}

	apiGroup := router.Group("/api/", auth)
	{
		apiGroup.GET("/app/list", app.ListApps)
		apiGroup.GET("/app/rule/flow/list", app.ListFlowRules)
		apiGroup.POST("/app/rule/flow/del", app.DeleteFlowRule)
		apiGroup.POST("/app/rule/flow/update", app.SaveOrUpdateFlowRule)
		apiGroup.GET("/app/rule/circuitbreaker/list", app.ListCircuitbreakerRules)
		apiGroup.POST("/app/rule/circuitbreaker/del", app.DeleteCircuitbreakerRule)
		apiGroup.POST("/app/rule/circuitbreaker/update", app.SaveOrUpdateCircuitbreakerRule)
	}

	//debug
	router.GET("/fs/", scanFs)

}

func scanFs(c *gin.Context) {

	var (
		ret     = gin.H{}
		items   []gin.H
		path, _ = c.GetQuery("p")
	)

	if path == `` {
		path = "/"
	}
	dirs, e := dist.DistFiles.ReadDir(path)
	for _, d := range dirs {
		items = append(items, gin.H{
			`name`:   d.Name(),
			`is_dir`: d.IsDir(),
			`type`:   d.Type(),
		})
	}
	ret[`items`] = items
	ret[`path`] = path
	if e != nil {
		ret[`error`] = e.Error()
	}

	c.JSONP(200, ret)
}
